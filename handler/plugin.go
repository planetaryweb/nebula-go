package handler

import (
	"fmt"
	"plugin"
)

// Plugin wraps a plugin.Plugin to turn it into a Handler
type Plugin struct {
	p          *plugin.Plugin
	NewHandler func(interface{}) (Handler, error)
	Configure  func(interface{}) error
}

// LoadPlugin loads a plugin from the given path as a Handler
func LoadPlugin(path string) (*Plugin, error) {
	p := &Plugin{}
	var err error

	p.p, err = plugin.Open(path)
	if err != nil {
		return nil, err
	}

	sym, err := p.p.Lookup("NewHandler")
	if err != nil {
		return nil, err
	}

	var ok bool
	p.NewHandler, ok = sym.(func(interface{}) (Handler, error))
	if !ok {
		return nil, fmt.Errorf(
			"plugin's NewHandler does not implement the right interface: want %s got %T",
			"func(interface{}) (Handler, error)", sym)
	}

	sym, err = p.p.Lookup("Configure")
	if err != nil {
		return nil, err
	}

	p.Configure, ok = sym.(func(interface{}) error)
	if !ok {
		return nil, fmt.Errorf(
			"plugin's Configure does not implement the right interface: want %s got %T",
			"func(interface{}) error", sym)
	}

	return p, nil
}
