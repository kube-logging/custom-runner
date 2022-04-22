//usr/local/go/bin/go run $0 "$@"; exit $?;
package main

import (
	"fmt"
	"net/http"
	"regexp"

	"example.com/gocr/src/api"
	"example.com/gocr/src/config"
	"example.com/gocr/src/events"
	"example.com/gocr/src/filewatcher"
	"example.com/gocr/src/httpapi"
	"example.com/gocr/src/process"
)

func main() {
	conf := config.New()
	if err := conf.LoadFile("config.yaml"); err != nil {
		panic(err)
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
			// fmt.Println(event)
			actions, err := conf.ActionsForEvent(event.Args())
			if err != nil {
				if config.IsNotFound(err) {
					continue
				}
				fmt.Println("event error", err)
			}
			res := api.RunActions(actions)
			for _, r := range res {
				if r.Error != nil {
					fmt.Println("error:", r)
				}
			}
		}
	}()

	httpApi := httpapi.New()

	apiRegx := regexp.MustCompile(httpapi.APIRegxPattern)

	httpApi.HandleFunc(apiRegx, httpapi.CommandHandler(api, apiRegx))

	events.Add(events.OnStart())
	http.ListenAndServe(":7357", httpApi)
}
