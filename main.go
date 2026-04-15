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

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kube-logging/custom-runner/src/api"
	"github.com/kube-logging/custom-runner/src/config"
	"github.com/kube-logging/custom-runner/src/events"
	"github.com/kube-logging/custom-runner/src/filewatcher"
	"github.com/kube-logging/custom-runner/src/httpapi"
	"github.com/kube-logging/custom-runner/src/metrics"
	"github.com/kube-logging/custom-runner/src/process"
)

type ExecArgs struct {
	cmds map[string]string
}

func (e *ExecArgs) Set(value string) error {
	parts := strings.Split(value, "->")
	key, val := strings.TrimSpace(parts[0]), strings.TrimSpace(strings.Join(parts[1:], "->"))
	e.cmds[key] = val
	return nil
}

func (e *ExecArgs) String() string {
	return fmt.Sprintf("%#v", e)
}

var cfg = flag.String("cfgfile", "", "config file")
var port = flag.Int("port", 7357, "listening port")
var configJson = flag.String("cfgjson", "", "config from json arg")
var startup = flag.String("startup", "", "execute command at startup")
var debug = flag.Bool("debug", false, "debug logs")
var logFormat = flag.String("log-format", "json", "log output format (json or text)")

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	execes := ExecArgs{cmds: make(map[string]string)}
	flag.Var(&execes, "exec", "exec command")

	flag.Parse()

	logLevel := slog.LevelInfo
	if *debug {
		logLevel = slog.LevelDebug
	}
	setupSlog(*logFormat, logLevel)

	conf := config.DefaultConfig
	if *cfg != "" {
		if err := conf.LoadFile(*cfg); err != nil {
			return fmt.Errorf("failed to load config file %q: %w", *cfg, err)
		}
	}

	if *configJson != "" {
		if err := json.Unmarshal([]byte(*configJson), &conf.Strimap); err != nil {
			return fmt.Errorf("failed to parse config json: %w", err)
		}
	}

	slog.Debug("config loaded", "config", fmt.Sprintf("%#v", conf))

	filesToWatch := conf.CollectFileEvents()
	if err := filewatcher.Start(); err != nil {
		return fmt.Errorf("failed to start file watcher: %w", err)
	}
	defer func() {
		if err := filewatcher.Stop(); err != nil {
			slog.Error("failed to stop file watcher", "error", err)
		}
	}()
	for _, f := range filesToWatch {
		if err := filewatcher.Add(f); err != nil {
			slog.Error("failed to watch file", "file", f, "error", err)
		}
	}

	runnerAPI := api.New(process.New())

	go func() {
		for {
			event := <-events.DefaultPipe
			slog.Debug("event received", "event", event)
			trackMetrics(event)
			actions, err := conf.ActionsForEvent(event.Args())
			if err != nil {
				if config.IsNotFound(err) {
					continue
				}
				slog.Error("event error", "error", err)
			}
			res := runnerAPI.RunActions(actions)
			for _, r := range res {
				if r.Error != nil {
					slog.Error("action error", "result", r)
				}
			}
		}
	}()

	httpApi := httpapi.New()
	httpApi.Handler(regexp.MustCompile(`^/metrics$`), promhttp.Handler())

	apiRegexp := regexp.MustCompile(httpapi.APIRegxPattern)
	httpApi.HandleFunc(apiRegexp, httpapi.CommandHandler(runnerAPI, apiRegexp))

	if *startup != "" {
		runnerAPI.Exec("startup", *startup)
	}

	for k, c := range execes.cmds {
		runnerAPI.Exec(k, c)
	}

	events.Add(events.OnStart())
	if *port != 0 {
		slog.Info("listening", "port", *port)
		if err := http.ListenAndServe(fmt.Sprintf(":%v", *port), httpApi); err != nil {
			return fmt.Errorf("http server: %w", err)
		}
	} else {
		slog.Info("listening port disabled")
	}

	return nil
}

func setupSlog(format string, level slog.Level) {
	logLevel := new(slog.LevelVar)
	logLevel.Set(level)
	opts := &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: level <= slog.LevelDebug,
	}

	switch strings.ToLower(format) {
	case "text":
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, opts)))
	default:
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, opts)))
	}
}

func trackMetrics(event events.IEvent) {
	switch e := event.(type) {
	case events.ApiEvent:
		switch e.Type {
		case events.EOnExec:
			metrics.RecordExecStart(e.Key)
		case events.EOnFinish:
			metrics.RecordExecSuccess(e.Key)
		}
	case events.ErrorEvent:
		if e.Key != "" {
			metrics.RecordExecError(e.Key)
		} else {
			metrics.RecordWatcherError()
		}
	}
}
