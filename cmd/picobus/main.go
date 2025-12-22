package main

import (
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/caarlos0/env"
	"github.com/objectix-labs/picobus/internal/logging"
	"github.com/objectix-labs/picobus/internal/network"
)

func main() {
	var config config
	if err := env.Parse(&config); err != nil {
		panic(err)
	}

	logging.Init("picobus", strings.ToLower(config.LogLevel), strings.ToLower(config.LogFormat))
	logging.Info("picobus started")

	// setup connection handler
	const maxMessageSize = 1024 * 1024 // 1 MB
	connectionHandler := network.NewConnectionHandler(maxMessageSize)

	// setup server socket, then wait and serve connections
	serverSocket := network.NewPicobusSocket(
		config.SocketPath,
		connectionHandler,
	)

	go func() {
		// Wait and listen for incoming connections to our server socket
		if err := serverSocket.ListenAndServe(); err != nil {
			logging.Error("listen and serve failed", "error", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logging.Info("shutdown signal received")

	// close server socket
	if err := serverSocket.Close(); err != nil {
		logging.Error("server shutdown failed", "error", err)
	}

	// Wait for

	logging.Info("server exited properly")
}

type config struct {
	LogLevel   string `env:"LOG_LEVEL"  envDefault:"info"`
	LogFormat  string `env:"LOG_FORMAT" envDefault:"text"`
	SocketPath string `env:"SOCKET"     envDefault:"/tmp/picobus.sock"`
}
