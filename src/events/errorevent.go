package events

type ErrorEvent struct {
	EventBase
	Error error
}

func (e ErrorEvent) Describe() EventTK {
	return EventTK{
		Kind: EKGeneric,
		Type: e.Type,
	}
}

func (e ErrorEvent) Args() []interface{} {
	return []interface{}{string(e.Type), e.Error}
}

func NewErrorEvent(eventType EventType, err error) ErrorEvent {
	return ErrorEvent{EventBase: EventBase{Type: eventType}, Error: err}
}
