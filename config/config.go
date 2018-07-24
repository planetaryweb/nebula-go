package config

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"sync"

	e "github.com/BluestNight/static-forms/errors"
	"github.com/BluestNight/static-forms/handler"
	"github.com/BluestNight/static-forms/handler/email"
	l "github.com/BluestNight/static-forms/log"
	"github.com/BurntSushi/toml"
	"github.com/Shadow53/interparser/parse"
	"github.com/fsnotify/fsnotify"
)

// Default values for configuration options go here

// DefaultMaxFileSize is the default value for maximum form upload body
const DefaultMaxFileSize = int64(5 * 1024 * 1024) // 5 MiB
// DefaultPort is the default port that the server will run at
const DefaultPort = int64(2002)

// labels contains the names of configuration options as found in the
// configuration file. This prevents unnoticed issues due to misspelling
// the key name.
var (
	// LabelPort is the label for the value containing the port the
	// server will run on.
	LabelPort = "port"
	// LabelErrorFile is the label for the path to place the error log file.
	LabelErrorFile = "error_file"
	// LabelLogFile is the label for the path to place the access log file.
	LabelLogFile = "log_file"
	// LabelMaxFileSize is the label for the maximum size of a single request.
	// This option is currently not respected, at least until I figure out how
	// to enforce it
	LabelMaxFileSize = "max_file_size"
)

// Config represents the parsed server configuration.
type Config struct {
	fWatcher    *fsnotify.Watcher
	Port        int64
	Logger      *l.Logger
	hMutex      sync.RWMutex
	handlers    map[string][]handler.Handler
	MaxFileSize int64
}

// WatchFile watches the given file as a configuration file and reloads the
// Config struct and the given server - if provided - on file changes.
// Spawns a goroutine that can be ended by calling Config.StopWatching or
// Config.StopWatchingAll.
func (c *Config) WatchFile(filename string, ch chan string) error {
	if c.fWatcher == nil {
		var err error
		c.fWatcher, err = fsnotify.NewWatcher()
		if err != nil {
			return fmt.Errorf("error creating file watcher: %s", err)
		}

		go func(ch chan string) {
			for {
				if c.fWatcher == nil {
					break
				}
				select {
				case event := <-c.fWatcher.Events:
					// Include rename because some editors use swap files,
					// which causes the rename op to be returned
					if event.Op&fsnotify.Write == fsnotify.Write ||
						event.Op&fsnotify.Rename == fsnotify.Rename {
						c.Logger.Logln("Detected file change")
						ch <- event.Name
					}
				case err := <-c.fWatcher.Errors:
					if err != nil {
						c.Logger.Errorf("Error while watching file: %s", err)
					} else {
						break
					}
				}
			}
		}(ch)
	}

	err := c.fWatcher.Add(filename)
	if err != nil {
		return fmt.Errorf("Error subscribing to %s: %s", filename, err)
	}

	return nil
}

// StopWatching stops the Config from watching a specific file, while
// continuing to watch any others that have been subscribed to. It does not
// close the file watcher or end the spawned goroutine for watching files.
// For that, use Config.StopWatchingAll.
func (c *Config) StopWatching(file string) error {
	return c.fWatcher.Remove(file)
}

// StopWatchingAll stops the Config from watching any of the subscribed files,
// closes the file watcher, and ends the goroutine that was watching the
// files.
func (c *Config) StopWatchingAll() error {
	err := c.fWatcher.Close()
	c.fWatcher = nil
	return err
}

