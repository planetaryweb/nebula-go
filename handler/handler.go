package handler

import (
	"fmt"
	"regexp"

	"gitlab.com/BluestNight/nebula-forms/errors"
	pb "gitlab.com/BluestNight/nebula-forms/proto"
)

var (
	// LabelHandlers is the label for the collection of form submission
	// handlers
	LabelHandlers = "handler"
	// LabelHandlerPath is the label for the path that a handler handles
	LabelHandlerPath = "path"
	// LabelAllowedDomain represents the domain a request is expected to come
	// from. Use "*" to represent all domains (dangerous).
	LabelAllowedOrigins = "allowed_origins"
	// LabelErrorFile is the label for the path to place the error log file.
	LabelErrorFile = "error_file"
	// LabelLogFile is the label for the path to place the access log file.
	LabelLogFile = "log_file"
	// LabelDebugEnable is the label for whether debugging should be enabled.
	LabelDebugEnable = "debug"
	// LabelHoneypot is the label for the honeypot input field. If the honeypot
	// has a value when the form is submitted, the form submission will be
	// discarded
	LabelHoneypot = "honeypot"
	// LabelHandleIf is the label for the mapping of form input names to
	// what kind of values they must have for the handler to handle. A value of
	// `true` indicates any non-empty value, while an array/slice of string values
	// indicate valid values - intended for checkboxes and radio buttons with
	// predefined values. A value of `""` indicates a value may be empty. An array
	// containing only `""` indicates the value *must* be empty.
	// All conditions must be met for the handler to handle. If validation is
	// desired instead (i.e. return an error if the field is empty), allow all
	// values and use the "Errorf" function in a template instead.
	LabelHandleIf = "handle_if"
	// LabelName is the label for the internal name of the plugin instance.
	// This is required to differentiate multiple handlers of the same type
	LabelName = "name"
	// LabelCookieKey is the label for RPC plugin cookie keys, to help
	// validate that you are using the correct plugin
	LabelCookieKey = "cookie_key"
	// LabelCookieVal is the label for RPC plugin cookie values, to help
	// validate that you are using the correct plugin
	LabelCookieVal = "cookie_val"
	// LabelProtocol is the label for the RPC plugin protocol version
	LabelProtocol = "protocol"
	// LabelCommand is the label for the command to run for running the
	// plugin server. It is parsed as a slice of strings.
	LabelCommand = "command"
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

// Handler represents anything that can handle a form submission.
//
// OriginAllowed and Honeypot are not strictly necessary but are here to
// allow for struct inheritance from handler.Base, where ShouldHandle uses
// both to determine if the form should be processed.
type Handler interface {
	Handle(*pb.HTTPRequest) *errors.HTTPError
	ShouldHandle(*pb.HTTPRequest) (bool, *errors.HTTPError)
}

// handleCondition indicates constraints on form values to determine if the
// handler can handle.
type handleCondition struct {
	MustBeNonEmpty bool
	AllowedValues  map[string]struct{}
}

// FormValuesFunc generates a "FormValues" function that returns the full
// slice of values instead of just the first
func FormValuesFunc(req *pb.HTTPRequest) func(string) []string {
	return func(name string) []string {
		if req.Form[name] == nil {
			return nil
		}

		var vals []string
		for _, val := range req.Form[name].Values {
			vals = append(vals, val.GetStr())
		}

		return vals
	}
}

// ErrorfFunc generates a function that wraps fmt.Errorf so that the error is
// returned as the second value, which tells the template that an error
// occurred and causes it to be returned. Also sets the given HTTPError pointer
// to the error that occurred.
//
// Keep in mind that errors are both returned to the user and logged on the
// system to help with detecting these sorts of things, so be careful when
// using Errorf on forms containing personal and/or sensitive information
func ErrorfFunc(err *errors.HTTPError) func(format string, v ...interface{}) (interface{}, error) {
	return func(format string, v ...interface{}) (interface{}, error) {
		*err = *errors.NewHTTPError(fmt.Sprintf(format, v...), 400)
		return nil, err
	}
}
