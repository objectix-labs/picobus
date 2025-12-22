package main

import (
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/caarlos0/env"
	"github.com/objectix-labs/picobus/internal/logging"
	"github.com/objectix-labs/picobus/internal/network"
	"github.com/objectix-labs/picobus/internal/protocol"
)

func main() {
	var config config
	if err := env.Parse(&config); err != nil {
		panic(err)
	}

	logging.Init("picobus", strings.ToLower(config.LogLevel), strings.ToLower(config.LogFormat))
	logging.Info("picobus started")

	// setup server socket, then wait and serve connections
	serverSocket := network.NewPicobusSocket(
		config.SocketPath,
	)

	// Receives inbound protocol messages from all connections
	protocolMessages := make(chan protocol.Message, maxInboundMessages)

	endpoint := network.NewConnectionManager(
		serverSocket,
		protocol.NewMessageCodec(),
		maxPendingConnections,
		protocolMessages,
	)

	endpoint.Start()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logging.Info("shutdown signal received")

	// close server socket
	if err := endpoint.Terminate(maxShutdownWait); err != nil {
		logging.Error("server shutdown failed", "error", err)
	}

	logging.Info("server exited properly")
}

type config struct {
	LogLevel   string `env:"LOG_LEVEL"  envDefault:"info"`
	LogFormat  string `env:"LOG_FORMAT" envDefault:"text"`
	SocketPath string `env:"SOCKET"     envDefault:"/tmp/picobus.sock"`
}

const maxPendingConnections = 10
const maxShutdownWait = 10 * time.Second
const maxInboundMessages = 100
