package events

import (
	ptypes "example.com/gocr/src/process/types"
)

type ApiEvent struct {
	EventBase
	// Process *types.Process
	Key ptypes.Key
}

func (a ApiEvent) Describe() EventTK {
	return EventTK{
		Kind: EKApi,
		Type: a.Type,
	}

}

func (a ApiEvent) Args() []interface{} {
	return []interface{}{
		string(a.Type),
		a.Key,
	}
}

func NewApiEvent(eventType EventType, key ptypes.Key) ApiEvent {
	return ApiEvent{EventBase: EventBase{Type: eventType}, Key: key}
}
