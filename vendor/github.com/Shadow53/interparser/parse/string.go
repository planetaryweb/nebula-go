package parse

import (
    "errors"
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
        return "", errors.New(e.ErrNotString)
    }

    return s, nil
}

// StringOrDefault attempts to parse the given interface as a string and
// returns the given default value if parsing fails
func StringOrDefault(d interface{}, def string) string {
    s, err := String(d)
    if err != nil {
        return def
    }
    return s
}
