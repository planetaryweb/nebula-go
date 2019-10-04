package config

import (
	"github.com/hashicorp/go-hclog"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/hashicorp/go-plugin"
	"gitlab.com/BluestNight/nebula-forms/handler"
)

// Default values for configuration options go here

// DefaultMaxFileSize is the default value for maximum form upload body
const DefaultMaxFileSize = int64(5 * 1024 * 1024) // 5 MiB
// DefaultPort is the default port that the server will run at
const DefaultPort = int64(2002)

// labels contains the names of configuration options as found in the
// configuration file. This prevents unnoticed issues due to misspelling
// the key name.
var (
	// LabelPort is the label for the value containing the port the
	// server will run on.
	LabelPort = "port"
	// LabelErrorFile is the label for the path to place the error log file.
	LabelErrorFile = "error_file"
	// LabelLogFile is the label for the path to place the access log file.
	LabelLogFile = "log_file"
	// LabelDebugEnable is the label for whether debugging should be enabled.
	LabelDebugEnable = "debug"
	// LabelMaxFileSize is the label for the maximum size of a single request.
	// This option is currently not respected, at least until I figure out how
	// to enforce it
	LabelMaxFileSize = "max_file_size"
	// LabelIncludeDir is the label for the *directory* from which to load more
	// configuration files. It is to be used by functions that load the
	// configuration file *before* parsing. In this case, by
	// ParseConfigTOMLFile instead of Unmarshal
	LabelIncludeDir = "include_dir"
	// LabelPluginDir is the label for the directory in which Nebula can find
	// and load handler plugins.
	LabelPluginDir = "plugins_dir"
)

// Config represents the parsed server configuration.
type Config struct {
	fWatcher    *fsnotify.Watcher
	RootConfig  string
	Port        int64
	PluginDir   string
	Logger      hclog.Logger
	hMutex      sync.RWMutex
	handlers    map[string][]handler.Handler
	clients     map[string]*plugin.Client
	plugins     map[string]handler.Handler
	MaxFileSize int64
}

// AddHandler adds a handler for a given handler path.
// Safe for parallel use.
func (c *Config) AddHandler(path string, h handler.Handler) {
	if h != nil && path != "" && path[0] == '/' {
		c.hMutex.Lock()
		if c.handlers == nil {
			c.handlers = make(map[string][]handler.Handler)
		}
		s := c.handlers[path]
		s = append(s, h)
		c.handlers[path] = s
		c.hMutex.Unlock()
	}
}

// GetHandlers retrieves the list of handlers for a given handler path.
// Safe for parallel use.
func (c *Config) GetHandlers(path string) []handler.Handler {
	c.hMutex.RLock()
	defer c.hMutex.RUnlock()
	return c.handlers[path]
}
