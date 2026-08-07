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

package events

import "sync"

// Bus is an ordered, unbounded, single-consumer event queue.
//
// Unbounded so publishing never blocks the file watcher: a watcher stalled behind
// a slow action misses inotify events, which for a reloader is a missed reload.
type Bus struct {
	mu     sync.Mutex
	cond   *sync.Cond
	queue  []Event
	closed bool
}

func NewBus() *Bus {
	b := &Bus{}
	b.cond = sync.NewCond(&b.mu)
	return b
}

func (b *Bus) Publish(e Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.queue = append(b.queue, e)
	b.cond.Signal()
}

// Next blocks until an event is available, returning false once the bus is closed
// and drained.
func (b *Bus) Next() (Event, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for len(b.queue) == 0 && !b.closed {
		b.cond.Wait()
	}
	if len(b.queue) == 0 {
		return Event{}, false
	}
	e := b.queue[0]
	b.queue = b.queue[1:]
	return e, true
}

// Close stops accepting events and wakes any waiting consumer once drained.
func (b *Bus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	b.cond.Broadcast()
}
