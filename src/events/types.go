package events

import (
	ptypes "example.com/gocr/src/process/types"
)

type EventType string
type EventKind string

type EventTK struct {
	Type EventType
	Kind EventKind
}

const (
	EKGeneric EventKind = "generic"
	EKApi     EventKind = "api"
	EKFile    EventKind = "file"

	ETOnStart EventType = "onStart"
	ETOnExit  EventType = "onExit"

	ETOnExec       EventType = "onExec"
	ETOnProcFinish EventType = "onFinish"
	ETOnProcError  EventType = "onError"

	ETOnFileCreate EventType = "onFileCreate"
	ETOnFileWrite  EventType = "onFileWrite"
	ETOnFileRemove EventType = "onFileRemove"
	ETOnFileRename EventType = "onFileRename"
	ETOnFileChmod  EventType = "onFileChmod"
)

func ListFileEvents() []EventType {
	return []EventType{
		ETOnFileCreate,
		ETOnFileWrite,
		ETOnFileRemove,
		ETOnFileRename,
		ETOnFileChmod,
	}
}

// type TEvent int
// type TGenericEvent TEvent
// type TApiEvent TEvent
// type TFileEvent TEvent

// const (
// 	EOnStart TGenericEvent = iota
// 	EOnExit

// 	FirstGenericEvent TGenericEvent = EOnStart
// 	LastGenericEvent  TGenericEvent = EOnExit
// )
// const (
// 	EOnExec TApiEvent = iota + 100
// 	EOnFinish
// 	EOnError

// 	FirstApiEvent TApiEvent = EOnExec
// 	LastApiEvent  TApiEvent = EOnError
// )
// const (
// 	EOnFileCreate TFileEvent = iota + 200
// 	EOnFileWrite
// 	EOnRemove
// 	EOnRename
// 	EOnChmod

// 	FirstFileEvent TFileEvent = EOnFileCreate
// 	LastFileEvent  TFileEvent = EOnChmod
// )

type Pipe chan IEvent

type IEvent interface {
	Describe() EventTK
	Args() []interface{}
}

type EventBase struct {
	Type EventType
}

func OnStart() IEvent {
	return NewGenericEvent(ETOnStart)
}

func OnFinish(key ptypes.Key) IEvent {
	return NewApiEvent(ETOnProcFinish, key)
}

func OnError(err error) IEvent {
	return NewErrorEvent(ETOnProcError, err)
}

func OnExec(key ptypes.Key) IEvent {
	return NewApiEvent(ETOnExec, key)
}

func OnFileCreate(file string) IEvent {
	return NewFileEvent(ETOnFileCreate, file)
}

func OnFileWrite(file string) IEvent {
	return NewFileEvent(ETOnFileWrite, file)
}

func OnFileRemove(file string) IEvent {
	return NewFileEvent(ETOnFileRemove, file)
}

func OnFileRename(file string) IEvent {
	return NewFileEvent(ETOnFileRename, file)
}

func OnFileChmod(file string) IEvent {
	return NewFileEvent(ETOnFileChmod, file)
}
