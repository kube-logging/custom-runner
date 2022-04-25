// Copyright (c) 2022 Cisco All Rights Reserved.
package api

import (
	"example.com/gocr/src/api/types"
	ptypes "example.com/gocr/src/process/types"
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
