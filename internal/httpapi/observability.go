// Copyright © 2026 Kube logging authors
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
	"net/http"
	"slices"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Readiness reports whether the runner can actually do its job.
type Readiness struct {
	mu       sync.Mutex
	started  bool
	degraded []string
}

// SetStarted marks startup complete.
func (r *Readiness) SetStarted(started bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.started = started
}

// Degrade records a reason the runner cannot serve its purpose.
func (r *Readiness) Degrade(reason string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.degraded = append(r.degraded, reason)
}

func (r *Readiness) Status() (bool, []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.started {
		return false, []string{"starting up"}
	}
	if len(r.degraded) > 0 {
		return false, slices.Clone(r.degraded)
	}

	return true, nil
}

// ObservabilityMux serves metrics and health on a listener separate from the
// command API, so scraping does not require exposing the exec endpoints.
func ObservabilityMux(ready *Readiness) *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle("GET /metrics", promhttp.Handler())

	// Liveness is deliberately trivial: if this answers, the process is alive and
	// restarting it would not help.
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		if ok, reasons := ready.Status(); !ok {
			http.Error(w, strings.Join(reasons, "; "), http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ok"))
	})

	return mux
}
