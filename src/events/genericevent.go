package events

type GenericeEvent struct {
	EventBase
}

func (g GenericeEvent) Args() []interface{} {
	return []interface{}{g.Type.String()}
}

func NewGenericEvent(eventType ITEvent) GenericeEvent {
	return GenericeEvent{EventBase: EventBase{Type: eventType}}
}
