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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"

	"gopkg.in/yaml.v3"

	"github.com/kube-logging/custom-runner/internal/events"
)

// Spec is the payload of a single action. Command is only meaningful for exec.
type Spec struct {
	Key     string `json:"key,omitempty"     yaml:"key,omitempty"`
	Command string `json:"command,omitempty" yaml:"command,omitempty"`
}

// Action verbs, also used to key the API command table.
const (
	ActionExec    = "exec"
	ActionKill    = "kill"
	ActionRestart = "restart"
	ActionGet     = "get"
	ActionList    = "list"
	ActionExit    = "exit"
	ActionConfig  = "config"
)

var actionVerbs = map[string]struct{}{
	ActionExec: {}, ActionKill: {}, ActionRestart: {},
	ActionGet: {}, ActionList: {}, ActionExit: {}, ActionConfig: {},
}

// Action maps an API command name ("exec", "restart", …) to its payload. A well
// formed action carries exactly one entry.
type Action map[string]Spec

// StartsProcess reports whether the action brings a new process into existence,
// so callers can tell it apart from a read like get.
func (a Action) StartsProcess() bool {
	_, exec := a[ActionExec]
	_, restart := a[ActionRestart]

	return exec || restart
}

// Events binds each event kind to the actions it triggers. List-valued kinds fire
// unconditionally; map-valued kinds are keyed by process key or watched path.
type Events struct {
	OnStart []Action `json:"onStart,omitempty" yaml:"onStart,omitempty"`
	OnExit  []Action `json:"onExit,omitempty"  yaml:"onExit,omitempty"`

	OnExec   map[string][]Action `json:"onExec,omitempty"   yaml:"onExec,omitempty"`
	OnFinish map[string][]Action `json:"onFinish,omitempty" yaml:"onFinish,omitempty"`
	OnError  map[string][]Action `json:"onError,omitempty"  yaml:"onError,omitempty"`

	OnFileCreate map[string][]Action `json:"onFileCreate,omitempty" yaml:"onFileCreate,omitempty"`
	OnFileWrite  map[string][]Action `json:"onFileWrite,omitempty"  yaml:"onFileWrite,omitempty"`
	OnFileRemove map[string][]Action `json:"onFileRemove,omitempty" yaml:"onFileRemove,omitempty"`
	OnFileRename map[string][]Action `json:"onFileRename,omitempty" yaml:"onFileRename,omitempty"`
	OnFileChmod  map[string][]Action `json:"onFileChmod,omitempty"  yaml:"onFileChmod,omitempty"`
}

type Config struct {
	Events Events `json:"events" yaml:"events,omitempty"`
}

func New() *Config {
	return &Config{}
}

// LoadYAML merges YAML over the current config, rejecting unknown keys so a typo
// fails at startup instead of silently never firing.
func (c *Config) LoadYAML(data []byte) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	if err := dec.Decode(c); err != nil {
		// An empty or comment-only file is a valid "nothing configured yet", not a
		// reason to crashloop a sidecar whose ConfigMap key happens to be blank.
		if errors.Is(err, io.EOF) {
			return nil
		}
		return fmt.Errorf("parse yaml config: %w", err)
	}

	if err := dec.Decode(&Config{}); !errors.Is(err, io.EOF) {
		return errors.New("parse yaml config: multi-document config is not supported")
	}

	return c.validate()
}

func (c *Config) LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file %q: %w", path, err)
	}
	return c.LoadYAML(data)
}

// LoadJSON merges JSON over the current config, with the same strictness as YAML.
func (c *Config) LoadJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	if err := dec.Decode(c); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return fmt.Errorf("parse json config: %w", err)
	}

	if dec.More() {
		return errors.New("parse json config: unexpected trailing content")
	}

	return c.validate()
}

// validate rejects an action carrying anything but exactly one verb; a map would
// otherwise let dispatch pick one at random.
func (c *Config) validate() error {
	kinds := append([]events.Kind{
		events.OnStart, events.OnExit,
		events.OnExec, events.OnFinish, events.OnError,
	}, events.FileKinds()...)

	for _, kind := range kinds {
		if err := validateActions(string(kind), "", c.ActionsFor(kind, "")); err != nil {
			return err
		}
		for subject, actions := range c.keyed(kind) {
			if err := validateActions(string(kind), subject, actions); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateActions(kind, subject string, actions []Action) error {
	where := kind
	if subject != "" {
		where = kind + "." + subject
	}

	for i, action := range actions {
		if len(action) != 1 {
			return fmt.Errorf("events.%s[%d]: an action must carry exactly one command, got %d", where, i, len(action))
		}
		for verb := range action {
			if _, ok := actionVerbs[verb]; !ok {
				return fmt.Errorf("events.%s[%d]: unknown command %q", where, i, verb)
			}
		}
	}

	return nil
}

// ActionsFor returns the actions bound to an event. Subject selects the entry for
// keyed kinds and is ignored for onStart and onExit.
func (c *Config) ActionsFor(kind events.Kind, subject string) []Action {
	switch kind {
	case events.OnStart:
		return c.Events.OnStart
	case events.OnExit:
		return c.Events.OnExit
	default:
		return c.keyed(kind)[subject]
	}
}

func (c *Config) keyed(kind events.Kind) map[string][]Action {
	switch kind {
	case events.OnExec:
		return c.Events.OnExec
	case events.OnFinish:
		return c.Events.OnFinish
	case events.OnError:
		return c.Events.OnError
	case events.OnFileCreate:
		return c.Events.OnFileCreate
	case events.OnFileWrite:
		return c.Events.OnFileWrite
	case events.OnFileRemove:
		return c.Events.OnFileRemove
	case events.OnFileRename:
		return c.Events.OnFileRename
	case events.OnFileChmod:
		return c.Events.OnFileChmod
	default:
		return nil
	}
}

// WatchPaths lists every path referenced by a file event, deduplicated.
func (c *Config) WatchPaths() []string {
	seen := make(map[string]struct{})
	for _, kind := range events.FileKinds() {
		for path := range c.keyed(kind) {
			seen[path] = struct{}{}
		}
	}

	return slices.Collect(maps.Keys(seen))
}
