// Copyright (c) 2022 Cisco All Rights Reserved.
package api

import (
	"github.com/kube-logging/custom-runner/src/api/types"
	ptypes "github.com/kube-logging/custom-runner/src/process/types"
)

func (a *API) List() types.ApiResult {
	a.processes.Lock()
	defer a.processes.Unlock()

	procs := []ptypes.Process{}

	for _, v := range a.processes.Map() {
		procs = append(procs, v)

	}

	return types.ApiResult{Success: true, Response: procs}
}
