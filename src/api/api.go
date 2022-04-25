// Copyright (c) 2022 Cisco All Rights Reserved.
package api

import (
	"fmt"

	"example.com/gocr/src/api/types"
	"example.com/gocr/src/config"
	ptypes "example.com/gocr/src/process/types"
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
