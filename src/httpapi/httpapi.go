// Copyright (c) 2022 Cisco All Rights Reserved.
package httpapi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"

	"example.com/gocr/src/api"
	"example.com/gocr/src/api/types"
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
	w.Write([]byte(errStr))
}

func CommandHandler(api *api.API, apiRegx *regexp.Regexp) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		matches := apiRegx.FindStringSubmatch(r.URL.Path)

		apiCommandPos := apiRegx.SubexpIndex(APIRegexPatternKeyApi)
		apiCommand := matches[apiCommandPos]

		if apiCmd, ok := api.Command(apiCommand); ok {
			apiKeyPos := apiRegx.SubexpIndex(APIRegexPatternKeyKey)
			apiKey := matches[apiKeyPos]

			body, err := ioutil.ReadAll(r.Body)
			defer r.Body.Close()
			if err != nil {
				handleError(w, ErrBodyRead, http.StatusInternalServerError)
				return
			}

			res := apiCmd(apiKey, body)
			jsonData, err := json.Marshal(res)
			if err != nil {
				handleError(w, ErrApiRespToJson, http.StatusInternalServerError)
				return
			}

			w.Write(jsonData)

			return
		}
		handleError(w, fmt.Sprintf(types.ErrUNKCommand, apiCommand), http.StatusNotFound)
	}
}
