// Copyright (c) 2022 Cisco All Rights Reserved.
package events

var DefaultPipe = make(Pipe)

func Add(event IEvent) {
	go func() {
		DefaultPipe <- event
	}()
}
