package events

type GenericeEvent struct {
	EventBase
}

func (g GenericeEvent) Describe() EventTK {
	return EventTK{
		Kind: EKGeneric,
		Type: g.Type,
	}
}

func (g GenericeEvent) Args() []interface{} {
	return []interface{}{string(g.Type)}
}

func NewGenericEvent(eventType EventType) GenericeEvent {
	return GenericeEvent{EventBase: EventBase{Type: eventType}}
}
