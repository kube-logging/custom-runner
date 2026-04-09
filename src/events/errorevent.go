// Copyright (c) 2022 Cisco All Rights Reserved.
package events

import ptypes "github.com/kube-logging/custom-runner/src/process/types"

type ErrorEvent struct {
	EventBase
	Key   ptypes.Key
	Error error
}

func (e ErrorEvent) Args() []interface{} {
	return []interface{}{e.Type.String(), e.Error}
}

func NewErrorEvent(eventType ITEvent, err error) ErrorEvent {
	return ErrorEvent{EventBase: EventBase{Type: eventType}, Error: err}
}

func NewErrorEventWithKey(eventType ITEvent, key ptypes.Key, err error) ErrorEvent {
	return ErrorEvent{EventBase: EventBase{Type: eventType}, Key: key, Error: err}
}
