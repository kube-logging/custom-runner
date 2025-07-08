// Copyright (c) 2022 Cisco All Rights Reserved.
package api

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/kube-logging/custom-runner/src/api/types"
	"github.com/kube-logging/custom-runner/src/events"
	ptypes "github.com/kube-logging/custom-runner/src/process/types"
)

func (a *API) Exec(key ptypes.Key, command string) types.ApiResult {
	a.processes.Lock()
	defer a.processes.Unlock()

	args := append([]string{"-c"}, command)

	return a.exec(key, args)
}

func (a *API) exec(key ptypes.Key, args []string) types.ApiResult {
	if _, ok := a.processes.Map()[key]; ok {
		return types.ApiResult{Error: fmt.Errorf(types.ErrAlreadyRunning, key)}
	}

	cmd := exec.Command("sh", args...)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	wait := make(ptypes.Waiter)
	done := make(ptypes.Waiter)
	var err error
	go func() {
		err = cmd.Start()
		close(wait)
		if err != nil {
			events.Add(events.OnError(err))
			return
		}
		if err = cmd.Wait(); err != nil {
			events.Add(events.OnError(err))
		}
		a.processes.Lock()
		defer a.processes.Unlock()
		delete(a.processes.Map(), key)
		events.Add(events.OnFinish(key))
		close(done)
	}()

	<-wait

	proc := ptypes.Process{Key: key, Cmd: cmd, Done: done}
	if err == nil {
		a.processes.Map()[key] = proc

		events.Add(events.OnExec(key))
	} else {
		events.Add(events.OnError(err))
	}

	return types.ApiResult{
		Error:    err,
		Success:  err == nil,
		Response: proc,
	}
}
