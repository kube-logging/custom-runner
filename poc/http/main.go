package main

import (
	"fmt"
	"net/http"
	"regexp"
)

const (
	APIRegexPatternKeyApi = "api"
	APIRegexPatternKeyKey = "key"
	APIRegxPattern        = `^/(?P<` + APIRegexPatternKeyApi + `>[^/]+)(/(?P<` + APIRegexPatternKeyKey + `>[^/]+))?$`
)

var (
	APICommands = map[string]bool{"exec": true, "kill": true, "exit": true}
)

type route struct {
	pattern *regexp.Regexp
	handler http.Handler
}

type RegexpHandler struct {
	routes []*route
}

func (h *RegexpHandler) Handler(pattern *regexp.Regexp, handler http.Handler) {
	h.routes = append(h.routes, &route{pattern, handler})
}

func (h *RegexpHandler) HandleFunc(pattern *regexp.Regexp, handler func(http.ResponseWriter, *http.Request)) {
	h.routes = append(h.routes, &route{pattern, http.HandlerFunc(handler)})
}

func (h *RegexpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, route := range h.routes {
		if route.pattern.MatchString(r.URL.Path) {
			route.handler.ServeHTTP(w, r)
			return
		}
	}
	// no pattern matched; send 404 response
	http.NotFound(w, r)
}

func CommandHandler(apiRegx *regexp.Regexp) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		matches := apiRegx.FindStringSubmatch(r.URL.Path)

		apiCommandPos := apiRegx.SubexpIndex(APIRegexPatternKeyApi)
		apiCommand := matches[apiCommandPos]

		if _, ok := APICommands[apiCommand]; ok {
			apiKeyPos := apiRegx.SubexpIndex(APIRegexPatternKeyKey)
			apiKey := matches[apiKeyPos]
			w.Write(append([]byte(apiCommand), []byte(apiKey)...))
		}
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf("unknown API command:%s\n", apiCommand)))
	}
}

func main() {
	handler := &RegexpHandler{}

	apiRegx := regexp.MustCompile(APIRegxPattern)

	handler.HandleFunc(apiRegx, CommandHandler(apiRegx))

	http.ListenAndServe(":7357", handler)
}
