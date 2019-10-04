package config

import (
	"errors"
	"fmt"
	"github.com/hashicorp/go-hclog"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/fsnotify/fsnotify"
	"github.com/hashicorp/go-plugin"
	e "gitlab.com/BluestNight/nebula-forms/errors"
	"gitlab.com/BluestNight/nebula-forms/handler"
	"gitlab.com/Shadow53/interparser/parse"
	"gitlab.com/Shadow53/merge-config/merge"
	"os"
	"path"
	"sync"
)

func (c *Config) unmarshalLoggers(data map[string]interface{}) error {
	filename, err := parse.StringOrDefault(data[LabelLogFile], DefaultLogFile)
	if err != nil {
		return fmt.Errorf(e.ErrConfigItem, LabelLogFile, err)
	}
	// Create parent folder before creating log file
	err = os.MkdirAll(path.Dir(filename), 0700)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	// Determine if debugging should be enabled
	debug, err := parse.BoolOrDefault(data[LabelDebugEnable], false)
	if err != nil {
		return err
	}

	var level hclog.Level
	if debug {
		level = hclog.Debug
	} else {
		level = hclog.Info
	}

	c.Logger = hclog.New(&hclog.LoggerOptions{
		Name: "nebula main",
		Output: file,
		Level: level,
		Mutex: &sync.Mutex{},
	})

	return nil
}

