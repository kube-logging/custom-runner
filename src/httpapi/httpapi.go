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

package httpapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"

	"github.com/kube-logging/custom-runner/src/api"
	"github.com/kube-logging/custom-runner/src/api/types"
)

const (
	APIRegexPatternKeyApi = "api"
	APIRegexPatternKeyKey = "key"
	APIRegxPattern        = `^/(?P<` + APIRegexPatternKeyApi + `>[^/]+)(/(?P<` + APIRegexPatternKeyKey + `>[^/]+))?$`

	ErrBodyRead      = "error reading request body\n"
	ErrApiRespToJson = "unable to json marshal api response"
)

type route struct {
	pattern *regexp.Regexp
	handler http.Handler
}

type HTTPAPI struct {
	routes []*route
}

func New() *HTTPAPI {
	return &HTTPAPI{}
}

func (h *HTTPAPI) Handler(pattern *regexp.Regexp, handler http.Handler) {
	h.routes = append(h.routes, &route{pattern, handler})
}

func (h *HTTPAPI) HandleFunc(pattern *regexp.Regexp, handleFunc func(http.ResponseWriter, *http.Request)) {
	h.routes = append(h.routes, &route{pattern, http.HandlerFunc(handleFunc)})
}

func (h *HTTPAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, route := range h.routes {
		if route.pattern.MatchString(r.URL.Path) {
			route.handler.ServeHTTP(w, r)
			return
		}
	}
	// no pattern matched; send 404 response
	http.NotFound(w, r)
}

func handleError(w http.ResponseWriter, errStr string, status int) {
	w.WriteHeader(status)
	w.Write([]byte(errStr)) //nolint:errcheck
}

func CommandHandler(api *api.API, apiRegx *regexp.Regexp) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		matches := apiRegx.FindStringSubmatch(r.URL.Path)

		apiCommandPos := apiRegx.SubexpIndex(APIRegexPatternKeyApi)
		apiCommand := matches[apiCommandPos]

		if apiCmd, ok := api.Command(apiCommand); ok {
			apiKeyPos := apiRegx.SubexpIndex(APIRegexPatternKeyKey)
			apiKey := matches[apiKeyPos]

			body, err := io.ReadAll(r.Body)
			if err != nil {
				handleError(w, ErrBodyRead, http.StatusInternalServerError)
				return
			}
			defer r.Body.Close() //nolint:errcheck

			res := apiCmd(apiKey, body)
			jsonData, err := json.Marshal(res)
			if err != nil {
				handleError(w, ErrApiRespToJson, http.StatusInternalServerError)
				return
			}

			w.Write(jsonData) //nolint:errcheck

			return
		}
		handleError(w, fmt.Sprintf(types.ErrUNKCommand, apiCommand), http.StatusNotFound)
	}
}
