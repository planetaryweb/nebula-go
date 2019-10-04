package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/hashicorp/go-hclog"
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
				c.Logger.Info("Updating configuration", "file", file)
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
					c.Logger.Error(
						"Error reloading config. Reusing old config", "error", err.Error())
					break ChanSel
				}
			}
			c.Logger.Debug("Configuration loaded", "files", hclog.Fmt("%#v", files))
			for _, file := range files {
				c.WatchFile(file, fCh)
			}
			c.Logger.Info("Loaded configuration file", "file", file)
			// Stop watching stuff with old config
			if oldConf != nil {
				c.Logger.Info("Ending old configuration's file watching")
				err = oldConf.StopWatchingAll()
				if err != nil {
					c.Logger.Error("Error while stopping file watching", "error", err)
				}
			}
			// Configuration (re)load worked, make and load server
			server = c.CreateServer()
			go func(s *http.Server, ch chan error) {
				c.Logger.Info("Starting server")
				err := s.ListenAndServe()
				if err != http.ErrServerClosed {
					ch <- err
				} else {
					ch <- nil
				}
			}(server, errCh)
			// Close previous server
			if oldServ != nil {
				c.Logger.Info("Shutting down old server")
				ctx, cancel := context.WithTimeout(
					context.Background(), time.Duration(1)*time.Minute)
				err = oldServ.Shutdown(ctx)
				cancel()
				if err != nil {
					c.Logger.Error(
						"Error while shutting down old server", "error", err.Error())
				}
			}
		case err := <-errCh:
			if err != nil {
				c.Logger.Info("Server exited with error")
				c.Logger.Error("Server exited with error", "error", err.Error())
			} else {
				c.Logger.Info("Server shutdown")
			}
		case sig := <-sigch:
			if sig == os.Interrupt || sig == syscall.SIGTERM {
				c.Logger.Info("Received interrupt signal. Exiting...")
				ctx, cancel := context.WithTimeout(
					context.Background(), time.Duration(1)*time.Minute)
				err := server.Shutdown(ctx)
				cancel()
				if err != nil {
					c.Logger.Error("Server exited with error", "error", err)
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
