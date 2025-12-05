package main

import (
	"net"
	"strings"

	"github.com/objectix-labs/picobus/internal/logging"
	"github.com/objectix-labs/picobus/internal/socket"
	"github.com/spf13/viper"
)

func main() {
	setupConfiguration()

	level := strings.ToLower(viper.GetString("log_level"))
	format := strings.ToLower(viper.GetString("log_format"))
	socketPath := viper.GetString("socket")

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

func setupConfiguration() {
	defaults := map[string]string{
		"log_level":  "info",
		"log_format": "text",
		"socket":     "/tmp/picobus.sock",
	}

	for key, value := range defaults {
		viper.SetDefault(key, value)
	}

	viper.SetEnvPrefix("pico")
	viper.AutomaticEnv()
}
