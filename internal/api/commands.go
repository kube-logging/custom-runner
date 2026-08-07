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
	"fmt"
	"os"
	"os/exec"

	"github.com/kube-logging/custom-runner/internal/events"
	"github.com/kube-logging/custom-runner/internal/metrics"
	"github.com/kube-logging/custom-runner/internal/process"
)

// Exec starts command under `sh -c`, registered as key.
func (a *API) Exec(key, command string) Result {
	return a.exec(key, []string{"-c", command})
}

func (a *API) exec(key string, args []string) Result {
	cmd := exec.CommandContext(a.procCtx, "sh", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	proc := process.New(key, cmd)

	if err := a.table.Add(proc); err != nil {
		return fail(fmt.Errorf("%w: %s", err, key))
	}

	metrics.RecordExecStart(key)

	if err := proc.Start(); err != nil {
		a.table.Remove(proc)
		proc.Finish()
		metrics.RecordExecError(key)
		a.bus.Publish(events.Error(key, err))

		return fail(err)
	}

	a.bus.Publish(events.New(events.OnExec, key))

	go a.reap(proc)

	return ok(proc)
}

// reap waits for the command, then retires it from the table exactly once.
func (a *API) reap(proc *process.Process) {
	err := proc.Wait()
	a.table.Remove(proc)

	// A deliberate kill exits as "signal: killed"; counting that as a failure would
	// make every restart look like a crash.
	if err != nil && !proc.Killed() {
		metrics.RecordExecError(proc.Key)
		a.bus.Publish(events.Error(proc.Key, err))
	} else {
		metrics.RecordExecSuccess(proc.Key)
	}

	a.bus.Publish(events.New(events.OnFinish, proc.Key))
	proc.Finish()
}

func (a *API) Kill(key string) Result {
	proc, err := a.table.Get(key)
	if err != nil {
		return fail(fmt.Errorf("%w: %s", err, key))
	}

	return a.kill(proc)
}

// kill acts on the fetched process, not on whatever now holds the key.
func (a *API) kill(proc *process.Process) Result {
	if err := proc.Kill(); err != nil {
		return fail(fmt.Errorf("kill %s: %w", proc.Key, err))
	}

	return ok(proc)
}

// Restart kills the process, waits for it to exit, then reruns its original argv.
func (a *API) Restart(key string) Result {
	proc, err := a.table.Get(key)
	if err != nil {
		return fail(fmt.Errorf("%w: %s", err, key))
	}

	args := proc.Args()
	if len(args) == 0 {
		return fail(fmt.Errorf("restart %s: %w", key, process.ErrNotStarted))
	}

	if res := a.kill(proc); res.Error != "" {
		return res
	}
	<-proc.Done

	// args[0] is the "sh" argv[0]; exec re-adds it.
	return a.exec(key, args[1:])
}

func (a *API) Get(key string) Result {
	proc, err := a.table.Get(key)
	if err != nil {
		return fail(fmt.Errorf("%w: %s", err, key))
	}

	return ok(proc)
}

func (a *API) List() Result {
	return ok(a.table.List())
}

func (a *API) Config() Result {
	return ok(a.cfg)
}

// Exit unwinds the runner; the server drains this request before shutting down.
func (a *API) Exit() Result {
	a.shutdown()
	return ok(nil)
}
