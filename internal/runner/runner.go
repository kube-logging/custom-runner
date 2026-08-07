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

package runner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/kube-logging/custom-runner/internal/api"
	"github.com/kube-logging/custom-runner/internal/config"
	"github.com/kube-logging/custom-runner/internal/events"
	"github.com/kube-logging/custom-runner/internal/filewatcher"
	"github.com/kube-logging/custom-runner/internal/httpapi"
	"github.com/kube-logging/custom-runner/internal/process"
)

const (
	shutdownTimeout = 10 * time.Second
	// exitGrace bounds how long onExit actions may run before children are killed.
	exitGrace = 5 * time.Second
	// reapTimeout bounds waiting for killed children to be collected.
	reapTimeout = 5 * time.Second
)

// Options is the runner's configuration, already parsed from the command line.
type Options struct {
	ConfigFile string
	ConfigJSON string
	Startup    string
	Execs      map[string]string

	// Address is loopback by default: the command API executes arbitrary shell.
	Address     string
	Port        int
	MetricsAddr string
	MetricsPort int
}

// Run wires the runner up and blocks until ctx is canceled or a server fails.
func Run(ctx context.Context, opts Options) error {
	cfg, err := loadConfig(opts)
	if err != nil {
		return err
	}

	// Children must outlive the request that made them and survive onExit actions.
	procCtx, killChildren := context.WithCancel(context.Background())
	defer killChildren()

	ctx, shutdown := context.WithCancel(ctx)
	defer shutdown()

	bus := events.NewBus()
	table := process.NewTable()
	runnerAPI := api.New(procCtx, shutdown, table, bus, cfg)

	watcher, err := filewatcher.New(bus)
	if err != nil {
		return err
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			slog.Error("failed to close file watcher", "error", err)
		}
	}()

	ready := &httpapi.Readiness{}

	// An unregistered watch means silently never reloading, so it fails readiness.
	for _, path := range cfg.WatchPaths() {
		if err := watcher.Add(path); err != nil {
			slog.Error("failed to watch path", "path", path, "error", err)
			ready.Degrade(fmt.Sprintf("not watching %s: %v", path, err))
		}
	}
	go watcher.Run(ctx)

	consumed := make(chan struct{})
	go func() {
		defer close(consumed)
		consume(bus, runnerAPI, cfg)
	}()

	serveErr := make(chan error, 2)
	servers := startServers(opts, runnerAPI, ready, serveErr)

	bus.Publish(events.Event{Kind: events.OnStart})
	startCommands(runnerAPI, opts)
	ready.SetStarted(true)

	select {
	case <-ctx.Done():
	case err = <-serveErr:
	}

	ready.SetStarted(false)
	stopServers(servers)
	runExitActions(runnerAPI, cfg)
	killRemaining(table)
	killChildren()
	awaitReaped(table)
	bus.Close()
	<-consumed

	return err
}

// killRemaining terminates children through the table so the reaper knows the
// exits were deliberate. Canceling procCtx alone signals them behind Process's
// back, booking every shutdown as a crash and firing onError mid-teardown.
func killRemaining(table *process.Table) {
	for _, proc := range table.List() {
		if err := proc.Kill(); err != nil && !errors.Is(err, process.ErrNotStarted) {
			slog.Error("failed to kill process", "key", proc.Key, "error", err)
		}
	}
}

// awaitReaped waits for the killed children to be collected. Exiting first leaves
// zombies for whatever the container's pid 1 is, which need not reap them.
func awaitReaped(table *process.Table) {
	deadline := time.After(reapTimeout)

	for len(table.List()) > 0 {
		select {
		case <-deadline:
			slog.Warn("children still unreaped at timeout", "count", len(table.List()))
			return
		case <-time.After(20 * time.Millisecond):
		}
	}
}

func loadConfig(opts Options) (*config.Config, error) {
	cfg := config.New()

	if opts.ConfigFile != "" {
		if err := cfg.LoadFile(opts.ConfigFile); err != nil {
			return nil, err
		}
	}

	if opts.ConfigJSON != "" {
		if err := cfg.LoadJSON([]byte(opts.ConfigJSON)); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

func startServers(opts Options, a *api.API, ready *httpapi.Readiness, serveErr chan<- error) []*http.Server {
	var servers []*http.Server

	if opts.Port != 0 {
		srv := newServer(opts.Address, opts.Port, httpapi.CommandMux(a))
		servers = append(servers, srv)
		slog.Info("command api listening", "address", srv.Addr)
		go listen(srv, "command api", serveErr)
	} else {
		slog.Info("command api disabled")
	}

	if opts.MetricsPort != 0 {
		srv := newServer(opts.MetricsAddr, opts.MetricsPort, httpapi.ObservabilityMux(ready))
		servers = append(servers, srv)
		slog.Info("metrics listening", "address", srv.Addr)
		go listen(srv, "metrics", serveErr)
	} else {
		slog.Info("metrics disabled")
	}

	return servers
}

func newServer(address string, port int, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              net.JoinHostPort(address, strconv.Itoa(port)),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func listen(srv *http.Server, name string, serveErr chan<- error) {
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		serveErr <- fmt.Errorf("%s server: %w", name, err)
	}
}

func stopServers(servers []*http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	for _, srv := range servers {
		if err := srv.Shutdown(ctx); err != nil {
			slog.Error("failed to shut down server", "addr", srv.Addr, "error", err)
		}
	}
}

func startCommands(a *api.API, opts Options) {
	if opts.Startup != "" {
		if res := a.Exec("startup", opts.Startup); res.Error != "" {
			slog.Error("startup command failed", "error", res.Error)
		}
	}

	for key, command := range opts.Execs {
		if res := a.Exec(key, command); res.Error != "" {
			slog.Error("exec command failed", "key", key, "error", res.Error)
		}
	}
}

func consume(bus *events.Bus, a *api.API, cfg *config.Config) {
	for {
		event, open := bus.Next()
		if !open {
			return
		}

		if event.Err != nil {
			slog.Error("event reported an error", "kind", event.Kind, "subject", event.Subject, "error", event.Err)
		}
		slog.Debug("event received", "kind", event.Kind, "subject", event.Subject)

		for _, res := range a.RunActions(cfg.ActionsFor(event.Kind, event.Subject)) {
			if res.Error != "" {
				slog.Error("action failed", "kind", event.Kind, "subject", event.Subject, "error", res.Error)
			}
		}
	}
}

// runExitActions dispatches onExit inline and waits only on what it spawned;
// the supervised daemon does not exit until killChildren runs after us.
func runExitActions(a *api.API, cfg *config.Config) {
	actions := cfg.ActionsFor(events.OnExit, "")
	if len(actions) == 0 {
		return
	}

	var spawned []*process.Process
	for i, res := range a.RunActions(actions) {
		if res.Error != "" {
			slog.Error("onExit action failed", "error", res.Error)
			continue
		}
		// get also answers with a process; waiting on it would stall shutdown on
		// the very daemon killChildren has yet to stop.
		if !actions[i].StartsProcess() {
			continue
		}
		if proc, ok := res.Response.(*process.Process); ok {
			spawned = append(spawned, proc)
		}
	}

	deadline := time.After(exitGrace)
	for _, proc := range spawned {
		select {
		case <-proc.Done:
		case <-deadline:
			slog.Warn("onExit actions still running at grace expiry", "key", proc.Key)
			return
		}
	}
}
