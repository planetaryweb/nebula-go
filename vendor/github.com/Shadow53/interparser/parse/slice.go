package parse

import (
	"errors"
	"fmt"

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
		return nil, fmt.Errorf(e.ErrNotSlice, d)
	}

	return s, nil
}

// SliceOrNil returns a nil slice if the given interface is nil, otherwise it
// attempts to parse the given interface as a slice of interfaces, returning
// any errors that occur.
func SliceOrNil(d interface{}) ([]interface{}, error) {
	if d == nil {
		return nil, nil
	}

	return Slice(d)
}
