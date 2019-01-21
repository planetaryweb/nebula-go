package handler

import (
	"fmt"
	"net/http"

	"gitlab.com/BluestNight/nebula-forms/errors"
	"gitlab.com/BluestNight/nebula-forms/log"
	"github.com/Shadow53/interparser/parse"
)

// Base contains properties and methods common to all handlers.
// Base is not itself a Handler, though it may implement some of the
// required functions of one. It is intended to be anonymously included into
// another struct that provides the actual handling of the form.
type Base struct {
	origins          map[string]struct{}
	honeypot         string
	handleConditions map[string]*handleCondition
}

func (h *Base) Unmarshal(data interface{}) error {
	// Parse interface as map
	d, err := parse.MapStringKeys(data)
	if err != nil {
		return fmt.Errorf(errors.ErrConfigItem, "handler", err)
	}

	// Parse honeypot field, if exists
	h.honeypot, err = parse.StringOrDefault(d[LabelHoneypot], "")
	if err != nil {
		return fmt.Errorf(errors.ErrConfigItem, LabelHoneypot, err)
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

func (h Base) ShouldHandle(req *http.Request, l *log.Logger) (bool, error) {
	if h.OriginAllowed(req.Header.Get("Origin")) {
		l.Debugf("Connection from origin %s is allowed", req.Header.Get("Origin"))
		err := req.ParseForm()
		if err != nil {
			l.Errorf("Error while parsing form: %s", err)
			return false, err
		}

		if req.FormValue(h.Honeypot()) != "" {
			l.Debug("Request fell into the honeypot")
			return false, nil
		}

		for input, cond := range h.handleConditions {
			l.Debugf("Checking input %s for validity", input)
			if cond.MustBeNonEmpty && req.FormValue(input) == "" {
				l.Debugf("Input %s must not be empty but is anyways")
				return false, nil
			}

			if len(cond.AllowedValues) > 0 {
				vals := req.Form[input]
				isAllowed := false
				if len(vals) > 0 {
					for _, str := range vals {
						if _, ok := cond.AllowedValues[str]; ok {
							isAllowed = true
							break
						}
					}
				} else if _, ok := cond.AllowedValues[""]; ok {
					isAllowed = true
				}

				if !isAllowed {
					l.Debugf("Form value(s) is %#v, not one of %#v", vals, cond.AllowedValues)
					return false, nil
				}
			}
		}
		l.Debugf("Connection from %s is allowed", req.Header.Get("Origin"))
		return true, nil
	}
	l.Debugf("Origin %s is not in %#v", req.Header.Get("Origin"), h.origins)
	return false, nil
}

// OriginAllowed returns whether the given origin is allowed to access
// this handler
func (h Base) OriginAllowed(origin string) bool {
	_, ok := h.origins[origin]
	if !ok {
		_, ok = h.origins["*"]
	}
	return ok
}

// Honeypot returns the name of the form field that is the honeypot
// against spam bots
func (h Base) Honeypot() string {
	return h.honeypot
}