// Unmarshal takes an interface{} representing the configuration and parses it
// to populate a Config struct
//
// The interface{} is expected to be a map[string]interface{}, such as those
// created when calling Unmarshal([]byte, interface) by passing a pointer to
// the interface as the second argument. This makes it easy to use
// configurations from multiple different kinds of configuration files,
// including JSON, YAML, and TOML, as well as programmatically creating them
// in a Go program using this as a library.
func (c *Config) Unmarshal(conf interface{}) error {
	data, err := parse.MapStringKeys(conf)
	if err != nil {
		return err
	}

	// If the interface was a nil map, error
	if data == nil {
		return errors.New("configuration map cannot be nil")
	}

	// We have a non-nil map - parse it for values

	// Max file size
	if data[LabelMaxFileSize] == nil {
		// Not specified, use default
		c.MaxFileSize = DefaultMaxFileSize
	} else {
		// Try to parse the value
		c.MaxFileSize, err = parse.Int64(data[LabelMaxFileSize])
		if err != nil {
			return err
		}

		// Check that parsed MaxFileSize is valid
		if c.MaxFileSize < 0 {
			return fmt.Errorf("%s must be non-negative", LabelMaxFileSize)
		}
	}

	// Port
	c.Port = parse.Int64OrDefault(data[LabelPort], DefaultPort)

	// Logger
	c.Logger = &l.Logger{}
	c.Logger.AddLogger(log.New(os.Stdout, "", log.LstdFlags))
	filename := parse.StringOrDefault(data[LabelLogFile], DefaultLogFile)
	// Create parent folder before creating log file
	err = os.MkdirAll(path.Dir(filename), 0700)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	c.Logger.AddLogger(log.New(file, "", log.LstdFlags))

	// Error logger
	c.Logger.AddErrorLogger(log.New(os.Stderr, "", log.LstdFlags))
	filename = parse.StringOrDefault(data[LabelErrorFile], DefaultErrorFile)
	err = os.MkdirAll(path.Dir(filename), 0700)
	if err != nil {
		return err
	}
	file, err = os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	c.Logger.AddErrorLogger(log.New(file, "error: ", log.LstdFlags))

	// Handlers
	// The bulk of the configuration is parsed here.
	// Email senders
	if data[email.LabelEmailSenders] != nil {
		emailSenders, err := parse.MapStringKeys(data[email.LabelEmailSenders])
		if err != nil {
			return fmt.Errorf(e.ErrBaseConfig, "email senders", err)
		}

		for key, val := range emailSenders {
			err = email.NewSender(key, val)
			if err != nil {
				return fmt.Errorf(e.ErrBaseConfig, "email senders", err)
			}
		}
	}

	// Actual handlers
	handlers, err := parse.MapStringKeys(data[handler.LabelHandlers])
	if err != nil {
		return fmt.Errorf(e.ErrBaseConfig, handler.LabelHandlers, err)
	}

	for hType, val := range handlers {
		hData, err := parse.MapStringKeys(val)
		if err != nil {
			return fmt.Errorf(e.ErrBaseConfig, hType+" handler", err)
		}

		for fName, form := range hData {

			f, err := parse.MapStringKeys(form)
			if err != nil {
				return fmt.Errorf(e.ErrBaseConfig, fName, err)
			}

			hPath, err := parse.String(f[handler.LabelHandlerPath])
			if err != nil {
				return fmt.Errorf(e.ErrBaseConfig, fName, err)
			}

			var h handler.Handler
			switch hType {
			default:
				return fmt.Errorf(e.ErrBaseConfig, fName,
					"invalid handler type: "+hType)
			case email.Type:
				h, err = email.NewHandler(f)
			}

			// Parsed the handler using one of the available parsers based on type
			// Handle errors, then save handler
			if err != nil {
				return fmt.Errorf(e.ErrBaseConfig, fName, err)
			}

			// Register handler under path
			c.AddHandler(hPath, h)
		}
	}

	return nil
}

// AddHandler adds a handler for a given handler path.
// Safe for parallel use.
func (c *Config) AddHandler(name string, h handler.Handler) {
	c.hMutex.Lock()
	if c.handlers == nil {
		c.handlers = make(map[string][]handler.Handler)
	}
	s := c.handlers[name]
	s = append(s, h)
	c.handlers[name] = s
	c.hMutex.Unlock()
}

// GetHandlers retrieves the list of handlers for a given handler path.
// Safe for parallel use.
func (c *Config) GetHandlers(name string) []handler.Handler {
	c.hMutex.RLock()
	defer c.hMutex.RUnlock()
	return c.handlers[name]
}

