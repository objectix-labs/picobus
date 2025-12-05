package main

import "github.com/objectix-labs/picobus/internal/logging"

func main() {
	logging.Init()
	logging.Info("picobus started")
	logging.Debug("this is a debug message")
}
