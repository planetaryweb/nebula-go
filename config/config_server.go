package config

import (
	"fmt"
	"github.com/hashicorp/go-hclog"
	"net/http"
	"sync"

	e "gitlab.com/BluestNight/nebula-forms/errors"
	"gitlab.com/BluestNight/nebula-forms/handler"
	pb "gitlab.com/BluestNight/nebula-forms/proto"
	"errors"
	"strings"
)

type handleFunc func(rw http.ResponseWriter, req *http.Request)

func parseForm(req *http.Request) error {
	var err error
	// HasPrefix because other information is added after the actual type
	if strings.HasPrefix(req.Header.Get("Content-Type"), "multipart/form-data") {
		err = req.ParseMultipartForm(http.DefaultMaxHeaderBytes)
	} else if strings.HasPrefix(req.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		err = req.ParseForm()
	} else {
		err = errors.New("Request body content type must be application/x-www-form-urlencoded or multipart/form-data")
	}

	return err
}

func getHandleFunc(path string, handlers []handler.Handler, l hclog.Logger) handleFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		// Create a buffered channel large enough to fit responses from
		// all handlers
		ch := make(chan *e.HTTPError, len(handlers))
		var wg sync.WaitGroup

		origin := req.Header.Get("Origin")
		l.Debug("Received request", "origin", origin)

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

			l.Debug("Converting request to protobuf structure")
			pReq, err := handler.RequestToProtoRequest(req)
			if err != nil {
				l.Error("Error while parsing request: %s", err)
				status = e.NewHTTPError(err.Error(), http.StatusInternalServerError)
			}

			l.Debug("Checking if handler should handle", "path", req.RequestURI)
			if ok, err := h.ShouldHandle(pReq); err == nil && ok {
				l.Debug(
					"Handler can handle this request", "path",
					req.RequestURI, "origin", origin)
				l.Info(
					"Received form submission", "path", path, "origin", origin)
				if req.Method == http.MethodPost {
					err := parseForm(req)
					if err == nil {
						wg.Add(1)
						go func(req *pb.HTTPRequest, ch chan *e.HTTPError, wg *sync.WaitGroup) {
							err := h.Handle(req)
							if err != nil {
								ch <- err
							}
							wg.Done()
						}(pReq, ch, &wg)
						// Return OK status even if honeypot is triggered
						// they might try again

						status = e.NewHTTPError("", http.StatusOK)
						l.Debug("Request has following form fields/entries", "form_debug", hclog.Fmt("%#v", req.Form))
					} else {
						l.Error("Error while parsing form", "path", req.RequestURI, "error", err)
						status = e.NewHTTPError(err.Error(), http.StatusInternalServerError)
					}
				} else {
					status = e.NewHTTPError("", http.StatusMethodNotAllowed)
				}
			} else if err != nil {
				l.Error("While determining if handler should handle", "path", req.RequestURI, "error", err)
				status = e.NewHTTPError(err.Error(),
					http.StatusInternalServerError)
			} else {
				l.Debug("Handler should not handle", "path", req.RequestURI, "origin", origin)
			}
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
				l.Error(err.Error())
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
					fallthrough
				case http.StatusForbidden:
					l.Error(http.StatusText(err.Status()), "status", err.Status(), "error",	err.Error())
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

		l.Info("Completed processing form submission")

		// If the source is allowed, add to response
		if status.Status() != http.StatusForbidden {
			l.Info("Setting CORS headers to match request")
			rw.Header().Set("Access-Control-Allow-Origin", origin)
			rw.Header().Set("Access-Control-Allow-Methods", "POST")
			//rw.Header().Set("Access-Control-Allow-Headers",
			//"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
			rw.Header().Set("Vary", "Origin")
		} else {
			l.Info(
				"Submission was not accepted", "path", path, "origin", origin)
		}

		l.Info("Request status", "status", status.Status(), "error", status.Error())

		rw.WriteHeader(status.Status())
		if status.Status() == http.StatusBadRequest && status.Error() != "" {
			rw.Write([]byte(status.Error()))
		} else if status.Status() >= 500 {
			rw.Write([]byte("A server error occurred. Please try again later."))
		}
	}
}

// CreateServer generates an http.Server that handles the handlers found in
// this configuration struct
func (c *Config) CreateServer() *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request){
		c.Logger.Error(
			"Received request to unhandled path", "path", req.RequestURI)
		rw.WriteHeader(http.StatusNotFound)
	})

	for path, handlers := range c.handlers {
		mux.HandleFunc(path, getHandleFunc(path, handlers, c.Logger))
	}

	// Create ServeMux, now create Server
	s := &http.Server{
		Addr:    fmt.Sprintf(":%d", c.Port),
		Handler: mux}

	return s
}
