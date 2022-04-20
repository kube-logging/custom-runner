package events

type FileEvent struct {
	EventBase
	File string
}

func (a FileEvent) Describe() EventTK {
	return EventTK{
		Kind: EKFile,
		Type: a.Type,
	}

}

func (a FileEvent) Args() []interface{} {
	return []interface{}{
		string(a.Type),
		a.File,
	}
}

func NewFileEvent(eventType EventType, file string) FileEvent {
	return FileEvent{EventBase: EventBase{Type: eventType}, File: file}
}
