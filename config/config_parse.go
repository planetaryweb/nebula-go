package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	e "git.shadow53.com/BluestNight/nebula-forms/errors"
	"git.shadow53.com/BluestNight/nebula-forms/handler"
	"git.shadow53.com/Shadow53/merge-config/merge"
	l "git.shadow53.com/BluestNight/nebula-forms/log"
	"github.com/BurntSushi/toml"
	"github.com/Shadow53/interparser/parse"
	"github.com/fsnotify/fsnotify"
	"sync"
	"os"
	"path"
	"log"
)

func (c *Config) unmarshalLoggers(data map[string]interface{}) error {
	// Might log to stdout in multiple places, stdout is not safe for
	// concurrent access. The logger is, so save logger to pass multiple times
	outlog := log.New(os.Stdout, "", log.LstdFlags)

	// Standard logger
	c.Logger.AddLogger(outlog)
	filename, err := parse.StringOrDefault(data[LabelLogFile], DefaultLogFile)
	if err != nil {
		return fmt.Errorf(e.ErrConfigItem, LabelLogFile, err)
	}
	// Create parent folder before creating log file
	err = os.MkdirAll(path.Dir(filename), 0700)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	c.Logger.AddLogger(log.New(file, "", log.LstdFlags))

	// Error logger
	c.Logger.AddErrorLogger(log.New(os.Stderr, "error: ", log.LstdFlags))
	filename, err = parse.StringOrDefault(data[LabelErrorFile], DefaultErrorFile)
	if err != nil {
		return fmt.Errorf(e.ErrConfigItem, LabelErrorFile, err)
	}
	err = os.MkdirAll(path.Dir(filename), 0700)
	if err != nil {
		return err
	}
	file, err = os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	c.Logger.AddErrorLogger(log.New(file, "error: ", log.LstdFlags))

	// Determine if debugging should be enabled
	c.Logger.IsDebug, err = parse.BoolOrDefault(data[LabelDebugEnable], false)
	if err != nil {
		return err
	}
	c.Logger.Debug("Debugging is enabled")
	return nil
}

func (c *Config) unmarshalHandlers(data map[string]interface{}) error {
	// Actual handlers
	// Checking for nil because configuration files can be partial
	if data[handler.LabelHandlers] != nil {
		handlerMap, err := parse.MapStringKeys(data[handler.LabelHandlers])
		for plugin, conf := range handlerMap {
			// Load plugin first
			plugPath := filepath.Join(c.PluginDir, plugin + ".so")
			// Attempt to load plugin into map, return error if occurs
			c.plugins[plugin], err = handler.LoadPlugin(plugPath)
			if err != nil {
				return fmt.Errorf("could not load plugin %s: %s", plugin, err)
			}

			// Run Configure on the plugin before creating handlers
			if data[plugin] != nil {
				err = c.plugins[plugin].Configure(data[plugin])
				if err != nil {
					return fmt.Errorf(e.ErrConfigItem, plugin, err)
				}
			}
			hMap, err := parse.MapStringKeys(conf)
			if err != nil {
				return fmt.Errorf(e.ErrConfigItem, handler.LabelHandlers,
					fmt.Sprintf(e.ErrConfigItem, plugin, err))
			}
			for hName, hConf := range hMap {
				hConfMap, err := parse.MapStringKeys(hConf)
				if err != nil {
					return fmt.Errorf(e.ErrConfigItem, handler.LabelHandlers,
						fmt.Sprintf(e.ErrConfigItem, plugin,
							fmt.Sprintf(e.ErrConfigItem, hName, err)))
				}
				hPath, err := parse.String(hConfMap[handler.LabelHandlerPath])
				if err != nil {
					return fmt.Errorf(e.ErrConfigItem, handler.LabelHandlers,
						fmt.Sprintf(e.ErrConfigItem, plugin,
							fmt.Sprintf(e.ErrConfigItem, hName, err)))
				}
				h, err := c.plugins[plugin].NewHandler(hConf)
				if err != nil {
					return fmt.Errorf(e.ErrConfigItem, handler.LabelHandlers,
						fmt.Sprintf(e.ErrConfigItem, plugin,
							fmt.Sprintf(e.ErrConfigItem, hName, err)))
				}
				c.AddHandler(hPath, h)
				c.Logger.Debugf("Registered handler for \"%s\" named %s",
					hPath, hName)
			}
		}
	} else {
		return errors.New("at least one handler should be configured for this server")
	}
	return nil
}

