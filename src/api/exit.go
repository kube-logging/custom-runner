// Copyright (c) 2022 Cisco All Rights Reserved.
package api

import (
	"os"
	"time"

	"example.com/gocr/src/api/types"
)

func (a *API) Exit() types.ApiResult {
	a.processes.Lock()
	defer a.processes.Unlock()

	quit := make(chan struct{})
	go func() {
		<-quit
		<-time.After(time.Second)
		os.Exit(0)
	}()

	for id := range a.processes.Map() {
		if r, ok := a.processes.Map()[id]; ok {
			r.Cmd.Process.Kill()
			delete(a.processes.Map(), id)
		}
	}

	defer close(quit)
	return types.ApiResult{Success: true}
}
