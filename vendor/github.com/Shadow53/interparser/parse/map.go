package parse

import (
    "errors"
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
        return nil, errors.New(e.ErrNotMapStringKeys)
    }

    return m, nil
}


// MapStringKeysOrNew attempts to parse the given interface as a map with string
// keys. If parsing fails, an empty non-nil map is returned.
func MapStringKeysOrNew(d interface{}) map[string]interface{} {
    m, err := MapStringKeys(d)
    if err != nil {
        return make(map[string]interface{})
    }
    return m
}
