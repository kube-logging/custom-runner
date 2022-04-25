package api

import (
	"fmt"
	"os"
	"os/exec"

	"example.com/gocr/src/api/types"
	"example.com/gocr/src/events"
	ptypes "example.com/gocr/src/process/types"
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
		// a.processes.Update(func(in ptypes.ProcessMap) ptypes.ProcessMap {
		// 	delete(in, key)
		// 	return in
		// })
		events.Add(events.OnFinish(key))
		close(done)
	}()

	<-wait

	proc := ptypes.Process{Key: key, Cmd: cmd, Done: done}
	if err == nil {
		a.processes.Map()[key] = proc
		// a.processes.Update(func(in ptypes.ProcessMap) ptypes.ProcessMap {
		// 	in[key] = proc
		// 	return in
		// })

		events.Add(events.OnExec(key))
	} else {
		events.Add(events.OnError(err))
	}

	return types.ApiResult{
		Error:    err,
		Success:  err == nil,
		Response: proc, //[]ptypes.Process{proc},
	}
}
