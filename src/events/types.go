package events

import (
	ptypes "example.com/gocr/src/process/types"
)

type ITEvent interface {
	String() string
}

type TEvent int
type TGenericEvent TEvent

func (g TGenericEvent) String() string {
	return EventNames[g]
}

type TApiEvent TEvent

func (a TApiEvent) String() string {
	return EventNames[a]
}

type TFileEvent TEvent

func (f TFileEvent) String() string {
	return EventNames[f]
}

const (
	EOnStart TGenericEvent = iota
	EOnExit

	FirstGenericEvent TGenericEvent = EOnStart
	LastGenericEvent  TGenericEvent = EOnExit
)

const (
	EOnExec TApiEvent = iota + 100
	EOnFinish
	EOnError

	FirstApiEvent TApiEvent = EOnExec
	LastApiEvent  TApiEvent = EOnError
)

const (
	EOnFileCreate TFileEvent = iota + 200
	EOnFileWrite
	EOnFileRemove
	EOnFileRename
	EOnFileChmod

	FirstFileEvent TFileEvent = EOnFileCreate
	LastFileEvent  TFileEvent = EOnFileChmod
)

var EventNames = map[ITEvent]string{
	EOnStart:      "onStart",
	EOnExit:       "onExit",
	EOnExec:       "onExec",
	EOnFinish:     "onFinish",
	EOnError:      "onError",
	EOnFileCreate: "onFileCreate",
	EOnFileWrite:  "onFileWrite",
	EOnFileRemove: "onFileRemove",
	EOnFileRename: "onFileRename",
	EOnFileChmod:  "onFileChmod",
}

func ListFileEvents() []TFileEvent {
	res := []TFileEvent{}
	for i := FirstFileEvent; i <= LastFileEvent; i++ {
		res = append(res, i)
	}
	return res
}

type Pipe chan IEvent

type IEvent interface {
	Args() []interface{}
}

type EventBase struct {
	Type ITEvent
}

func OnStart() IEvent {
	return NewGenericEvent(EOnStart)
}

func OnFinish(key ptypes.Key) IEvent {
	return NewApiEvent(EOnFinish, key)
}

func OnError(err error) IEvent {
	return NewErrorEvent(EOnError, err)
}

func OnExec(key ptypes.Key) IEvent {
	return NewApiEvent(EOnExec, key)
}

func OnFileCreate(file string) IEvent {
	return NewFileEvent(EOnFileCreate, file)
}

func OnFileWrite(file string) IEvent {
	return NewFileEvent(EOnFileWrite, file)
}

func OnFileRemove(file string) IEvent {
	return NewFileEvent(EOnFileRemove, file)
}

func OnFileRename(file string) IEvent {
	return NewFileEvent(EOnFileRename, file)
}

func OnFileChmod(file string) IEvent {
	return NewFileEvent(EOnFileChmod, file)
}
