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

package process

import (
	"encoding/json"
	"errors"
	"os/exec"
	"slices"
	"strings"
	"sync"
)

var (
	ErrAlreadyRunning = errors.New("process already running")
	ErrNotFound       = errors.New("process not found")
	ErrNotStarted     = errors.New("process has not started yet")
)

// Process is a supervised command.
type Process struct {
	Key  string
	Cmd  *exec.Cmd
	Done chan struct{}

	mu      sync.Mutex
	pid     int
	args    []string
	started bool
	exited  bool
	killed  bool
}

func New(key string, cmd *exec.Cmd) *Process {
	return &Process{Key: key, Cmd: cmd, Done: make(chan struct{})}
}

// Start launches the command, snapshotting what the API reports so readers never
// touch live exec.Cmd state.
func (p *Process) Start() error {
	Isolate(p.Cmd)

	if err := p.Cmd.Start(); err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.started = true
	p.pid = p.Cmd.Process.Pid
	p.args = slices.Clone(p.Cmd.Args)

	return nil
}

// Wait blocks until the command exits, marking it reaped.
func (p *Process) Wait() error {
	err := p.Cmd.Wait()

	p.mu.Lock()
	p.exited = true
	p.mu.Unlock()

	return err
}

// Kill terminates the process group, recording the exit as deliberate.
func (p *Process) Kill() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started {
		return ErrNotStarted
	}
	// Wait released the pid, so signaling now could hit a recycled group.
	if p.exited {
		return nil
	}
	if err := KillGroup(p.Cmd); err != nil {
		return err
	}
	p.killed = true

	return nil
}

// Killed reports whether termination was requested rather than a crash.
func (p *Process) Killed() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.killed
}

// Finish releases anything waiting on the process.
func (p *Process) Finish() {
	close(p.Done)
}

// Args returns the argv the command was started with, or nil before Start.
func (p *Process) Args() []string {
	p.mu.Lock()
	defer p.mu.Unlock()

	return slices.Clone(p.args)
}

// MarshalJSON projects onto serializable fields; exec.Cmd carries func members
// that encoding/json rejects outright.
func (p *Process) MarshalJSON() ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	return json.Marshal(struct {
		Key  string   `json:"key"`
		Args []string `json:"args,omitempty"`
		Pid  int      `json:"pid,omitempty"`
	}{Key: p.Key, Args: p.args, Pid: p.pid})
}

// Table is a concurrency-safe registry of running processes.
type Table struct {
	mu    sync.Mutex
	procs map[string]*Process
}

func NewTable() *Table {
	return &Table{procs: make(map[string]*Process)}
}

// Add registers a process, refusing to replace a live entry under the same key.
func (t *Table) Add(p *Process) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, ok := t.procs[p.Key]; ok {
		return ErrAlreadyRunning
	}
	t.procs[p.Key] = p

	return nil
}

func (t *Table) Get(key string) (*Process, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	p, ok := t.procs[key]
	if !ok {
		return nil, ErrNotFound
	}

	return p, nil
}

// Remove retires an entry only if it is still the same instance, so a stale
// reaper cannot evict its replacement.
func (t *Table) Remove(p *Process) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if current, ok := t.procs[p.Key]; ok && current == p {
		delete(t.procs, p.Key)
	}
}

// List returns the live processes ordered by key, so output is deterministic.
func (t *Table) List() []*Process {
	t.mu.Lock()
	defer t.mu.Unlock()

	out := make([]*Process, 0, len(t.procs))
	for _, p := range t.procs {
		out = append(out, p)
	}
	slices.SortFunc(out, func(a, b *Process) int { return strings.Compare(a.Key, b.Key) })

	return out
}
