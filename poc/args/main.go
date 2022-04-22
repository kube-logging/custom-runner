//usr/local/go/bin/go run $0 "$@"; exit $?;
package main

import (
	"flag"
	"fmt"
	"strings"

	"example.com/gocr/src/config"
)

type foo []string

func (f *foo) String() string {
	return fmt.Sprint(*f)
}

func (f *foo) Set(value string) error {
	*f = append(*f, value)
	return nil
}

// ./poc/args/main.go -config "events.onStart.[].exec.command=./bin/node_exporter --web.listen-address=:9200 --collector.disable-defaults --collector.filesystem"

var configExt foo

const (
	ImapArrayId = "[]"
)

func main() {

	cfg := config.Imap{}

	flag.Var(&configExt, "config", "comma-separated list of intervals to use between events")

	flag.Parse()

	x := strings.Split(configExt[0], "=")

	path, v := x[0], strings.Join(x[1:], "=")

	fmt.Println(path, v)

	pathElts := strings.Split(path, ".")

	cfg.SetIn(pathElts, v)

	fmt.Printf("%#v\n", cfg)
}

// func SetIn(m config.Imap, path string, val string) (config.Imap, error) {
// 	pathElts := strings.Split(path, ".")
// 	m[pathElts]
// }
