package parse

import (
    "errors"
    e "github.com/Shadow53/interparser/errors"
)

// Slice attempts to parse the given interface as a slice of interfaces that
// can each then be parsed with a member of this package.
func Slice(d interface{}) ([]interface{}, error) {
    if d == nil {
        return nil, errors.New(e.ErrNilInterface)
    }

    s, ok := d.([]interface{})
    if !ok {
        return nil, errors.New(e.ErrNotSlice)
    }

    return s, nil
}

// SliceOrNil attempts to parse the given interface as a slice of interfaces,
// returning a nil slice on errors.
func SliceOrNil(d interface{}) []interface{} {
    s, err := Slice(d)
    if err != nil {
        return nil
    }
    return s
}
