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

package filewatcher

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/kube-logging/custom-runner/internal/events"
)

// subjects is the routing decision: which configured paths a filesystem event
// should fire for. The ..data cases are the Kubernetes ConfigMap/Secret swap.
func TestSubjects(t *testing.T) {
	testCases := []struct {
		name  string
		watch []string
		event string
		kind  events.Kind
		want  []subject
	}{
		{
			name:  "exact match keeps the native kind",
			watch: []string{"/etc/app/conf"},
			event: "/etc/app/conf",
			kind:  events.OnFileWrite,
			want:  []subject{{"/etc/app/conf", events.OnFileWrite}},
		},
		{
			name:  "unrelated sibling is ignored",
			watch: []string{"/etc/app/conf"},
			event: "/etc/app/other",
			kind:  events.OnFileWrite,
			want:  nil,
		},
		{
			// kubelet creates ..data; the configured file is never written, so a
			// raw create would leave onFileWrite silent.
			name:  "a ..data swap reaches siblings as a write",
			watch: []string{"/etc/app/conf", "/etc/app/extra"},
			event: "/etc/app/..data",
			kind:  events.OnFileCreate,
			want: []subject{
				{"/etc/app/conf", events.OnFileWrite},
				{"/etc/app/extra", events.OnFileWrite},
			},
		},
		{
			name:  "a ..data swap elsewhere is ignored",
			watch: []string{"/etc/app/conf"},
			event: "/etc/other/..data",
			kind:  events.OnFileCreate,
			want:  nil,
		},
		{
			name:  "watching ..data directly keeps the native kind",
			watch: []string{"/etc/app/..data"},
			event: "/etc/app/..data",
			kind:  events.OnFileCreate,
			want:  []subject{{"/etc/app/..data", events.OnFileCreate}},
		},
		{
			name:  "..data and a sibling both fire, each with its own kind",
			watch: []string{"/etc/app/..data", "/etc/app/conf"},
			event: "/etc/app/..data",
			kind:  events.OnFileCreate,
			want: []subject{
				{"/etc/app/..data", events.OnFileCreate},
				{"/etc/app/conf", events.OnFileWrite},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := &Watcher{paths: map[string]struct{}{}, byDir: map[string][]string{}}
			for _, p := range tc.watch {
				w.paths[p] = struct{}{}
				dir := filepath.Dir(p)
				w.byDir[dir] = append(w.byDir[dir], p)
			}

			got := w.subjects(tc.event, tc.kind)
			slices.SortFunc(got, func(a, b subject) int { return strings.Compare(a.path, b.path) })
			want := slices.Clone(tc.want)
			slices.SortFunc(want, func(a, b subject) int { return strings.Compare(a.path, b.path) })

			if !slices.Equal(got, want) {
				t.Errorf("subjects(%q, %q) = %v, want %v", tc.event, tc.kind, got, want)
			}
		})
	}
}

// End to end against the real filesystem: perform the swap kubelet performs and
// assert the runner hears about the user-visible filename.
func TestKubernetesConfigMapSwapPublishes(t *testing.T) {
	dir := t.TempDir()
	conf := filepath.Join(dir, "app.conf")

	bus := events.NewBus()
	w, err := New(bus)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	t.Cleanup(func() { _ = w.Close() })

	if err := w.Add(conf); err != nil {
		t.Fatalf("add: %v", err)
	}

	go w.Run(t.Context())

	// What the kubelet atomic writer does: new timestamped dir, then swap ..data.
	payload := filepath.Join(dir, "..2026_01_01")
	if err := os.MkdirAll(payload, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(payload, "app.conf"), []byte("v2"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.Symlink("..2026_01_01", filepath.Join(dir, "..data")); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	got := make(chan events.Event, 1)
	go func() {
		if e, open := bus.Next(); open {
			got <- e
		}
	}()

	select {
	case e := <-got:
		if e.Subject != conf {
			t.Errorf("event subject = %q, want %q", e.Subject, conf)
		}
		if e.Kind != events.OnFileWrite {
			t.Errorf("event kind = %q, want %q — onFileWrite on the mounted path would never fire", e.Kind, events.OnFileWrite)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no event published for a ConfigMap swap; the watched file would never reload")
	}
}