func (c *Config) unmarshalHandlers(data map[string]interface{}) error {
	// Actual handlers
	// Checking for nil because configuration files can be partial
	if data[handler.LabelHandlers] != nil {
		handlerMap, err := parse.MapStringKeys(data[handler.LabelHandlers])
		if err != nil {
			return fmt.Errorf(e.ErrConfigItem, handler.LabelHandlers, err)
		}

		if c.plugins == nil {
			c.plugins = make(map[string]handler.Handler)
		}

		if c.clients == nil {
			c.clients = make(map[string]*plugin.Client)
		}

		for name, conf := range handlerMap {
			d, err := parse.MapStringKeys(conf)
			if err != nil {
				return fmt.Errorf(e.ErrConfigItem, name, err)
			}

			protocol, err := parse.Int64OrDefault(d[handler.LabelProtocol], 1)
			if err != nil {
				return fmt.Errorf(e.ErrConfigItem, name,
					fmt.Errorf(e.ErrConfigItem, handler.LabelProtocol, err))
			}

			p, err := parse.String(d[handler.LabelHandlerPath])
			if err != nil {
				return fmt.Errorf(e.ErrConfigItem, name,
					fmt.Errorf(e.ErrConfigItem, handler.LabelHandlerPath, err))
			}

			cookieKey, err := parse.String(d[handler.LabelCookieKey])
			if err != nil {
				return fmt.Errorf(e.ErrConfigItem, name,
					fmt.Errorf(e.ErrConfigItem, handler.LabelCookieKey, err))
			}

			cookieVal, err := parse.String(d[handler.LabelCookieVal])
			if err != nil {
				return fmt.Errorf(e.ErrConfigItem, name,
					fmt.Errorf(e.ErrConfigItem, handler.LabelCookieVal, err))
			}

			commandInt, err := parse.Slice(d[handler.LabelCommand])
			if err != nil {
				return fmt.Errorf(e.ErrConfigItem, name,
					fmt.Errorf(e.ErrConfigItem, handler.LabelCommand, err))
			}

			var command []string
			for _, cmd := range commandInt {
				cmdStr, err := parse.String(cmd)
				if err != nil {
					return fmt.Errorf(e.ErrConfigItem, name,
						fmt.Errorf(e.ErrConfigItem, handler.LabelCommand, err))
				}

				command = append(command, cmdStr)
			}

			client := plugin.NewClient(&plugin.ClientConfig{
				HandshakeConfig: plugin.HandshakeConfig{
					ProtocolVersion:  uint(protocol),
					MagicCookieKey:   cookieKey,
					MagicCookieValue: cookieVal,
				},
				Plugins: map[string]plugin.Plugin{
					name: &handler.Plugin{},
				},
				Cmd:    exec.Command(command[0], command[1:]...),
				AllowedProtocols: []plugin.Protocol { plugin.ProtocolGRPC },
				Logger: c.Logger,
			})

			rpcClient, err := client.Client()
			if err != nil {
				return fmt.Errorf(e.ErrRPCStart, name, err)
			}

			raw, err := rpcClient.Dispense(name)
			if err != nil {
				return fmt.Errorf(e.ErrRPCStart, name, err)
			}

			c.AddHandler(p, raw.(handler.Handler))
			c.plugins[name] = raw.(handler.Handler)
			c.clients[name] = client
		}
	} else {
		return errors.New("at least one handler should be configured for this server")
	}
	return nil
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
func (c *Config) Unmarshal(conf interface{}) (err error) {
	// Prepare the *Config - i.e. reset
	c.handlers = make(map[string][]handler.Handler)
	c.hMutex = sync.RWMutex{}
	c.fWatcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	// Parse configuration as map
	data, err := parse.MapStringKeys(conf)
	if err != nil {
		return fmt.Errorf("could not unmarshal config as map: %s", err)
	}

	// If the interface was a nil map, error
	if data == nil {
		return errors.New("configuration map cannot be nil")
	}

	// We have a non-nil map - parse it for values

	// Try to parse the value
	c.MaxFileSize, err = parse.Int64OrDefault(data[LabelMaxFileSize], DefaultMaxFileSize)
	if err != nil {
		return fmt.Errorf(e.ErrConfigItem, LabelMaxFileSize, err)
	}

	// Check that parsed MaxFileSize is valid
	if c.MaxFileSize < 0 {
		return fmt.Errorf("%s must be non-negative", LabelMaxFileSize)
	}

	// Port
	c.Port, err = parse.Int64OrDefault(data[LabelPort], DefaultPort)
	if err != nil {
		return fmt.Errorf(e.ErrConfigItem, LabelPort, err)
	}

	if err = c.unmarshalLoggers(data); err != nil {
		return err
	}

	return c.unmarshalHandlers(data)
}

func (c *Config) tomlInner(filename string) (map[string]interface{}, []string, error) {
	var v interface{}
	var files []string

	_, err := toml.DecodeFile(filename, &v)
	if err != nil {
		return nil, nil, err
	}

	// Add to list of files
	files = append(files, filename)

	// If first file parsed, save
	if c.RootConfig == "" {
		c.RootConfig = filename
	}

	// Check for included directories and load TOML files in them too
	conf, err := parse.MapStringKeys(v)
	if err != nil {
		return nil, nil, fmt.Errorf("could not parse config for include_dir: %s", err)
	}

	// Get directory
	dir, err := parse.StringOrDefault(conf[LabelIncludeDir], "")
	if err != nil {
		return nil, nil, err
	}

	if dir != "" {
		// Make absolute if it is not already
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(filepath.Dir(filename), dir)
		}

		// Get file listing from inclde_dir
		included, err := ioutil.ReadDir(dir)
		if err != nil {
			return nil, nil, err
		}

		// Parse each file
		for _, file := range included {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".toml") {
				// Get file's path
				filename := filepath.Join(dir, file.Name())
				// Parse inner file's TOML
				innerConf, innerFiles, err := c.tomlInner(filename)
				if err != nil {
					return nil, nil, err
				}
				// Merge nested configuration with this one
				conf = merge.Merge(conf, innerConf)
				// Append current file and all files included in it
				files = append(files, innerFiles...)
			}
		}
	}

	return conf, files, nil
}

// ParseConfigTOMLFile parses the TOML file at the given path and returns a
// *Config upon successful parsing.
//
// For parsing other kinds of files, parse the configuration into an interface{}
// and use the Config.Unmarshal function instead.
func (c *Config) ParseConfigTOMLFile(filename string) ([]string, error) {
	// Parse files and merge
	conf, files, err := c.tomlInner(filename)
	if err != nil {
		return nil, err
	}

	// Full configuration is built, parse
	err = c.Unmarshal(conf)
	if err != nil {
		return nil, err
	}

	// Return read files
	return files, nil
}
