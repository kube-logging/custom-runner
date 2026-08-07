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
	"io"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/kube-logging/custom-runner/internal/api"
)

// maxBodyBytes bounds the exec payload; the body is a shell command, not a stream.
const maxBodyBytes = 1 << 20

// Process keys become Prometheus label values, so an unvalidated key lets a caller
// mint unbounded series on the metrics listener.
var validKey = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

// CommandMux routes the runner verbs. Mutating verbs are POST, reads are GET, so
// a stray GET cannot start or stop a process.
func CommandMux(a *api.API) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /exec/{key}", handle(a, "exec"))
	mux.HandleFunc("POST /kill/{key}", handle(a, "kill"))
	mux.HandleFunc("POST /restart/{key}", handle(a, "restart"))
	mux.HandleFunc("POST /exit", handle(a, "exit"))

	mux.HandleFunc("GET /get/{key}", handle(a, "get"))
	mux.HandleFunc("GET /list", handle(a, "list"))
	mux.HandleFunc("GET /config", handle(a, "config"))

	// No "/" catch-all: registering one would match every path and suppress the
	// mux's own 405 for a method mismatch. Unmatched paths already 404.
	return mux
}

// handle resolves the verb once at wiring time; every name below is a literal.
func handle(a *api.API, name string) http.HandlerFunc {
	cmd, found := a.Command(name)
	if !found {
		panic("httpapi: unregistered command " + name)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")
		if key != "" && !validKey.MatchString(key) {
			http.Error(w, "key must match "+validKey.String(), http.StatusBadRequest)
			return
		}

		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
		if err != nil {
			http.Error(w, "error reading request body", http.StatusRequestEntityTooLarge)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(cmd(key, body)); err != nil {
			slog.Error("failed to encode api response", "error", err)
		}
	}
}