// Unmarshal takes an interface{} representing the configuration and parses it
// to populate a Config struct
//
// The interface{} is expected to be a map[string]interface{}, such as those
// created when calling Unmarshal([]byte, interface) by passing a pointer to
// the interface as the second argument. This makes it easy to use
// configurations from multiple different kinds of configuration files,
// including JSON, YAML, and TOML, as well as programmatically creating them
// in a Go program using this as a library.
func (c *Config) Unmarshal(conf interface{}) (err error) {
	// Prepare the *Config - i.e. reset
	c.plugins = make(map[string]*handler.Plugin)
	c.handlers = make(map[string][]handler.Handler)
	c.Logger = &l.Logger{}
	c.hMutex = sync.RWMutex{}
	c.fWatcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	// Parse configuration as map
	data, err := parse.MapStringKeys(conf)
	if err != nil {
		return fmt.Errorf("could not unmarshal config as map: %s", err)
	}

	// If the interface was a nil map, error
	if data == nil {
		return errors.New("configuration map cannot be nil")
	}

	// We have a non-nil map - parse it for values

	// Try to parse the value
	c.MaxFileSize, err = parse.Int64OrDefault(data[LabelMaxFileSize], DefaultMaxFileSize)
	if err != nil {
		return fmt.Errorf(e.ErrConfigItem, LabelMaxFileSize, err)
	}

	// Check that parsed MaxFileSize is valid
	if c.MaxFileSize < 0 {
		return fmt.Errorf("%s must be non-negative", LabelMaxFileSize)
	}

	// Port
	c.Port, err = parse.Int64OrDefault(data[LabelPort], DefaultPort)
	if err != nil {
		return fmt.Errorf(e.ErrConfigItem, LabelPort, err)
	}

	// Plugins directory - required or else won't know where to load from
	c.PluginDir, err = parse.StringOrDefault(data[LabelPluginDir], DefaultPluginDir)
	if err != nil {
		return fmt.Errorf(e.ErrConfigItem, LabelPluginDir, err)
	}

	if err = c.unmarshalLoggers(data); err != nil {
		return err
	}

	return c.unmarshalHandlers(data)
}

func (c *Config) tomlInner(filename string) (map[string]interface{}, []string, error) {
	var v interface{}
	var files []string

	_, err := toml.DecodeFile(filename, &v)
	if err != nil {
		return nil, nil, err
	}

	// Add to list of files
	files = append(files, filename)

	// If first file parsed, save
	if c.RootConfig == "" {
		c.RootConfig = filename
	}

	// Check for included directories and load TOML files in them too
	conf, err := parse.MapStringKeys(v)
	if err != nil {
		return nil, nil, fmt.Errorf("could not parse config for include_dir: %s", err)
	}

	// Get directory
	dir, err := parse.StringOrDefault(conf[LabelIncludeDir], "")
	if err != nil {
		return nil, nil, err
	}

	if dir != "" {
		// Make absolute if it is not already
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(filepath.Dir(filename), dir)
		}

		// Get file listing from inclde_dir
		included, err := ioutil.ReadDir(dir)
		if err != nil {
			return nil, nil, err
		}

		// Parse each file
		for _, file := range included {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".toml") {
				// Get file's path
				filename := filepath.Join(dir, file.Name())
				// Parse inner file's TOML
				innerConf, innerFiles, err := c.tomlInner(filename)
				if err != nil {
					return nil, nil, err
				}
				// Merge nested configuration with this one
				conf = merge.Merge(conf, innerConf)
				// Append current file and all files included in it
				files = append(files, innerFiles...)
			}
		}
	}

	return conf, files, nil
}

// ParseConfigTOMLFile parses the TOML file at the given path and returns a
// *Config upon successful parsing.
//
// For parsing other kinds of files, parse the configuration into an interface{}
// and use the Config.Unmarshal function instead.
func (c *Config) ParseConfigTOMLFile(filename string) ([]string, error) {
	// Parse files and merge
	conf, files, err := c.tomlInner(filename)
	if err != nil {
		return nil, err
	}

	// Full configuration is built, parse
	err = c.Unmarshal(conf)
	if err != nil {
		return nil, err
	}

	// Return read files
	return files, nil
}
