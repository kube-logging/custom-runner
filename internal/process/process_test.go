// Copyright © 2026 Kube logging authors
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
	"sync"
	"testing"
)

func sleeper(t *testing.T) *Process {
	t.Helper()

	p := New("sleeper", exec.CommandContext(t.Context(), "sh", "-c", "sleep 30"))
	if err := p.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() {
		_ = p.Kill()
		_ = p.Cmd.Wait()
	})

	return p
}

// exec.Cmd carries func fields, so a naive marshal fails outright.
func TestProcessMarshalJSONBeforeStart(t *testing.T) {
	got, err := json.Marshal(New("bare", exec.CommandContext(t.Context(), "sh", "-c", "date")))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// argv and pid are only meaningful once Start has run.
	if want := `{"key":"bare"}`; string(got) != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestProcessMarshalJSONAfterStart(t *testing.T) {
	p := sleeper(t)

	got, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var out struct {
		Key  string   `json:"key"`
		Args []string `json:"args"`
		Pid  int      `json:"pid"`
	}
	if err := json.Unmarshal(got, &out); err != nil {
		t.Fatalf("unmarshal %s: %v", got, err)
	}
	if out.Pid != p.Cmd.Process.Pid {
		t.Errorf("pid = %d, want %d", out.Pid, p.Cmd.Process.Pid)
	}
	if len(out.Args) != 3 {
		t.Errorf("args = %v, want the full argv", out.Args)
	}
}

// A concurrent /list marshals the process while Start writes Cmd.Process.
func TestProcessMarshalDoesNotRaceStart(t *testing.T) {
	for range 50 {
		p := New("racer", exec.CommandContext(t.Context(), "sh", "-c", "true"))

		var wg sync.WaitGroup
		wg.Go(func() { _ = p.Start() })
		wg.Go(func() { _, _ = json.Marshal(p) })
		wg.Wait()

		_ = p.Cmd.Wait()
	}
}

// A process is registered before it starts, so there is a window with no pid.
func TestKillBeforeStartIsRejected(t *testing.T) {
	p := New("unstarted", exec.CommandContext(t.Context(), "sh", "-c", "true"))

	if err := p.Kill(); !errors.Is(err, ErrNotStarted) {
		t.Errorf("Kill before Start = %v, want ErrNotStarted", err)
	}
	if p.Killed() {
		t.Error("a rejected kill must not mark the process as deliberately killed")
	}
}

func TestKillMarksDeliberateTermination(t *testing.T) {
	p := sleeper(t)

	if p.Killed() {
		t.Fatal("process reported killed before any kill")
	}
	if err := p.Kill(); err != nil {
		t.Fatalf("kill: %v", err)
	}
	if !p.Killed() {
		t.Error("Killed() = false after Kill; a restart would be reported as a crash")
	}
}

// After Wait the pid is released to the kernel and could be recycled, so killing
// must become a no-op rather than signaling a stranger's process group.
func TestKillAfterExitIsANoop(t *testing.T) {
	p := New("quick", exec.CommandContext(t.Context(), "sh", "-c", "true"))
	if err := p.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	_ = p.Wait()

	if err := p.Kill(); err != nil {
		t.Errorf("Kill after exit = %v, want nil", err)
	}
	if p.Killed() {
		t.Error("a no-op kill must not mark the exit as deliberate")
	}
}

// The key is claimed before the command starts.
func TestTableAddRejectsDuplicateKey(t *testing.T) {
	table := NewTable()

	if err := table.Add(New("dupe", nil)); err != nil {
		t.Fatalf("first add: %v", err)
	}
	if err := table.Add(New("dupe", nil)); !errors.Is(err, ErrAlreadyRunning) {
		t.Errorf("second add = %v, want ErrAlreadyRunning", err)
	}
}

// A stale reaper must not evict its replacement.
func TestTableRemoveOnlyEvictsTheSameInstance(t *testing.T) {
	table := NewTable()
	old := New("app", nil)
	replacement := New("app", nil)

	if err := table.Add(old); err != nil {
		t.Fatalf("add: %v", err)
	}
	table.Remove(old)
	if err := table.Add(replacement); err != nil {
		t.Fatalf("re-add: %v", err)
	}

	table.Remove(old)

	got, err := table.Get("app")
	if err != nil {
		t.Fatalf("replacement was evicted by the stale reaper: %v", err)
	}
	if got != replacement {
		t.Error("table holds the wrong instance")
	}
}
