package main

import (
	"net"
	"os"
	"strings"

	"github.com/objectix-labs/picobus/internal/logging"
	"github.com/objectix-labs/picobus/internal/socket"
)

func main() {
	level := strings.ToLower(os.Getenv("PICOBUS_LOG_LEVEL"))
	format := strings.ToLower(os.Getenv("PICOBUS_LOG_FORMAT"))
	socketPath := os.Getenv("PICOBUS_SOCKET_PATH")

	if socketPath == "" {
		socketPath = "/tmp/picobus.sock"
	}

	if level == "" {
		level = "info"
	}

	if format == "" {
		format = "text"
	}

	logging.Init("picobus", level, format)
	logging.Info("picobus started")
	logging.Debug("this is a debug message")

	// setup server socket, then wait and serve connections
	serverSocket := socket.NewPicobusSocket(socketPath, func(conn net.Conn) {
		logging.Info("accepted new connection", "remoteAddr", conn.RemoteAddr().String())
		// handle connection...
	})

	if err := serverSocket.ListenAndServe(); err != nil {
		logging.Error("listen and serve failed", "error", err)
	}
}
