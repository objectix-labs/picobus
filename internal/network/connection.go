package network

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/objectix-labs/picobus/internal/logging"
)

// Connection represents a network connection to a connected Picobus client.
type Connection struct {
	id             string
	context        context.Context
	conn           net.Conn
	waitGroup      *sync.WaitGroup
	writeQueue     chan []byte
	readQueue      chan []byte
	isClosed       atomic.Bool
	maxMessageSize uint32
}

// NewConnection creates a new Connection instance.
func NewConnection(
	ctx context.Context,
	conn net.Conn,
	wg *sync.WaitGroup,
	maxMessageSize uint32,
) *Connection {
	// Increment wait group for this connection
	wg.Add(1)

	return &Connection{
		id:             uuid.New().String(),
		context:        ctx,
		conn:           conn,
		waitGroup:      wg,
		maxMessageSize: maxMessageSize,
		writeQueue:     make(chan []byte, maxMessages),
		readQueue:      make(chan []byte, maxMessages),
	}
}

// ID returns the unique identifier of the connection.
func (c *Connection) ID() string {
	return c.id
}

// Write asynchronously sends a message to the connection's outbound message queue.
// If the connection is closed or the write queue is full, an error is returned.
func (c *Connection) Write(msg []byte) error {
	if c.isClosed.Load() {
		return fmt.Errorf("cannot write on closed connection")
	}

	select {
	case c.writeQueue <- msg:
		return nil
	default:
		return fmt.Errorf("write queue is full, cannot write message to connection")
	}
}

// Messages returns the channel for receiving inbound messages from the connection.
func (c *Connection) Messages() chan []byte {
	return c.readQueue
}

// Handle starts processing the connection for inbound and outbound messages. Method
// blocks until the connection is closed.
func (c *Connection) Handle() {
	// When this handler is terminated, we need to close the connection.
	defer logging.Info("terminated connection", "remoteAddr", c.conn.RemoteAddr().String())
	defer c.close()

	logging.Info("handling new connection", "remoteAddr", c.conn.RemoteAddr().String())

	// Observe context cancellation to terminate connection handler, and write any outbound messages
	// in the meantime.
	go c.observeConnection()

	// Attach message reader and writer to our connection
	messageReader := NewMessageReader(c.conn, c.maxMessageSize)

	// We now enter the inbound message loop for this connection.
	for {
		// Read next message from connection
		msg, err := messageReader.Read()
		if err != nil {
			// Could not read message, leave connection handler
			logging.Error(
				"failed to read message from connection",
				"error",
				err,
				"remoteAddr",
				c.conn.RemoteAddr().String(),
			)
			return
		}

		logging.Info(
			"received message on connection",
			"message",
			msg,
			"remoteAddr",
			c.conn.RemoteAddr().String(),
		)

		// Send inbound message to read queue. If read queue is full, drop the message.
		select {
		case c.readQueue <- msg:
			// Message successfully sent to read queue
		default:
			// Read queue is full, drop the message
			logging.Warn(
				"dropping inbound message, connection's inbound message queue is full",
				"remoteAddr",
				c.conn.RemoteAddr().String(),
			)
		}
	}
}

func (c *Connection) observeConnection() {
	messageWriter := NewMessageWriter(c.conn, c.maxMessageSize)
	for {
		select {
		case <-c.context.Done():
			// Context cancelled, close socket to unblock any pending reads
			c.conn.Close()
			return
		case outMsg, ok := <-c.writeQueue:
			if ok {
				// Write outbound message to connection
				err := messageWriter.Write(outMsg)
				if err != nil {
					logging.Warn(
						"failed to write message on connection",
						"error",
						err,
						"remoteAddr",
						c.conn.RemoteAddr().String(),
					)
					c.conn.Close()
					return
				}
				continue
			}

			// At this point the writeQueue channel is closed, exit goroutine
			return
		}
	}
}

func (c *Connection) close() {
	if c.isClosed.Load() {
		return
	}

	c.isClosed.Store(true)

	// Close read and write queues
	close(c.readQueue)
	close(c.writeQueue)

	// Close underlying connection
	c.conn.Close()

	// Release wait group slot
	c.waitGroup.Done()
}

const maxMessages = 10
