package events

import (
	ptypes "example.com/gocr/src/process/types"
)

type ApiEvent struct {
	EventBase
	Key ptypes.Key
}

func (a ApiEvent) Args() []interface{} {
	return []interface{}{
		a.Type.String(),
		a.Key,
	}
}

func NewApiEvent(eventType ITEvent, key ptypes.Key) ApiEvent {
	return ApiEvent{EventBase: EventBase{Type: eventType}, Key: key}
}
