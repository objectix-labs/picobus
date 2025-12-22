package network

import (
	"context"
	"net"
	"sync"

	"github.com/objectix-labs/picobus/internal/logging"
)

type ConnectionHandler struct {
	maxMessageSize uint32
}

func NewConnectionHandler(maxMessageSize uint32) *ConnectionHandler {
	return &ConnectionHandler{
		maxMessageSize: maxMessageSize,
	}
}

func (h *ConnectionHandler) Handle(ctx context.Context, conn net.Conn, wg *sync.WaitGroup) {
	// When this handler is terminated, we need to release a wait group slot
	// and close the associated connection.
	defer wg.Done()
	defer conn.Close()
	defer logging.Info("terminated connection handler", "remoteAddr", conn.RemoteAddr().String())

	logging.Info("accepted new connection", "remoteAddr", conn.RemoteAddr().String())

	// Detect termination of this handler
	done := make(chan struct{})
	defer close(done)

	// Write queue containing the outbound messages to be sent on this connection
	writeQueue := make(chan []byte, maxOutboundMessages)
	defer close(writeQueue)

	// Observe context cancellation to terminate connection handler, and write any outbound messages
	// in the meantime.
	go h.observeConnection(ctx, conn, writeQueue, done)

	// Attach message reader and writer to our connection
	messageReader := NewMessageReader(conn, h.maxMessageSize)

	// We now enter the inbound message loop for this connection.
	for {
		// Read next message from connection
		msg, err := messageReader.Read()
		if err != nil {
			// Could not read message, leave connection handler
			logging.Error("failed to read message", "error", err, "remoteAddr", conn.RemoteAddr().String())
			return
		}

		logging.Info("received message", "message", msg, "remoteAddr", conn.RemoteAddr().String())
	}
}

func (h *ConnectionHandler) observeConnection(
	ctx context.Context,
	conn net.Conn,
	writeQueue chan []byte,
	done chan struct{},
) {
	messageWriter := NewMessageWriter(conn, h.maxMessageSize)

	for {
		select {
		case <-ctx.Done():
			// Context cancelled, close socket to unblock any pending reads
			conn.Close()
			return
		case <-done:
			// Done signal received, exit goroutine
			return
		case outMsg, ok := <-writeQueue:
			if ok {
				// Write outbound message to connection
				err := messageWriter.Write(outMsg)
				if err != nil {
					logging.Warn("failed to write message", "error", err, "remoteAddr", conn.RemoteAddr().String())
					conn.Close()
					return
				}
				continue
			}

			// At this point the writeQueue channel is closed, exit goroutine
			return
		}
	}
}

const maxOutboundMessages = 10
