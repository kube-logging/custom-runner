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

package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const namespace = "sidecar_reloader"

var (
	// LastExecError tracks whether the last execution of a process resulted in an error.
	// 1 = error, 0 = success.
	LastExecError = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "config_reloader_last_reload_error",
		Help:      "Whether the last reload resulted in an error (1 for error, 0 for success)",
	}, []string{"process"})

	// ExecDuration tracks the duration of the last execution for a process.
	ExecDuration = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "config_reloader_last_request_duration_seconds",
		Help:      "Duration of last reload execution in seconds",
	}, []string{"process"})

	// SuccessReloads counts the total number of successful executions per process.
	SuccessReloads = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "config_reloader_success_reloads_total",
		Help:      "Total number of successful reload executions",
	}, []string{"process"})

	// ExecErrors counts the total number of failed executions per process.
	ExecErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "config_reloader_request_errors_total",
		Help:      "Total number of reload execution errors",
	}, []string{"process"})

	// WatcherErrors counts the total number of file-watcher errors.
	WatcherErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "config_reloader_watcher_errors_total",
		Help:      "Total number of filesystem watcher errors",
	})

	// TotalExecs counts the total number of execution attempts per process.
	TotalExecs = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "config_reloader_requests_total",
		Help:      "Total number of reload execution attempts",
	}, []string{"process"})
)

func init() {
	prometheus.MustRegister(LastExecError)
	prometheus.MustRegister(ExecDuration)
	prometheus.MustRegister(SuccessReloads)
	prometheus.MustRegister(ExecErrors)
	prometheus.MustRegister(WatcherErrors)
	prometheus.MustRegister(TotalExecs)
}

// state tracks per-process start times to compute execution duration.
var (
	mu         sync.Mutex
	startTimes = make(map[string]time.Time)
)

// RecordExecStart records the start of an execution for the given process key.
func RecordExecStart(key string) {
	mu.Lock()
	defer mu.Unlock()
	startTimes[key] = time.Now()
	TotalExecs.WithLabelValues(key).Inc()
}

// RecordExecSuccess records a successful completion for the given process key.
func RecordExecSuccess(key string) {
	mu.Lock()
	start, ok := startTimes[key]
	delete(startTimes, key)
	mu.Unlock()

	if ok {
		ExecDuration.WithLabelValues(key).Set(time.Since(start).Seconds())
	}
	SuccessReloads.WithLabelValues(key).Inc()
	LastExecError.WithLabelValues(key).Set(0.0)
}

// RecordExecError records a failed execution for the given process key.
func RecordExecError(key string) {
	mu.Lock()
	delete(startTimes, key)
	mu.Unlock()

	ExecErrors.WithLabelValues(key).Inc()
	LastExecError.WithLabelValues(key).Set(1.0)
}

// RecordWatcherError records a file-watcher error.
func RecordWatcherError() {
	WatcherErrors.Inc()
}
