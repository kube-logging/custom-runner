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

import (
	"testing"
)

func TestBusPreservesPublishOrder(t *testing.T) {
	bus := NewBus()

	const count = 1000
	for i := range count {
		bus.Publish(Event{Kind: OnExec, Subject: string(rune('a' + i%26)), Err: nil})
	}
	bus.Close()

	var got []string
	for {
		e, open := bus.Next()
		if !open {
			break
		}
		got = append(got, e.Subject)
	}

	if len(got) != count {
		t.Fatalf("drained %d events, want %d", len(got), count)
	}
	for i, subject := range got {
		if want := string(rune('a' + i%26)); subject != want {
			t.Fatalf("event %d = %q, want %q — ordering was not preserved", i, subject, want)
		}
	}
}

// A consumer dispatching an action may publish from inside its own handler; that
// must never block, or the runner deadlocks on itself.
func TestBusPublishFromConsumerDoesNotBlock(t *testing.T) {
	bus := NewBus()
	bus.Publish(Event{Kind: OnStart})

	seen := 0
	for seen < 50 {
		e, open := bus.Next()
		if !open {
			t.Fatal("bus closed early")
		}
		seen++
		if e.Kind == OnStart && seen < 50 {
			bus.Publish(Event{Kind: OnExec, Subject: "chained"})
			bus.Publish(Event{Kind: OnStart})
		}
	}
}

func TestBusNextReturnsFalseOnceDrained(t *testing.T) {
	bus := NewBus()
	bus.Publish(Event{Kind: OnStart})
	bus.Close()

	if _, open := bus.Next(); !open {
		t.Fatal("queued event should still drain after Close")
	}
	if _, open := bus.Next(); open {
		t.Fatal("Next should report closed once drained")
	}
}

func TestBusPublishAfterCloseIsDropped(t *testing.T) {
	bus := NewBus()
	bus.Close()
	bus.Publish(Event{Kind: OnStart})

	if _, open := bus.Next(); open {
		t.Fatal("publish after close should be dropped")
	}
}
