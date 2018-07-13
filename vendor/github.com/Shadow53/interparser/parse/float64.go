package parse

import (
    "errors"
    "encoding/json"
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
            return 0.0, errors.New(e.ErrNotFloat64)
        }

        return n.Float64()
    }

    return f, nil
}

// Float64OrDefault attempts to parse the given interface as a float64 and
// returns it if successful, otherwise returns the given default on errors.
func Float64OrDefault(d interface{}, def float64) float64 {
    f, err := Float64(d)
    if err != nil {
        return def
    }
    return f
}
