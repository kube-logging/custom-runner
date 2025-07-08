// Copyright (c) 2022 Cisco All Rights Reserved.
package api

import (
	"fmt"

	"github.com/kube-logging/custom-runner/src/api/types"
	ptypes "github.com/kube-logging/custom-runner/src/process/types"
)

func (a *API) Restart(key ptypes.Key) types.ApiResult {
	if proc, ok := a.processes.Map()[key]; ok {
		if res := a.Kill(key); res.Error != nil {
			return res
		}
		<-proc.Done
		a.processes.Lock()
		defer a.processes.Unlock()
		return a.exec(key, proc.Cmd.Args[1:])
	}

	return types.ApiResult{Error: fmt.Errorf(types.ErrNoProcFound, key)}
}
