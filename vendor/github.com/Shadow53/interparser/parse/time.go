package parse

import (
	"time"
)

// TimeFromMilliseconds takes an interface{} that is an integer representing
// Unix/epoch time in milliseconds and converts it to a time.Time
func TimeFromMilliseconds(data interface{}) (time.Time, error) {
	i, err := Int64(data)
	if err != nil {
		return time.Time{}, err
	}

	// Trim off milliseconds
	sec := i / 1000
	// Get milliseconds and convert to nanoseconds
	nsec := (i % 1000) * 1000000
	// Return result of time.Unix
	return time.Unix(sec, nsec), nil
}
