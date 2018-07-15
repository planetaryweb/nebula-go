package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/BluestNight/static-forms/config"
)

func main() {
	// Create file channel, populate with initial config file
	// (So for loop will create config/server)
	fCh := make(chan string, 1)
	fCh <- config.DefaultConfigFile

	// Create error channel
	errCh := make(chan error)

	// Create signal channel, listen for interrupt
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)

	// Create empty pointers
	var c *config.Config
	var server *http.Server
	var err error

	for {
		select {
		case file := <-fCh:
			// Backup pointers
			oldServ := server
			oldConf := c
			// (Re)load config
			c, err = config.ParseConfigFile(file)
			if err != nil {
				if oldConf == nil {
					// Exit with error if no backed up config
					fmt.Fprintf(os.Stderr,
						"Error occurred while parsing configuration file: %s\n", err)
					os.Exit(1)
				} else {
					// If backed up config, keep using it
					c = oldConf
					c.Logger.Errorf(
						"Error reloading config: %s\nReusing old config\n", err)
					continue
				}
			}
			c.Logger.Logf("Loaded configuration file at %s", file)
			// Stop watching stuff with old config
			if oldConf != nil {
				c.Logger.Logln("Ending old configuration's file watching")
				err = oldConf.StopWatchingAll()
				if err != nil {
					c.Logger.Errorf(
						"Error while stopping file watching: %s\n", err)
				}
			}
			// Watch configuration file for changes
			err := c.WatchFile(file, fCh)
			if err != nil {
				c.Logger.Errorf("Error registering file watcher: %s", err)
			}
			// Configuration (re)load worked, make and load server
			server = c.CreateServer()
			go func(s *http.Server, ch chan error) {
				c.Logger.Logln("Starting server")
				err := s.ListenAndServe()
				if err != http.ErrServerClosed {
					ch <- err
				} else {
					ch <- nil
				}
			}(server, errCh)
			// Close previous server
			if oldServ != nil {
				c.Logger.Logln("Shutting down old server")
				ctx, cancel := context.WithTimeout(
					context.Background(), time.Duration(1)*time.Minute)
				err = oldServ.Shutdown(ctx)
				cancel()
				if err != nil {
					c.Logger.Errorf(
						"Error while shutting down old server: %s\n", err)
				}
			}
		case err := <-errCh:
			if err != nil {
				c.Logger.Logln("Server exited with error")
				c.Logger.Errorf("Server exited with error: %s\n", err)
			} else {
				c.Logger.Logln("Server shutdown")
			}
		case sig := <-sigch:
			if sig == os.Interrupt {
				c.Logger.Logln("Received interrupt signal. Exiting...")
				ctx, cancel := context.WithTimeout(
					context.Background(), time.Duration(1)*time.Minute)
				err := server.Shutdown(ctx)
				cancel()
				if err != nil {
					c.Logger.Errorf("Server exited with error: %s\n", err)
					os.Exit(1)
				}
				os.Exit(0)
			}
		}
	}
}
