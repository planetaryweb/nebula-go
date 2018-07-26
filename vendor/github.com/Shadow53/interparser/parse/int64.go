package parse

import (
	"encoding/json"
	"errors"
	"fmt"

	e "github.com/Shadow53/interparser/errors"
)

// Int attempts to parse the given interface{} as an int and returns an error
// if it could not be parsed as an int. For compatibility with JSON data, it
// may convert float64 values to integers
func Int64(d interface{}) (int64, error) {
	if d == nil {
		return -1, errors.New(e.ErrNilInterface)
	}

	i, ok := d.(int64)
	if !ok {
		if i2, ok := d.(int); ok {
			// Was a normal int - convert to int64
			return int64(i2), nil
		} else if n, ok := d.(json.Number); ok {
			// Some interfaces, i.e. JSON, parse all numbers as float64
			// Check for json.Number and try to parse int that way
			i2, err := n.Int64()
			return i2, err
		} else {
			// Not an int64
			return -1, fmt.Errorf(e.ErrNotInt64, d)
		}
	}

	// Parsed as int, return
	return i, nil
}

// Int64OrDefault returns the default value if the interface is nil, otherwise
// it attempts to parse the value as an int64 and returns it or an error, if
// one occurs.
func Int64OrDefault(d interface{}, def int64) (int64, error) {
	if d == nil {
		return def, nil
	}

	return Int64(d)
}
