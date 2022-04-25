//usr/local/go/bin/go run $0 "$@"; exit $?;
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"regexp"

	"example.com/gocr/src/api"
	"example.com/gocr/src/config"
	"example.com/gocr/src/events"
	"example.com/gocr/src/filewatcher"
	"example.com/gocr/src/httpapi"
	"example.com/gocr/src/info"
	"example.com/gocr/src/process"
)

var cfg = flag.String("cfgfile", "", "config file")
var port = flag.Int("port", 7357, "listening port")
var configJson = flag.String("cfgjson", "", "config from json arg")
var startup = flag.String("startup", "", "execute command at startup")

func main() {

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

	// info.Printf("%#v\n", conf)

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
			// info.Println(event)
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

	events.Add(events.OnStart())
	info.Printf("listening on port:%v\n", *port)
	info.Println(http.ListenAndServe(fmt.Sprintf(":%v", *port), httpApi))
}
