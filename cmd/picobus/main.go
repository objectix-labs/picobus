package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/caarlos0/env"
	"github.com/objectix-labs/picobus/internal/logging"
	"github.com/objectix-labs/picobus/internal/socket"
)

func main() {
	var config config
	if err := env.Parse(&config); err != nil {
		panic(err)
	}

	logging.Init("picobus", strings.ToLower(config.LogLevel), strings.ToLower(config.LogFormat))
	logging.Info("picobus started")
	logging.Debug("this is a debug message")

	// setup server socket, then wait and serve connections
	serverSocket := socket.NewPicobusSocket(
		config.SocketPath,
		func(ctx context.Context, conn net.Conn, wg *sync.WaitGroup) {
			// When this handler is terminated, we need to release a wait group slot
			// and close the associated connection.
			defer wg.Done()
			defer conn.Close()

			logging.Info("accepted new connection", "remoteAddr", conn.RemoteAddr().String())

			// TODO: handle connection

			// Wait for shutdown signal
			<-ctx.Done()
			logging.Info("terminating connection handler", "remoteAddr", conn.RemoteAddr().String())
		},
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
