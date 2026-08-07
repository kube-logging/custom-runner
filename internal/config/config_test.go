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

package config

import (
	"slices"
	"testing"

	"github.com/kube-logging/custom-runner/internal/events"
)

const sample = `
events:
  onStart:
    - exec:
        key: boot
        command: echo boot
  onExec:
    boot:
      - exec:
          key: next
          command: echo next
  onFileWrite:
    /etc/app/conf:
      - restart:
          key: app
  onFileCreate:
    /etc/app/conf:
      - exec:
          key: notify
          command: echo created
    /etc/other/..data:
      - restart:
          key: app
`

func TestActionsForDispatch(t *testing.T) {
	cfg := New()
	if err := cfg.LoadYAML([]byte(sample)); err != nil {
		t.Fatalf("load: %v", err)
	}

	testCases := []struct {
		name    string
		kind    events.Kind
		subject string
		want    int
	}{
		{"unkeyed onStart ignores subject", events.OnStart, "anything", 1},
		{"lifecycle kind matches its key", events.OnExec, "boot", 1},
		{"lifecycle kind misses other keys", events.OnExec, "other", 0},
		{"file kind matches its path", events.OnFileWrite, "/etc/app/conf", 1},
		{"same path under a different kind is distinct", events.OnFileCreate, "/etc/app/conf", 1},
		{"unconfigured kind yields nothing", events.OnFinish, "boot", 0},
		{"unconfigured path yields nothing", events.OnFileWrite, "/nope", 0},
		{"onExit absent yields nothing", events.OnExit, "", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := len(cfg.ActionsFor(tc.kind, tc.subject)); got != tc.want {
				t.Errorf("ActionsFor(%q, %q) = %d actions, want %d", tc.kind, tc.subject, got, tc.want)
			}
		})
	}
}

func TestWatchPathsDeduplicatesAcrossKinds(t *testing.T) {
	cfg := New()
	if err := cfg.LoadYAML([]byte(sample)); err != nil {
		t.Fatalf("load: %v", err)
	}

	got := cfg.WatchPaths()
	slices.Sort(got)
	want := []string{"/etc/app/conf", "/etc/other/..data"}

	if !slices.Equal(got, want) {
		t.Errorf("WatchPaths() = %v, want %v", got, want)
	}
}

// A misspelled event name must fail at load, not silently never fire.
func TestLoadRejectsUnknownEventName(t *testing.T) {
	cfg := New()
	err := cfg.LoadYAML([]byte("events:\n  onFileWirte:\n    /tmp/x: []\n"))

	if err == nil {
		t.Fatal("expected a misspelled event key to be rejected")
	}
}

func TestLoadJSONRejectsUnknownEventName(t *testing.T) {
	cfg := New()
	err := cfg.LoadJSON([]byte(`{"events":{"onNonsense":{"/tmp/x":[]}}}`))

	if err == nil {
		t.Fatal("expected a misspelled event key to be rejected")
	}
}

func TestLoadJSONMergesOverYAML(t *testing.T) {
	cfg := New()
	if err := cfg.LoadYAML([]byte(sample)); err != nil {
		t.Fatalf("load yaml: %v", err)
	}
	if err := cfg.LoadJSON([]byte(`{"events":{"onStart":[{"exec":{"key":"override","command":"echo override"}}]}}`)); err != nil {
		t.Fatalf("load json: %v", err)
	}

	actions := cfg.ActionsFor(events.OnStart, "")
	if len(actions) != 1 {
		t.Fatalf("got %d onStart actions, want 1", len(actions))
	}
	if got := actions[0]["exec"].Key; got != "override" {
		t.Errorf("onStart key = %q, want %q", got, "override")
	}

	// Untouched sections survive the merge.
	if len(cfg.ActionsFor(events.OnExec, "boot")) != 1 {
		t.Error("onExec should survive a JSON merge that only sets onStart")
	}
}

// An empty ConfigMap key must be a no-op, not a crashloop.
func TestLoadAcceptsEmptyDocuments(t *testing.T) {
	if err := New().LoadYAML([]byte("# nothing configured yet\n")); err != nil {
		t.Errorf("LoadYAML(comment only) = %v, want nil", err)
	}
}

func TestLoadJSONAcceptsEmptyDocument(t *testing.T) {
	if err := New().LoadJSON(nil); err != nil {
		t.Errorf("LoadJSON(nil) = %v, want nil", err)
	}
}

// A second document must be rejected, not silently dropped.
func TestLoadRejectsMultiDocument(t *testing.T) {
	err := New().LoadYAML([]byte("events:\n  onStart: []\n---\nevents:\n  onExit: []\n"))
	if err == nil {
		t.Fatal("expected a multi-document config to be rejected rather than silently truncated")
	}
}

// A map accepts two verbs; dispatch would then pick one at random.
func TestLoadRejectsMultiVerbAction(t *testing.T) {
	err := New().LoadYAML([]byte(`
events:
  onStart:
    - exec:
        key: a
        command: echo a
      kill:
        key: b
`))
	if err == nil {
		t.Fatal("expected an action with two verbs to be rejected")
	}
}

func TestLoadRejectsEmptyAction(t *testing.T) {
	if err := New().LoadYAML([]byte("events:\n  onStart:\n    - {}\n")); err == nil {
		t.Fatal("expected an action with no verb to be rejected")
	}
}

// The strict loader used to check only that an action had one key, so a
// misspelled verb loaded fine and failed later when the event fired.
func TestLoadRejectsUnknownActionVerb(t *testing.T) {
	err := New().LoadYAML([]byte("events:\n  onStart:\n    - exce:\n        key: a\n"))

	if err == nil {
		t.Fatal("expected a misspelled action verb to be rejected at load")
	}
}

func TestLoadAcceptsEveryRegisteredVerb(t *testing.T) {
	for _, verb := range []string{
		ActionExec, ActionKill, ActionRestart,
		ActionGet, ActionList, ActionExit, ActionConfig,
	} {
		if err := New().LoadYAML([]byte("events:\n  onStart:\n    - " + verb + ":\n        key: a\n")); err != nil {
			t.Errorf("verb %q rejected: %v", verb, err)
		}
	}
}

func TestLoadJSONRejectsTrailingContent(t *testing.T) {
	if err := New().LoadJSON([]byte(`{"events":{}} {"events":{}}`)); err == nil {
		t.Fatal("expected trailing JSON to be rejected, as YAML's second document is")
	}
}

func TestStartsProcess(t *testing.T) {
	testCases := map[string]bool{
		ActionExec: true, ActionRestart: true,
		ActionGet: false, ActionKill: false, ActionList: false,
	}

	for verb, want := range testCases {
		if got := (Action{verb: Spec{}}).StartsProcess(); got != want {
			t.Errorf("%q.StartsProcess() = %v, want %v", verb, got, want)
		}
	}
}
