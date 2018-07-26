package errors

import "strings"

const ErrNilInterface = "empty value (nil interface{})"

const ErrNotInt64 = "could not parse \"%#v\" as integer"
const ErrNotFloat64 = "could not parse \"%#v\" as float"
const ErrNotBool = "could not parse \"%#v\" as boolean"
const ErrNotString = "could not parse \"%#v\" as string"

const ErrNotSlice = "could not parse \"%#v\" as slice"
const ErrNotMapStringKeys = "could not parse \"%#v\" as a map with string keys"

func IsNilInterface(e error) bool {
	return e.Error() == ErrNilInterface
}

func IsNotInt64(e error) bool {
	return strings.HasSuffix(e.Error(), "integer")
}

func IsNotFloat64(e error) bool {
	return strings.HasSuffix(e.Error(), "float")
}

func IsNotBool(e error) bool {
	return strings.HasSuffix(e.Error(), "boolean")
}

func IsNotString(e error) bool {
	return strings.HasSuffix(e.Error(), "string")
}

func IsNotMapStringKeys(e error) bool {
	return strings.HasSuffix(e.Error(), "map with string keys")
}

func IsNotSlice(e error) bool {
	return strings.HasSuffix(e.Error(), "slice")
}
