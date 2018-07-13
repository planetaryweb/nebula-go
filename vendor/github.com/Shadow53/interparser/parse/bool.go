package parse

import (
    "errors"
    e "github.com/Shadow53/interparser/errors"
)

// Bool attempts to parse the given interface as a boolean and returns an error
// if it could not be parsed as a boolean. Does not convert strings like "true"
// and "false" or "yes" and "no" to booleans.
func Bool(d interface{}) (bool, error) {
    if d == nil {
        return false, errors.New(e.ErrNilInterface)
    }

    b, ok := d.(bool)
    if !ok {
        return false, errors.New(e.ErrNotBool)
    }

    return b, nil
}

// BoolOrDefault attempts to parse the given interface as a boolean. If the
// parsing fails for any reason, the default value is returned
func BoolOrDefault(d interface{}, def bool) bool {
    b, err := Bool(d)
    if err != nil {
        return def
    }
    return b
}
