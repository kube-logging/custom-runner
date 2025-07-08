// Copyright (c) 2022 Cisco All Rights Reserved.
package api

import (
	"github.com/kube-logging/custom-runner/src/api/types"
	"github.com/kube-logging/custom-runner/src/config"
)

func (a *API) Config() types.ApiResult {
	return types.ApiResult{Success: true, Response: config.DefaultConfig}
}
