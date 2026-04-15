// Copyright © 2022 Cisco Systems, Inc. and/or its affiliates
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"errors"
	"os"

	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v3"

	"github.com/kube-logging/custom-runner/src/events"
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
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	return c.Load(data)
}

func (c *Config) ActionsForEvent(args []interface{}) ([]Action, error) {
	args = append([]interface{}{"events"}, args...)
	acts := c.GetIn(args...)
	if acts == nil {
		return nil, errors.New(ErrNotFound)
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
