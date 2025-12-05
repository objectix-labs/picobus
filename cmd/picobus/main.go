package main

import "github.com/objectix-labs/picobus/internal/logging"

func main() {
	logging.Init("picobus")
	logging.Info("picobus started")
	logging.Debug("this is a debug message")
}
