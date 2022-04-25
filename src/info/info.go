package info

import (
	"fmt"
	"os"
)

func Println(a ...interface{}) (int, error) {
	return fmt.Fprintln(os.Stderr, a...)
}

func Printf(format string, a ...interface{}) (int, error) {
	return fmt.Fprintf(os.Stderr, format, a...)
}
