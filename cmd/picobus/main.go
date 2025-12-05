package main

import (
	"net"

	"github.com/objectix-labs/picobus/internal/logging"
	"github.com/objectix-labs/picobus/internal/socket"
)

func main() {
	logging.Init("picobus")
	logging.Info("picobus started")
	logging.Debug("this is a debug message")

	// setup server socket, then wait and serve connections
	serverSocket := socket.NewPicobusSocket("/tmp/picobus.sock", func(conn net.Conn) {
		logging.Info("accepted new connection", "remoteAddr", conn.RemoteAddr().String())
		// handle connection...
	})

	if err := serverSocket.ListenAndServe(); err != nil {
		logging.Error("listen and serve failed", "error", err)
	}
}
