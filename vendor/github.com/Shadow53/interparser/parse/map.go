package parse

import (
	"errors"
	"fmt"

	e "github.com/Shadow53/interparser/errors"
)

// MapStringKeys attempts to parse the given interface as a map with string
// keys.
func MapStringKeys(d interface{}) (map[string]interface{}, error) {
	if d == nil {
		return nil, errors.New(e.ErrNilInterface)
	}

	m, ok := d.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf(e.ErrNotMapStringKeys, d)
	}

	return m, nil
}

// MapStringKeysOrNew returns a new map[string]interface{} if the given
// interface is nil, otherwise it attempts to parse the given interface as a
// map with string keys, returning any errors that occur. If the interface is
// a nil *map*, the nil map is returned.
func MapStringKeysOrNew(d interface{}) (map[string]interface{}, error) {
	if d == nil {
		return make(map[string]interface{}), nil
	}

	return MapStringKeys(d)
}
