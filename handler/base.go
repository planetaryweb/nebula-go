package handler

import (
	"fmt"
	"github.com/hashicorp/go-hclog"
	"gitlab.com/BluestNight/nebula-forms/errors"
	pb "gitlab.com/BluestNight/nebula-forms/proto"
	"gitlab.com/Shadow53/interparser/parse"
	"net/http"
	"os"
	"sync"
)

// Base contains properties and methods common to all handlers.
// Base is not itself a Handler, though it may implement some of the
// required functions of one. It is intended to be anonymously included into
// another struct that provides the actual handling of the form.
type Base struct {
	Logger		     hclog.Logger
	origins          map[string]struct{}
	honeypot         string
	handleConditions map[string]*handleCondition
	name             string
	path			 string
}

func (h Base) Name() string {
	return h.name
}

func (h *Base) Unmarshal(data interface{}) error {
	// Parse interface as map
	d, err := parse.MapStringKeys(data)
	if err != nil {
		return fmt.Errorf(errors.ErrConfigItem, "handler", err)
	}

	// Parse the plugin name
	h.name, err = parse.String(d[LabelName])
	if err != nil {
		return fmt.Errorf(errors.ErrConfigItem, LabelName, err)
	}

	debug, err := parse.Bool(d[LabelDebugEnable])
	if err != nil {
		return fmt.Errorf(errors.ErrConfigItem, LabelDebugEnable, err)
	}

	var level hclog.Level
	if debug {
		level = hclog.Debug
	} else {
		level = hclog.Info
	}

	if d[LabelLogFile] != nil {
		file, err := parse.String(d[LabelLogFile])
		if err != nil {
			return fmt.Errorf(errors.ErrConfigItem, LabelLogFile, err)
		}

		f, err := os.OpenFile(file, os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf(errors.ErrConfigItem, LabelLogFile, err)
		}

		// Parse plugin logger
		h.Logger = hclog.New(&hclog.LoggerOptions{
			Name: h.name,
			Level: level,
			Output: f,
			Mutex: &sync.Mutex{},
		})
	}

	// Parse honeypot field, if exists
	h.honeypot, err = parse.StringOrDefault(d[LabelHoneypot], "")
	if err != nil {
		return fmt.Errorf(errors.ErrConfigItem, LabelHoneypot, err)
	}

	h.path, err = parse.String(d[LabelHandlerPath])
	if err != nil {
		return fmt.Errorf(errors.ErrConfigItem, LabelHandlerPath, err)
	}

	// Parse allowed origins
	origins, err := parse.Slice(d[LabelAllowedOrigins])
	if err != nil {
		return fmt.Errorf(
			errors.ErrConfigItem, LabelAllowedOrigins, err)
	}
	h.origins = make(map[string]struct{})

	for _, origin := range origins {
		o, err := parse.String(origin)
		if err != nil {
			return fmt.Errorf(
				errors.ErrConfigItem, LabelAllowedOrigins, err)
		}
		h.Logger.Debug("Registering origin", "origin", o, "handler", h.name)
		h.origins[o] = struct{}{}
	}

	// Parse handling conditions
	if d[LabelHandleIf] != nil {
		conditions, err := parse.MapStringKeys(d[LabelHandleIf])
		if err != nil {
			return fmt.Errorf(errors.ErrConfigItem, LabelHandleIf, err)
		}

		// Parse allowed value slices
		for key, val := range conditions {
			cond := handleCondition{}

			cond.MustBeNonEmpty, err = parse.Bool(val)
			if err != nil {
				s, err := parse.Slice(val)
				if err != nil {
					return fmt.Errorf(errors.ErrConfigItem,
						fmt.Sprintf("%s (%s)", LabelHandleIf, key), err)
				}

				// Make sure list of values is not empty
				if len(s) == 0 {
					return fmt.Errorf(errors.ErrConfigItem,
						fmt.Sprintf("%s (%s)", LabelHandleIf, key),
						"list of values must contain at least one value")
				}

				// Parse list for values
				for _, str := range s {
					allowedVal, err := parse.String(str)
					if err != nil {
						return fmt.Errorf(errors.ErrConfigItem,
							fmt.Sprintf("%s (%s)", LabelHandleIf, key), err)
					}
					// Initialize map if nil
					if cond.AllowedValues == nil {
						cond.AllowedValues = make(map[string]struct{})
					}
					cond.AllowedValues[allowedVal] = struct{}{}
				}
			}

			if h.handleConditions == nil {
				h.handleConditions = make(map[string]*handleCondition)
			}

			h.handleConditions[key] = &cond
		}
	}

	return nil
}

func (h Base) ShouldHandle(req *pb.HTTPRequest) (bool, *errors.HTTPError) {
	h.Logger.Debug("Testing if service should handle request")
	if req == nil {
		h.Logger.Error("Cannot process empty request")
		return false, errors.NewHTTPError("Cannot process empty request", http.StatusInternalServerError)
	}

	origins := req.Headers["Origin"]
	origin := ""
	if origins != nil && len(origins.All) > 0 {
		origin = origins.All[0]
	}

	if h.OriginAllowed(origin) {
		h.Logger.Debug("Origin is allowed", "origin", origin)
		if honeypot := req.Form[h.Honeypot()]; honeypot != nil {
			for _, val := range honeypot.Values {
				if f := val.GetFile(); f != nil {
					if f.FileName != "" || f.Size > 0 {
						h.Logger.Debug("Request fell into the honeypot")
						return false, nil
					}
				} else if v := val.GetStr(); v != "" {
					h.Logger.Debug("Request fell into the honeypot")
					return false, nil
				}
			}
		}

		for input, cond := range h.handleConditions {
			// This section assumes the field is not a file
			val := ""
			if req.Form[input] != nil {
				if len(req.Form[input].GetValues()) > 0{
					val = req.Form[input].GetValues()[0].GetStr()
				}
			}
			if cond.MustBeNonEmpty && val == "" {
				h.Logger.Debug("Required value was left empty", "input", input)
				return false, nil
			}

			if len(cond.AllowedValues) > 0 {
				vals := req.Form[input].GetValues()
				isAllowed := false
				if len(vals) > 0 {
					for _, val := range vals {
						str := val.GetStr()
						if _, ok := cond.AllowedValues[str]; ok {
							isAllowed = true
							break
						}
					}
				} else if _, ok := cond.AllowedValues[""]; ok {
					isAllowed = true
				}

				if !isAllowed {
					h.Logger.Debug("Value for input is not in allowed values",
						"input", input, "value", val, "allowed", hclog.Fmt("%#v", cond.AllowedValues))
					return false, nil
				}
			}
		}
		h.Logger.Debug("Origin %s is allowed", origin)
		return true, nil
	}
	h.Logger.Debug("Origin %s is not allowed", origin)
	return false, nil
}

// OriginAllowed returns whether the given origin is allowed to access
// this handler
func (h Base) OriginAllowed(origin string) bool {
	_, ok := h.origins[origin]
	if !ok {
		_, ok = h.origins["*"]
	}
	if ok {
		h.Logger.Debug("Origin is allowed", "origin", origin)
	} else {
		h.Logger.Debug("Origin is not allowed", "origin", origin)
	}
	return ok
}

// Honeypot returns the name of the form field that is the honeypot
// against spam bots
func (h Base) Honeypot() string {
	return h.honeypot
}
