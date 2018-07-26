package parse

import (
	"encoding/json"
	"errors"
	"fmt"

	e "github.com/Shadow53/interparser/errors"
)

// Float64 attempts to parse the given interface as float64. It is compatible
// with package json's Number struct but does not attempt to convert strings
func Float64(d interface{}) (float64, error) {
	if d == nil {
		return 0.0, errors.New(e.ErrNilInterface)
	}

	f, ok := d.(float64)
	if !ok {
		// Some interfaces, i.e. json, treat all numbers as float64 but
		// wrap the number in a struct, i.e. json.Number. Retrieve the
		// number from this struct
		n, ok := d.(json.Number)
		if !ok {
			return 0.0, fmt.Errorf(e.ErrNotFloat64, d)
		}

		return n.Float64()
	}

	return f, nil
}

// Float64OrDefault returns the default value if the given interface is nil,
// otherwise it attempts to parse the interface as a float64, returning an
// error if one occurs
func Float64OrDefault(d interface{}, def float64) (float64, error) {
	if d == nil {
		return def, nil
	}

	return Float64(d)
}
