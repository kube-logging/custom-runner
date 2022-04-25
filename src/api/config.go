package api

import (
	"example.com/gocr/src/api/types"
	"example.com/gocr/src/config"
)

func (a *API) Config() types.ApiResult {
	return types.ApiResult{Success: true, Response: config.DefaultConfig}
}
