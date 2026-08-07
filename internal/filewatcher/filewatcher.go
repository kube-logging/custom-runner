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

package filewatcher

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"

	"github.com/kube-logging/custom-runner/internal/events"
	"github.com/kube-logging/custom-runner/internal/metrics"
)

// kubeDataDir is the symlink kubelet swaps atomically when a mounted ConfigMap or
// Secret changes. The user-visible filenames are never touched.
const kubeDataDir = "..data"

// Watcher translates fsnotify events into runner events for a fixed set of paths.
type Watcher struct {
	watcher *fsnotify.Watcher
	bus     *events.Bus

	mu    sync.RWMutex
	paths map[string]struct{}
	byDir map[string][]string
}

func New(bus *events.Bus) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create file watcher: %w", err)
	}

	return &Watcher{
		watcher: w,
		bus:     bus,
		paths:   make(map[string]struct{}),
		byDir:   make(map[string][]string),
	}, nil
}

// Add registers interest in an exact path, watching its parent directory.
func (w *Watcher) Add(path string) error {
	dir := filepath.Dir(path)

	if err := w.watcher.Add(dir); err != nil {
		return fmt.Errorf("watch %s: %w", dir, err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	w.paths[path] = struct{}{}
	w.byDir[dir] = append(w.byDir[dir], path)

	return nil
}

func (w *Watcher) Close() error {
	return w.watcher.Close()
}

// Run pumps events until ctx is canceled or the watcher closes.
func (w *Watcher) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case event, open := <-w.watcher.Events:
			if !open {
				return
			}
			kind, ok := kindOf(event.Op)
			if !ok {
				continue
			}
			for _, path := range w.subjects(event.Name) {
				w.bus.Publish(events.New(kind, path))
			}

		case err, open := <-w.watcher.Errors:
			if !open {
				return
			}
			metrics.RecordWatcherError()
			w.bus.Publish(events.Error("", err))
		}
	}
}

// subjects maps a filesystem event onto the configured paths it should fire for.
func (w *Watcher) subjects(name string) []string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if _, ok := w.paths[name]; ok {
		return []string{name}
	}

	if filepath.Base(name) != kubeDataDir {
		return nil
	}

	// Excluded if configured directly: the exact match above already fired it.
	siblings := w.byDir[filepath.Dir(name)]
	out := make([]string, 0, len(siblings))
	for _, path := range siblings {
		if path != name {
			out = append(out, path)
		}
	}

	return out
}

func kindOf(op fsnotify.Op) (events.Kind, bool) {
	switch {
	case op.Has(fsnotify.Create):
		return events.OnFileCreate, true
	case op.Has(fsnotify.Write):
		return events.OnFileWrite, true
	case op.Has(fsnotify.Remove):
		return events.OnFileRemove, true
	case op.Has(fsnotify.Rename):
		return events.OnFileRename, true
	case op.Has(fsnotify.Chmod):
		return events.OnFileChmod, true
	default:
		return "", false
	}
}
