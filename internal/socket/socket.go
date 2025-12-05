package socket

import (
	"fmt"
	"net"
	"os"
)

// PicobusSocket wraps access to a Unix domain socket
// and accepts inbound connections to that socket. Accepted connections
// are then handled with the specified handler function.

type PicobusSocket struct {
	path     string
	handler  PicobusSocketHandler
	listener net.Listener
}

type PicobusSocketHandler func(conn net.Conn)

func NewPicobusSocket(path string, handler PicobusSocketHandler) *PicobusSocket {
	return &PicobusSocket{
		path:    path,
		handler: handler,
	}
}

// ListenAndServe starts listening on the Unix domain socket
// and serves incoming connections using the specified handler.
func (s *PicobusSocket) ListenAndServe() error {
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
		if err == net.ErrClosed {
			return nil
		} else if err != nil {
			return fmt.Errorf("failed to accept connection: %w", err)
		}

		go s.handler(conn)
	}
}

func (s *PicobusSocket) Close() error {
	err := s.listener.Close()
	if err != nil {
		return fmt.Errorf("failed to close socket listener: %w", err)
	}
	return nil
}

func unlink(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing socket file %s: %w", path, err)
	}
	return nil
}
