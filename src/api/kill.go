// Copyright (c) 2022 Cisco All Rights Reserved.
package api

import (
	"fmt"

	"example.com/gocr/src/api/types"
	ptypes "example.com/gocr/src/process/types"
)

func (a *API) Kill(key ptypes.Key) types.ApiResult {
	a.processes.Lock()
	defer a.processes.Unlock()
	return a.kill(key)
}

func (a *API) kill(key ptypes.Key) types.ApiResult {
	r, ok := a.processes.Map()[key]
	if !ok {
		return types.ApiResult{Error: fmt.Errorf(types.ErrNoProcFound, key)}
	}
	if err := r.Cmd.Process.Kill(); err != nil {
		return types.ApiResult{Error: err}
	}
	return types.ApiResult{Success: true, Response: []ptypes.Process{r}}
}
