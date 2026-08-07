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
		want  []string
	}{
		{
			name:  "exact match on a plain file",
			watch: []string{"/etc/app/conf"},
			event: "/etc/app/conf",
			want:  []string{"/etc/app/conf"},
		},
		{
			name:  "unrelated sibling is ignored",
			watch: []string{"/etc/app/conf"},
			event: "/etc/app/other",
			want:  nil,
		},
		{
			name:  "a ..data swap fires every watched file in that directory",
			watch: []string{"/etc/app/conf", "/etc/app/extra"},
			event: "/etc/app/..data",
			want:  []string{"/etc/app/conf", "/etc/app/extra"},
		},
		{
			name:  "a ..data swap elsewhere is ignored",
			watch: []string{"/etc/app/conf"},
			event: "/etc/other/..data",
			want:  nil,
		},
		{
			name:  "watching ..data directly fires exactly once",
			watch: []string{"/etc/app/..data"},
			event: "/etc/app/..data",
			want:  []string{"/etc/app/..data"},
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

			got := w.subjects(tc.event)
			slices.Sort(got)
			want := slices.Clone(tc.want)
			slices.Sort(want)

			if !slices.Equal(got, want) {
				t.Errorf("subjects(%q) = %v, want %v", tc.event, got, want)
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
	case <-time.After(5 * time.Second):
		t.Fatal("no event published for a ConfigMap swap; the watched file would never reload")
	}
}
