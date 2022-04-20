package types

import (
	"os/exec"
	"sync"
)

type Key = string
type Waiter chan struct{}

type ProcessMap = map[Key]Process

type ProcessUpdater func(reg ProcessMap) ProcessMap

type IProcess interface {
	Map() map[Key]Process
	sync.Locker
	Update(ProcessUpdater)
}

type Process struct {
	Key  Key       `json:"key" yaml:"key"`
	Cmd  *exec.Cmd `json:"cmd" yaml:"cmd"`
	Done Waiter    `json:"-" yaml:"-"`
}
