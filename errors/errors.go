package errors

// Errors are created here so they can be referenced later

// ErrBaseConfig is a template for an error where a set of configuration
// options could not be parsed.
const ErrBaseConfig = "could not parse config for %s: %s"

// ErrConfigItem is a template for an error where a particular
// configuration option could not be parsed.
const ErrConfigItem = "could not parse \"%s\": %s"