// ParseConfigFile parses the TOML file at the given path and returns a *Config
// upon successful parsing.
//
// For parsing other kinds of files, parse the configuration into an interface{}
// and use the Config.Unmarshal function instead.
func ParseConfigFile(filename string) (c *Config, err error) {
	var v interface{}
	_, err = toml.DecodeFile(filename, &v)
	if err != nil {
		return nil, err
	}

	c = &Config{}
	err = c.Unmarshal(v)
	return c, err
}

type handleFunc func(rw http.ResponseWriter, req *http.Request)

func getHandleFunc(path string, handlers []handler.Handler, l *l.Logger) handleFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		// Create a buffered channel large enough to fit responses from
		// all handlers
		ch := make(chan *e.HTTPError, len(handlers))
		var wg sync.WaitGroup
		// Keeping track of allowed domains for better logging
		var allowedDomains []string

		status := e.NewHTTPError("", http.StatusNotFound)
		// Run a goroutine for each handler
		for _, h := range handlers {
			// Checking like this because I may change the above status
			if status.Status() != http.StatusOK &&
				status.Status() != http.StatusForbidden {
				status = e.NewHTTPError(
					"This origin is not allowed to access this form endpoint",
					http.StatusForbidden)
			}

			if h.AllowedDomain() == "*" ||
				h.AllowedDomain() == req.Header.Get("Origin") {
				l.Logf(
					"Received form submission on path %s from origin %s\n",
					path, req.Header.Get("Origin"))
				wg.Add(1)
				go h.Handle(req, ch, &wg)
				status = e.NewHTTPError("", http.StatusOK)
			}
			allowedDomains = append(allowedDomains, h.AllowedDomain())
		}

		wg.Wait()
		close(ch)

		// Check that all goroutines "return" and no errors occurred
		// Possible codes returned, as far as I can find useful:
		// 200 - OK
		// 201 - Created
		// 202 - Accepted (still processing)
		// 204 - NoContent
		// 400 - BadRequest
		// 403 - Forbidden
		// 404 - NotFound
		// 405 - MethodNotAllowed
		// 500 - InternalServerError
		// 501 - NotImplemented
		// The following switch has the codes I expect to see used above or
		// by handlers, in order of least to greatest precedence.
		for err := range ch {
			if err != nil {
				l.Errorln(err)
				switch status.Status() {
				case http.StatusOK:
					status = err
				case http.StatusInternalServerError:
					// Assume that a client error may have caused the server
					// error
					if err.Status() >= 400 {
						status = err
					}
				case http.StatusNotFound:
					l.Errorf(
						"(404) Received an error from a handler that shouldn't have handled: %s",
						err.Error())
				case http.StatusForbidden:
					l.Errorf(
						"(403) Received an error from a handler that shouldn't have handled: %s",
						err.Error())
				case http.StatusMethodNotAllowed:
					if err.Status() >= 400 &&
						err.Status() < http.StatusMethodNotAllowed {
						status = err
					}
				case http.StatusBadRequest:
					// Have as greatest precedence because it should indicate
					// what the client did wrong
					status = err
				}
			}
		}

		l.Logln("Completed processing form submission")

		// If the source is allowed, add to response
		//if status.Status() != http.StatusForbidden {
		l.Logln("Setting CORS headers to match request")
		rw.Header().Set("Access-Control-Allow-Origin", req.Header.Get("Origin"))
		rw.Header().Set("Access-Control-Allow-Methods", "POST")
		//rw.Header().Set("Access-Control-Allow-Headers",
		//"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		rw.Header().Set("Vary", "Origin")
		/*} else {
			l.Logf(
				"Submission from %s to %s was not accepted: not in %v",
				req.Header.Get("Origin"), path, allowedDomains)
		}*/

		rw.WriteHeader(status.Status())
		if status.Status() >= 400 && status.Error() != "" {
			rw.Write([]byte(status.Error()))
		}
	}
}

// CreateServer generates an http.Server that handles the handlers found in
// this configuration struct
func (c *Config) CreateServer() *http.Server {
	mux := http.NewServeMux()

	for path, handlers := range c.handlers {
		mux.HandleFunc(path, getHandleFunc(path, handlers, c.Logger))
	}

	// Create ServeMux, now create Server
	s := &http.Server{
		Addr:    fmt.Sprintf(":%d", c.Port),
		Handler: mux}

	return s
}
