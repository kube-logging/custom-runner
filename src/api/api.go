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
	"github.com/kube-logging/custom-runner/src/config"
	ptypes "github.com/kube-logging/custom-runner/src/process/types"
)

type API struct {
	processes ptypes.IProcess
	commands  map[string]types.APICommandProto
}

func New(processes ptypes.IProcess) *API {
	api := &API{processes: processes}
	proto := map[string]types.APICommandProto{
		"exec": types.APICommandProto(func(key string, args []byte) types.ApiResult {
			return api.Exec(key, string(args))
		}),
		"kill": types.APICommandProto(func(key string, args []byte) types.ApiResult {
			return api.Kill(key)
		}),
		"restart": types.APICommandProto(func(key string, args []byte) types.ApiResult {
			return api.Restart(key)
		}),
		"get": types.APICommandProto(func(key string, args []byte) types.ApiResult {
			return api.Get(key)
		}),
		"list": types.APICommandProto(func(key string, args []byte) types.ApiResult {
			return api.List()
		}),
		"exit": types.APICommandProto(func(key string, args []byte) types.ApiResult {
			return api.Exit()
		}),
		"config": types.APICommandProto(func(key string, args []byte) types.ApiResult {
			return api.Config()
		}),
	}
	api.commands = proto
	return api
}

func (a *API) Command(command string) (types.APICommandProto, bool) {
	proto, ok := a.commands[command]
	return proto, ok
}

func (a *API) RunAction(action config.Action) types.ApiResult {
	for k, v := range action {
		apiCmd, ok := a.Command(k)
		if !ok {
			return types.ApiResult{
				Error: fmt.Errorf(types.ErrUNKCommand, k),
			}
		}
		return apiCmd(v.Key, []byte(v.Args))
	}
	return types.ApiResult{}
}

func (a *API) RunActions(actions []config.Action) []types.ApiResult {
	var results []types.ApiResult
	for _, action := range actions {
		res := a.RunAction(action)
		results = append(results, res)
	}
	return results
}
