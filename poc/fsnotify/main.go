package main

import (
	"fmt"
	"log"
	"os"

	"github.com/fsnotify/fsnotify"
)

func main() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event)
				// if event.Op&fsnotify.Write == fsnotify.Write {
				// log.Println("modified file:", event.Name)
				// }
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	path, err := os.Getwd()
	fmt.Println(path)
	if err != nil {
		fmt.Println(err)
	}

	err = watcher.Add(path)
	if err != nil {
		log.Fatal(err)
	}
	// err = watcher.Add(path + "/foo")
	// if err != nil {
	// 	// log.Fatal(err)
	// 	fmt.Println("no file")
	// }
	<-done
}
