package network

import (
	"fmt"
	"sync"

	"github.com/objectix-labs/picobus/internal/protocol"
)

type ConnectionManager struct {
	serverSocket           *PicobusSocket
	protocolMessageFactory *protocol.MessageCodec

	connectionMap sync.Map // map[string]*Connection
}

func NewConnectionManager(
	serverSocket *PicobusSocket,
	protocolMessageFactory *protocol.MessageCodec,
) *ConnectionManager {
	return &ConnectionManager{
		serverSocket:           serverSocket,
		protocolMessageFactory: protocolMessageFactory,
		connectionMap:          sync.Map{},
	}
}

func (e *ConnectionManager) Start() {
	// Start handling inbound connections
	go e.handleConnections()

	// Start server activity
	go e.serverSocket.ListenAndServe()
}

func (e *ConnectionManager) Terminate() error {
	if err := e.serverSocket.Close(); err != nil {
		return fmt.Errorf("termination failed: %w", err)
	}

	return nil
}

func (e *ConnectionManager) WriteProtocolMessage(connID string, msg protocol.Message) error {
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
	for conn := range e.serverSocket.ConnectionQueue() {
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
		for msg := range conn.Inbound() {
			fmt.Printf("PicobusEndpoint received message: %v\n", msg)
		}

		// At this point, connection is closed. Forget the connection.
		e.connectionMap.Delete(conn.ID())
	}()
}
