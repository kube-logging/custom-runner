package api

import (
	"fmt"

	"example.com/gocr/src/api/types"
	ptypes "example.com/gocr/src/process/types"
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
