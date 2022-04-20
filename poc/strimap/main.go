package main

import (
	"fmt"

	"example.com/gocr/src/config"
	"github.com/MakeNowJust/heredoc"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v2"
)

type ActionInner struct {
	Key  string `json:"key,omitempty"`
	Args string `json:"command,omitempty"`
}

type Action map[string]ActionInner

type RawMessage struct {
}

func main() {
	co := []byte(heredoc.Doc(`
events:
  onStart:
    - exec:
        command: echo "foobar"
    - exec:
        key: shifter
        command: while true; do date; sleep 1; done
`))

	conf := config.Imap{}
	// conf := make(map[string]json.RawMessage)
	if err := yaml.Unmarshal(co, &conf); err != nil {
		panic(err.Error())
	}

	fmt.Printf("%+v\n", conf)

	foo := conf.GetIn("events", "onStart")
	// // foo := conf.GetIn("events", interface{}("onStart"))
	fmt.Printf("%#v\n", foo)

	var actions []Action

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{TagName: "json", Result: &actions})
	if err != nil {
		panic(err)
	}

	if err := decoder.Decode(foo); err != nil {
		panic(err)
	}

	fmt.Printf("%#v\n", actions)

}
