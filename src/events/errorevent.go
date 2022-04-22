package events

type ErrorEvent struct {
	EventBase
	Error error
}

func (e ErrorEvent) Args() []interface{} {
	return []interface{}{e.Type.String(), e.Error}
}

func NewErrorEvent(eventType ITEvent, err error) ErrorEvent {
	return ErrorEvent{EventBase: EventBase{Type: eventType}, Error: err}
}
