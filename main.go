//usr/local/go/bin/go run $0 "$@"; exit $?;
package main

import (
	"flag"
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

type configExt []string

func (c *configExt) String() string {
	return fmt.Sprint(*c)
}

func (c *configExt) Set(value string) error {
	*c = append(*c, value)
	return nil
}

var configArgs configExt
var cfg = flag.String("cfgfile", "", "config file")
var port = flag.Int("port", 7357, "listening port")

func main() {

	flag.Var(&configArgs, "config", "config override")

	flag.Parse()

	conf := config.New()
	if *cfg != "" {
		if err := conf.LoadFile(*cfg); err != nil {
			fmt.Printf("no config file found:%v\n", *cfg)
			return
		}
	}

	conf = conf.Override(configArgs)

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
	fmt.Printf("listening on port:%v\n", *port)
	fmt.Println(http.ListenAndServe(fmt.Sprintf(":%v", *port), httpApi))
}
