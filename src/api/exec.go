// Copyright © 2022 Cisco Systems, Inc. and/or its affiliates
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
			events.Add(events.OnErrorWithKey(key, err))
			return
		}
		if err = cmd.Wait(); err != nil {
			events.Add(events.OnErrorWithKey(key, err))
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
