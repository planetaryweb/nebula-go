package parse

import (
    "errors"
    "encoding/json"
    e "github.com/Shadow53/interparser/errors"
)

// Int attempts to parse the given interface{} as an int and returns an error
// if it could not be parsed as an int. For compatibility with
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
            return -1, errors.New(e.ErrNotInt64)
        }
    }

    // Parsed as int, return
    return i, nil
}

func Int64OrDefault(d interface{}, def int64) int64 {
    i, err := Int64(d)
    if err != nil {
        return def
    }
    return i
}
