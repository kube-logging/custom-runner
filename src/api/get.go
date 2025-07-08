// Copyright (c) 2022 Cisco All Rights Reserved.
package api

import (
	"fmt"

	"github.com/kube-logging/custom-runner/src/api/types"
	ptypes "github.com/kube-logging/custom-runner/src/process/types"
)

func (a *API) Get(key ptypes.Key) types.ApiResult {
	a.processes.Lock()
	defer a.processes.Unlock()

	r, ok := a.processes.Map()[key]
	if !ok {
		return types.ApiResult{Error: fmt.Errorf(types.ErrNoProcFound, key)}
	}
	return types.ApiResult{Success: true, Response: []ptypes.Process{r}}
}
