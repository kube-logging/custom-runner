// Copyright (c) 2022 Cisco All Rights Reserved.
package events

type FileEvent struct {
	EventBase
	File string
}

func (a FileEvent) Args() []interface{} {
	return []interface{}{
		a.Type.String(),
		a.File,
	}
}

func NewFileEvent(eventType ITEvent, file string) FileEvent {
	return FileEvent{EventBase: EventBase{Type: eventType}, File: file}
}
