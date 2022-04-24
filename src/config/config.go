package config

import (
	"fmt"
	"io/ioutil"

	"example.com/gocr/src/events"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v3"
)

const (
	ErrNotFound = "no configuration exists for the given key"
)

type ActionInner struct {
	Key  string `json:"key,omitempty" yaml:"key,omitempty"`
	Args string `json:"command,omitempty" yaml:"command,omitempty"`
}

type Action map[string]ActionInner

type Config struct {
	Strimap
}

func New() *Config {
	return &Config{Strimap: Strimap{}}
}

func (c *Config) Load(data []byte) error {
	if err := yaml.Unmarshal(data, &c.Strimap); err != nil {
		return err
	}
	return nil
}

func (c *Config) LoadFile(file string) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	return c.Load(data)
}

func (c *Config) ActionsForEvent(args []interface{}) ([]Action, error) {
	args = append([]interface{}{"events"}, args...)
	acts := c.GetIn(args...)
	if acts == nil {
		return nil, fmt.Errorf(ErrNotFound)
	}
	var actions []Action

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{TagName: "yaml", Result: &actions})
	if err != nil {
		return nil, err
	}

	if err := decoder.Decode(acts); err != nil {
		return nil, err
	}
	return actions, nil
}

func IsNotFound(err error) bool {
	return err.Error() == ErrNotFound
}

func (c *Config) CollectFileEvents() []string {
	var res []string

	evts := c.GetIn("events")
	if evts == nil {
		return nil
	}
	fileEvts := events.ListFileEvents()

	evtsMap, ok := evts.(Strimap)
	if !ok {
		return nil
	}

	for _, evt := range fileEvts {
		e := evtsMap.GetIn(evt.String())
		fileMap, ok := e.(Strimap)
		if !ok {
			continue
		}
		for k := range fileMap {
			res = append(res, k)
		}
	}

	return res
}
