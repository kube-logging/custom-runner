// Copyright (c) 2022 Cisco All Rights Reserved.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/kube-logging/custom-runner/src/api"
	"github.com/kube-logging/custom-runner/src/config"
	"github.com/kube-logging/custom-runner/src/events"
	"github.com/kube-logging/custom-runner/src/filewatcher"
	"github.com/kube-logging/custom-runner/src/httpapi"
	"github.com/kube-logging/custom-runner/src/info"
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

func main() {
	execes := ExecArgs{cmds: make(map[string]string)}
	flag.Var(&execes, "exec", "exec command")

	flag.Parse()

	conf := config.DefaultConfig
	if *cfg != "" {
		if err := conf.LoadFile(*cfg); err != nil {
			info.Printf("no config file found:%v\n", *cfg)
			return
		}
	}

	if *configJson != "" {
		if err := json.Unmarshal([]byte(*configJson), &conf.Strimap); err != nil {
			info.Printf("unable parse config json:%v", err)
			return
		}
	}

	if *debug {
		info.Printf("%#v\n", conf)
	}

	filesToWatch := conf.CollectFileEvents()
	filewatcher.Start()
	defer filewatcher.Stop()
	for _, f := range filesToWatch {
		filewatcher.Add(f)
	}

	api := api.New(process.New())

	go func() {
		for {
			event := <-events.DefaultPipe
			if *debug {
				info.Println(event)
			}
			actions, err := conf.ActionsForEvent(event.Args())
			if err != nil {
				if config.IsNotFound(err) {
					continue
				}
				info.Println("event error", err)
			}
			res := api.RunActions(actions)
			for _, r := range res {
				if r.Error != nil {
					info.Println("error:", r)
				}
			}
		}
	}()

	httpApi := httpapi.New()

	apiRegx := regexp.MustCompile(httpapi.APIRegxPattern)

	httpApi.HandleFunc(apiRegx, httpapi.CommandHandler(api, apiRegx))

	if *startup != "" {
		api.Exec("startup", *startup)
	}

	for k, c := range execes.cmds {
		api.Exec(k, c)
	}

	events.Add(events.OnStart())
	if *port != 0 {
		info.Printf("listening on port:%v\n", *port)
		info.Println(http.ListenAndServe(fmt.Sprintf(":%v", *port), httpApi))
	} else {
		info.Printf("listening port disabled\n")
	}
}
