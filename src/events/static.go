package events

var DefaultPipe = make(Pipe)

func Add(event IEvent) {
	go func() {
		DefaultPipe <- event
	}()
}
