package network

import (
	"fmt"
	"sync"
	"time"

	"github.com/objectix-labs/picobus/internal/logging"
	"github.com/objectix-labs/picobus/internal/protocol"
)

type ConnectionManager struct {
	serverSocket           *PicobusSocket
	protocolMessageFactory *protocol.MessageCodec

	connectionMap          sync.Map // map[string]*Connection
	pendingConnectionQueue chan *Connection
	inboundProtocolQueue   chan protocol.Message
}

func NewConnectionManager(
	serverSocket *PicobusSocket,
	protocolMessageFactory *protocol.MessageCodec,
	backlogSize int,
	inboundProtocolQueue chan protocol.Message,
) *ConnectionManager {
	return &ConnectionManager{
		serverSocket:           serverSocket,
		protocolMessageFactory: protocolMessageFactory,
		connectionMap:          sync.Map{},
		pendingConnectionQueue: make(chan *Connection, backlogSize),
		inboundProtocolQueue:   inboundProtocolQueue,
	}
}

// Start begins accepting and handling incoming connections.
func (e *ConnectionManager) Start() {
	// Start handling inbound connections
	go e.handleConnections()

	// Start server activity
	go e.serverSocket.ListenAndServe(e.pendingConnectionQueue)
}

// Terminate stops the connection manager and closes all connections and server socket.
func (e *ConnectionManager) Terminate(gracefulTimeout time.Duration) error {
	if err := e.serverSocket.Close(gracefulTimeout); err != nil {
		return fmt.Errorf("termination failed: %w", err)
	}

	return nil
}

func (e *ConnectionManager) Write(connID string, msg protocol.Message) error {
	value, ok := e.connectionMap.Load(connID)
	if !ok {
		return fmt.Errorf("connection with ID %s not found", connID)
	}

	connection, ok := value.(*Connection)
	if !ok {
		return fmt.Errorf("invalid connection type for ID %s", connID)
	}

	return connection.Write(msg.Bytes())
}

func (e *ConnectionManager) handleConnections() {
	// Wait for inbound connections and handle them
	for conn := range e.pendingConnectionQueue {
		// New connection has become available. Register it with this endpoint.
		e.registerConnection(conn)

		// Handle connection in new goroutine
		go conn.Handle()
	}
}

func (e *ConnectionManager) registerConnection(conn *Connection) {
	// Remember the connection
	e.connectionMap.Store(conn.ID(), conn)

	// Process read messages from connection, until connection is closed
	go func() {
		for msg := range conn.Messages() {
			// Decode protocol message
			protoMsg, err := e.protocolMessageFactory.Decode(msg)

			if err != nil {
				logging.Error(
					"failed to decode protocol message from connection",
					"connectionID", conn.ID(),
					"error", err,
				)
				continue
			}

			// Send decoded protocol message to inbound queue
			e.inboundProtocolQueue <- protoMsg
		}

		// At this point, connection is closed. Forget the connection.
		e.connectionMap.Delete(conn.ID())
	}()
}
