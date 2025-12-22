package network

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/objectix-labs/picobus/internal/logging"
)

// PicobusSocket wraps access to a Unix domain socket
// and accepts inbound connections to that socket. Accepted connections
// are then handled with the specified handler function.

type PicobusSocket struct {
	path      string
	listener  net.Listener
	waitGroup sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewPicobusSocket(path string) *PicobusSocket {
	ctx, cancel := context.WithCancel(context.Background())

	return &PicobusSocket{
		path:   path,
		ctx:    ctx,
		cancel: cancel,
	}
}

// ListenAndServe starts listening on the Unix domain socket
// and serves incoming connections using the specified handler.
func (s *PicobusSocket) ListenAndServe(connQueue chan *Connection) error {
	// Unbind any existing socket file
	unlink(s.path)

	listener, err := net.Listen("unix", s.path)
	if err != nil {
		return fmt.Errorf("failed to listen on unix socket %s: %w", s.path, err)
	}

	defer func() {
		_ = listener.Close()
		unlink(s.path)
	}()

	s.listener = listener

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				// listener closed during shutdown
				return nil
			default:
				logging.Warn("accept error:", err)
				continue
			}
		}

		connection := NewConnection(s.ctx, conn, &s.waitGroup, defaultMaxMessageSize)
		connQueue <- connection
	}
}

func (s *PicobusSocket) Close(gracefulTimeout time.Duration) error {
	// Cancel our context to signal handlers to stop and
	s.cancel()

	// Stop accepting new connections
	err := s.listener.Close()
	if err != nil {
		return fmt.Errorf("failed to close socket listener: %w", err)
	}

	// Wait for all active connections to finish, but do not linger more than X seconds
	done := make(chan struct{})

	go func() {
		s.waitGroup.Wait()
		close(done)
	}()

	// wait for either all handlers to finish or timeout
	select {
	case <-done:
		// all handlers finished
		logging.Info("all active socket handlers finished")
	case <-time.After(gracefulTimeout):
		// timeout reached
		logging.Warn("timeout waiting for socket handlers to finish")
	}

	return nil
}

func unlink(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing socket file %s: %w", path, err)
	}
	return nil
}

const defaultMaxMessageSize = 1024 * 1024 // 1 MB
