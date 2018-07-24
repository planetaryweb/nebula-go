package errors

// Errors are created here so they can be referenced later

// ErrBaseConfig is a template for an error where a set of configuration
// options could not be parsed.
const ErrBaseConfig = "could not parse config for %s: %s"

// ErrConfigItem is a template for an error where a particular
// configuration option could not be parsed.
const ErrConfigItem = "could not parse \"%s\": %s"

// HTTPError provides the closest HTTP status code to the accompanying
// error for more nuanced responses to clients
type HTTPError struct {
	err    string
	status int
}

// NewHTTPError returns a new instance of HTTPError
func NewHTTPError(e string, s int) *HTTPError {
	return &HTTPError{
		err:    e,
		status: s}
}

func (e HTTPError) Error() string {
	return e.err
}

// Status returns the HTTP status code associated with this Error
func (e HTTPError) Status() int {
	return e.status
}

// HTTPErrorToChan provides a function for sending an HTTPError on a channel,
// creating the HTTPError if necessary, using `def` as the status code.
func HTTPErrorToChan(ch chan *HTTPError, err error, def int) {
	httperr, ok := err.(*HTTPError)
	if !ok { // Just a normal error
		ch <- NewHTTPError(err.Error(), def)
	} else { // HTTPError
		ch <- httperr
	}
}
