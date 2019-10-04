//
// main.go
// Copyright (C) 2019 shadow53 <shadow53@shadow53.com>
//
// Distributed under terms of the MIT license.
//

package main

import (
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/hashicorp/go-plugin"
	"gitlab.com/BluestNight/nebula-forms/handler"
	"io/ioutil"
	"os"
)

var handshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "NEBULA_EMAIL",
	MagicCookieValue: "EMAIL",
}

var confFile string

func init() {
	flag.StringVar(&confFile, "config", "email.toml",
		"path to a configuration file for this plugin")
	flag.Parse()
}

func errorExit(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

func main() {
	var config interface{}

	cBytes, err := ioutil.ReadFile(confFile)

	if err != nil {
		errorExit(err.Error())
	}

	err = toml.Unmarshal(cBytes, &config)
	if err != nil {
		errorExit(err.Error())
	}

	handle, err := NewHandler(config)
	if err != nil {
		errorExit(err.Error())
	}

	name := handle.Name()
	var pluginMap = map[string]plugin.Plugin{
		name: &handler.Plugin{Impl: handle},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
		GRPCServer:      plugin.DefaultGRPCServer,
	})
}
