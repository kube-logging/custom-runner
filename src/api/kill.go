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

	"github.com/kube-logging/custom-runner/src/api/types"
	ptypes "github.com/kube-logging/custom-runner/src/process/types"
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
