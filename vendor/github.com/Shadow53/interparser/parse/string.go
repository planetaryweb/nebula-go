package parse

import (
	"errors"
	"fmt"

	e "github.com/Shadow53/interparser/errors"
)

// String attempts to parse the given interface as a string and returns an
// error if parsing fails.
func String(d interface{}) (string, error) {
	if d == nil {
		return "", errors.New(e.ErrNilInterface)
	}

	s, ok := d.(string)
	if !ok {
		return "", fmt.Errorf(e.ErrNotString, d)
	}

	return s, nil
}

// StringOrDefault returns the default string if the given interface is nil,
// otherwise it attempts to parse the given interface as a string and
// returns any errors that may occur.
func StringOrDefault(d interface{}, def string) (string, error) {
	if d == nil {
		return def, nil
	}

	return String(d)
}
