package handler

import (
	"fmt"
	"net/http"
	"regexp"
	"sync"

	"github.com/BluestNight/static-forms/errors"
)

var (
	// LabelHandlers is the label for the collection of form submission
	// handlers
	LabelHandlers = "handler"
	// LabelHandlerPath is the label for the path that a handler handles
	LabelHandlerPath = "path"
	// LabelAllowedDomain represents the domain a request is expected to come
	// from. Use "*" to represent all domains (dangerous).
	LabelAllowedDomain = "allowed_domain"
	// LabelHoneypot is the label for the honeypot input field. If the honeypot
	// has a value when the form is submitted, the form submission will be
	// discarded
	LabelHoneypot = "honeypot"
)

type regexpContext struct {
	Email *regexp.Regexp
}

type templateContext struct {
	Regexp regexpContext
}

// TemplateContext provides variables to be used inside templates
var TemplateContext = templateContext{
	Regexp: regexpContext{
		// Required reading: https://www.regular-expressions.info/email.html
		Email: regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._%+-]{0,63}@(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,62}[a-zA-Z0-9])?\.){1,8}[a-zA-Z]{2,63}$`)}}

// Handler represents anything that can handle a form submission
type Handler interface {
	Handle(req *http.Request, ch chan *errors.HTTPError, wg *sync.WaitGroup)
	AllowedDomain() string
	Honeypot() string
}

// FormValuesFunc generates a "FormValues" function that returns the full
// slice of values instead of just the first
func FormValuesFunc(req *http.Request) func(string) ([]string, error) {
	return func(name string) ([]string, error) {
		err := req.ParseForm()
		if err != nil {
			return nil, errors.NewHTTPError(
				fmt.Sprintf("Could not parse form: %s", err), 500)
		}
		return req.Form[name], nil
	}
}

// ErrorfFunc generates a function that wraps fmt.Errorf so that the error is
// returned as the second value, which tells the template that an error
// occurred and causes it to be returned. Also sets the given HTTPError pointer
// to the error that occurred.
func ErrorfFunc(err *errors.HTTPError) func(format string, v ...interface{}) (interface{}, error) {
	return func(format string, v ...interface{}) (interface{}, error) {
		*err = *errors.NewHTTPError(fmt.Sprintf(format, v...), 400)
		return nil, err
	}
}
