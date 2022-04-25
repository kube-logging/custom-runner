// Copyright (c) 2022 Cisco All Rights Reserved.
package process

import (
	"sync"

	"example.com/gocr/src/process/types"
)

type Process struct {
	Reg   types.ProcessMap
	mutex sync.Mutex
}

func New() *Process {
	return &Process{Reg: make(types.ProcessMap)}
}

func (p *Process) Update(fn types.ProcessUpdater) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.Reg = fn(p.Reg)
}

func (p *Process) Lock() {
	p.mutex.Lock()
}

func (p *Process) Unlock() {
	p.mutex.Unlock()
}

func (p *Process) Map() types.ProcessMap {
	return p.Reg
}
