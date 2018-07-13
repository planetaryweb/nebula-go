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
	c, err := config.ParseConfigFile(config.DefaultConfigFile)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"Error occurred while parsing configuration file: %s\n", err)
	}

	server := c.CreateServer()

	ch := make(chan error)
	go func(s *http.Server, ch chan error) {
		c.Logger.Logln("Starting server")
		err := s.ListenAndServe()
		if err != http.ErrServerClosed {
			ch <- err
		} else {
			ch <- nil
		}
	}(server, ch)

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)

	for {
		select {
		case err := <-ch:
			if err != nil {
				c.Logger.Logln("Server exited with error")
				c.Logger.Errorf("Server exited with error: %s\n", err)
			} else {
				c.Logger.Logln("Server shutdown")
			}
		case sig := <-sigch:
			if sig == os.Interrupt {
				c.Logger.Logln("Received interrupt signal. Exiting...")
				ctx, _ := context.WithTimeout(
					context.Background(), time.Duration(1)*time.Minute)
				err := server.Shutdown(ctx)
				if err != nil {
					c.Logger.Errorf("Server exited with error: %s\n", err)
					os.Exit(1)
				}
				os.Exit(0)
			}
		}
	}
}
