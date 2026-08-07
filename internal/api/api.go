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

package api

import (
	"context"
	"fmt"

	"github.com/kube-logging/custom-runner/internal/config"
	"github.com/kube-logging/custom-runner/internal/events"
	"github.com/kube-logging/custom-runner/internal/process"
)

// Command is one API verb. key and args come from the request path and body.
type Command func(key string, args []byte) Result

// Result is the API's wire shape. It is part of the public contract — the
// logging-operator sidecars parse it.
type Result struct {
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
	Response any    `json:"response,omitempty"`
}

func ok(response any) Result { return Result{Success: true, Response: response} }
func fail(err error) Result  { return Result{Error: err.Error()} }

// API executes runner verbs against a process table.
type API struct {
	// procCtx bounds supervised children to the runner's lifetime. It is
	// deliberately not a request context — a process outlives the call that made it.
	procCtx  context.Context
	shutdown context.CancelFunc

	table    *process.Table
	bus      *events.Bus
	cfg      *config.Config
	commands map[string]Command
}

// New builds an API.
func New(procCtx context.Context, shutdown context.CancelFunc, table *process.Table, bus *events.Bus, cfg *config.Config) *API {
	a := &API{procCtx: procCtx, shutdown: shutdown, table: table, bus: bus, cfg: cfg}

	a.commands = map[string]Command{
		"exec":    func(key string, args []byte) Result { return a.Exec(key, string(args)) },
		"kill":    func(key string, _ []byte) Result { return a.Kill(key) },
		"restart": func(key string, _ []byte) Result { return a.Restart(key) },
		"get":     func(key string, _ []byte) Result { return a.Get(key) },
		"list":    func(string, []byte) Result { return a.List() },
		"exit":    func(string, []byte) Result { return a.Exit() },
		"config":  func(string, []byte) Result { return a.Config() },
	}

	return a
}

func (a *API) Command(name string) (Command, bool) {
	cmd, found := a.commands[name]
	return cmd, found
}

func (a *API) RunAction(action config.Action) Result {
	for name, spec := range action {
		cmd, found := a.Command(name)
		if !found {
			return fail(fmt.Errorf("unknown API command: %s", name))
		}
		return cmd(spec.Key, []byte(spec.Command))
	}

	return Result{}
}

func (a *API) RunActions(actions []config.Action) []Result {
	results := make([]Result, 0, len(actions))
	for _, action := range actions {
		results = append(results, a.RunAction(action))
	}

	return results
}
