package api

import (
	"fmt"

	"example.com/gocr/src/api/types"
	ptypes "example.com/gocr/src/process/types"
)

func (a *API) Get(key ptypes.Key) types.ApiResult {
	a.processes.Lock()
	defer a.processes.Unlock()

	r, ok := a.processes.Map()[key]
	if !ok {
		return types.ApiResult{Error: fmt.Errorf(types.ErrNoProcFound, key)}
	}
	return types.ApiResult{Success: true, Processes: []ptypes.Process{r}}
}
