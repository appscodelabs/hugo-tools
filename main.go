package main

import (
	"os"

	logs "github.com/appscode/go/log/golog"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
