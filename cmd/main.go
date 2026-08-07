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
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/kube-logging/custom-runner/internal/runner"
)

// execFlag collects repeated `-exec "key -> command"` pairs.
type execFlag struct {
	cmds map[string]string
}

func (e *execFlag) Set(value string) error {
	key, command, found := strings.Cut(value, "->")
	if !found {
		return fmt.Errorf("expected \"key -> command\", got %q", value)
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("empty key in %q", value)
	}
	e.cmds[key] = strings.TrimSpace(command)

	return nil
}

func (e *execFlag) String() string {
	return fmt.Sprintf("%v", e.cmds)
}

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		configFile  = flag.String("cfgfile", "", "config file")
		configJSON  = flag.String("cfgjson", "", "config from json arg")
		startup     = flag.String("startup", "", "execute command at startup")
		address     = flag.String("address", "127.0.0.1", "command api bind address; it executes arbitrary shell, keep it off 0.0.0.0")
		port        = flag.Int("port", 7357, "command api port, 0 to disable")
		metricsAddr = flag.String("metrics-address", "0.0.0.0", "metrics and health bind address")
		metricsPort = flag.Int("metrics-port", 9533, "metrics and health port, 0 to disable")
		debug       = flag.Bool("debug", false, "debug logs")
		logFormat   = flag.String("log-format", "json", "log output format (json or text)")
	)

	execs := execFlag{cmds: make(map[string]string)}
	flag.Var(&execs, "exec", "exec command, as \"key -> command\"")

	flag.Parse()

	setupLogging(*logFormat, *debug)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return runner.Run(ctx, runner.Options{
		ConfigFile:  *configFile,
		ConfigJSON:  *configJSON,
		Startup:     *startup,
		Execs:       execs.cmds,
		Address:     *address,
		Port:        *port,
		MetricsAddr: *metricsAddr,
		MetricsPort: *metricsPort,
	})
}

func setupLogging(format string, debug bool) {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{Level: level, AddSource: debug}

	var handler slog.Handler = slog.NewJSONHandler(os.Stderr, opts)
	if strings.EqualFold(format, "text") {
		handler = slog.NewTextHandler(os.Stderr, opts)
	}

	slog.SetDefault(slog.New(handler))
}
