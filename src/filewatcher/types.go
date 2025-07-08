// Copyright (c) 2022 Cisco All Rights Reserved.
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
