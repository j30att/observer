package main

import (
	stdlog "log"
	"os"
)

func main() {
	if len(os.Args) == 0 {
		stdlog.Fatal("allowed in main")
	}
	if len(os.Args) == 1 {
		os.Exit(0)
	}
}
