package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gitlab.com/BluestNight/nebula-forms/config"
)

func main() {
	// Set flags
	configFile := flag.String("conf", config.DefaultConfigFile,
		"the configuration file to use")
	showHelp := flag.Bool("help", false, "show this help message")

	flag.Parse()

	// Print help message, if requested
	if *showHelp {
		flag.PrintDefaults()
		os.Exit(0)
	}

	// Create file channel, populate with initial config file
	// (So for loop will create config/server)
	// Buffer with two in case of changed file followed by SIGHUP
	fCh := make(chan string, 2)
	fCh <- *configFile

	// Create error channel
	errCh := make(chan error)

	// Create signal channel, listen for interrupt
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	// Create empty pointers
	var c *config.Config
	var server *http.Server

	for {
	ChanSel:
		select {
		case file := <-fCh:
			if c != nil {
				c.Logger.Logf(
					"Config file %s was modified; updating configuration\n", file)
				file = c.RootConfig
			}
			// Backup pointers
			oldServ := server
			oldConf := c
			c = &config.Config{}
			// (Re)load config
			files, err := c.ParseConfigTOMLFile(file)
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
					break ChanSel
				}
			}
			c.Logger.Debugf("Configuration loaded from following files: %#v", files)
			for _, file := range files {
				c.WatchFile(file, fCh)
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
			if sig == os.Interrupt || sig == syscall.SIGTERM {
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
			} else if sig == syscall.SIGHUP {
				// FreeBSD manual says this signal tells things to
				// reload config files. Do this by sending the file on
				// the channel
				fCh <- *configFile
			}
		}
	}
}
