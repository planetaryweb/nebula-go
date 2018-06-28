package config

import (
	"fmt"

	"github.com/Shadow53/hugo-forms/handler"
)

// Default values for configuration options go here
const DefaultMaxFileSize = uint(5 * 1024 * 1024) // 5 MiB

// labels contains the names of configuration options as found in the
// configuration file. This prevents unnoticed issues due to misspelling
// the key name.
var labels = struct{}{
	MaxFileSize: "max_file_size"}

// Config represents the parsed server configuration.
type Config struct {
	Handlers    map[string]handler.Handler
	MaxFileSize int
}

// Parse takes an interface{} representing the configuration and parses it
// to populate a Config struct
//
// The interface{} is expected to be a map[string]interface{}, such as those
// created when calling Unmarshal([]byte, interface) by passing a pointer to
// the interface as the second argument. This makes it easy to use
// configurations from multiple different kinds of configuration files,
// including JSON, YAML, and TOML, as well as programmatically creating them
// in a Go program using this as a library.
func Parse(c interface{}) (Config, error) {
	var conf Config

	// Check if the passed interface is nil
	if c == nil {
		return conf, errors.New("cannot parse a nil configuration")
	}

	// Try to parse non-nil data as a map
	data, ok := c.(map[string]interface{})
	if !ok {
		return conf, errors.New("could not parse configuration as a map")
	}

	// If the interface was a nil map, error
	if data == nil {
		return conf, errors.New("configuration map cannot be nil")
	}

	// We have a non-nil map - parse it for values

	// Max file size
	if data[labels.MaxFileSize] == nil {
		// Not specified, use default
		conf.MaxFileSize = defaults.MaxFileSize
	} else {
		// Try to parse the value
		conf.MaxFileSize, ok = data[labels.MaxFileSize].(int)
		if !ok {
			// JSON does numbers as float64s, so check for that
			f, ok := data[labels.MaxFileSize].(float64)
			if !ok {
				// Wasn't a float either, so error
				return conf, fmt.Errorf("could not parse %s as unsigned int", labels.MaxFileSize)
			} else {
				// Parse float as int as store
				conf.MaxFileSize = int(f)
			}
		}

		// Check that parsed MaxFileSize is valid
		if conf.MaxFileSize < 0 {
			return conf, fmt.Errorf("%s must be non-negative", labels.MaxFileSize)
		}
	}

	// Handlers
	// The bulk of the configuration is parsed here.
}
