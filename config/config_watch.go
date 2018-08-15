package config

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
)

// WatchFile watches the given file as a configuration file and reloads the
// Config struct and the given server - if provided - on file changes.
// Spawns a goroutine that can be ended by calling Config.StopWatching or
// Config.StopWatchingAll.
func (c *Config) WatchFile(filename string, ch chan string) error {
	if c.fWatcher == nil {
		var err error
		c.fWatcher, err = fsnotify.NewWatcher()
		if err != nil {
			return fmt.Errorf("error creating file watcher: %s", err)
		}

		go func(ch chan string) {
			for {
				if c.fWatcher == nil {
					break
				}
				select {
				case event := <-c.fWatcher.Events:
					// Include rename because some editors use swap files,
					// which causes the rename op to be returned
					if event.Op&fsnotify.Write == fsnotify.Write ||
						event.Op&fsnotify.Rename == fsnotify.Rename {
						c.Logger.Logln("Detected file change")
						ch <- event.Name
					}
				case err := <-c.fWatcher.Errors:
					if err != nil {
						c.Logger.Errorf("Error while watching file: %s", err)
					} else {
						break
					}
				}
			}
		}(ch)
	}

	c.Logger.Debugf("Adding watcher for %s", filename)
	err := c.fWatcher.Add(filename)
	if err != nil {
		return fmt.Errorf("Error subscribing to %s: %s", filename, err)
	}

	return nil
}

// StopWatching stops the Config from watching a specific file, while
// continuing to watch any others that have been subscribed to. It does not
// close the file watcher or end the spawned goroutine for watching files.
// For that, use Config.StopWatchingAll.
func (c *Config) StopWatching(file string) error {
	c.Logger.Debugf("Removing watcher for %s", file)
	return c.fWatcher.Remove(file)
}

// StopWatchingAll stops the Config from watching any of the subscribed files,
// closes the file watcher, and ends the goroutine that was watching the
// files.
func (c *Config) StopWatchingAll() error {
	c.Logger.Debug("Ending watching of all files")
	err := c.fWatcher.Close()
	c.fWatcher = nil
	return err
}
