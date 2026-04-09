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
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/kube-logging/custom-runner/src/events"
)

type FileWatcher struct {
	watcher *fsnotify.Watcher
	ctx     context.Context
	cancel  context.CancelFunc
	files   map[string]bool
}

func New() *FileWatcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &FileWatcher{ctx: ctx, cancel: cancel, files: make(map[string]bool)}
}

func (f *FileWatcher) Start() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	f.watcher = watcher
	go f.listen()
	return nil
}

func (f *FileWatcher) Stop() error {
	f.cancel()
	defer f.watcher.Close()
	return nil
}

func (f *FileWatcher) listen() {
	for {
		select {
		case <-f.ctx.Done():
			return
		case event, ok := <-f.watcher.Events:
			if !ok {
				f.Stop()
			}
			// info.Println(event, ok)
			if e := f.eventForFile(event); e != nil {
				events.Add(e)
			}
		case err, ok := <-f.watcher.Errors:
			if !ok {
				f.Stop()
			}
			// info.Println(err, ok)
			events.Add(events.OnError(err))
		}
	}
}

func (f *FileWatcher) Add(file string) error {
	path := filepath.Dir(file)
	f.files[file] = true
	return f.watcher.Add(path)
}

func (f *FileWatcher) eventForFile(event fsnotify.Event) events.IEvent {
	file := event.Name
	if _, ok := f.files[file]; ok {
		switch event.Op {
		case fsnotify.Create:
			return events.OnFileCreate(event.Name)
		case fsnotify.Write:
			return events.OnFileWrite(event.Name)
		case fsnotify.Remove:
			return events.OnFileRemove(event.Name)
		case fsnotify.Rename:
			return events.OnFileRename(event.Name)
		case fsnotify.Chmod:
			return events.OnFileChmod(event.Name)
		default:
			return nil
		}
	}
	return nil
}
