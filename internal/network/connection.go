package network

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

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
	reader         *bufio.Reader
	writer         *bufio.Writer
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
		reader:         bufio.NewReader(conn),
		writer:         bufio.NewWriter(conn),
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

	// We now enter the inbound message loop for this connection.
	for {
		// Read next message from connection
		msg, err := c.read()
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
	for {
		select {
		case <-c.context.Done():
			// Context cancelled, close socket to unblock any pending reads
			c.conn.Close()
			return
		case outMsg, ok := <-c.writeQueue:
			if ok {
				// Write outbound message to connection
				err := c.write(outMsg)
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

func (c *Connection) read() ([]byte, error) {
	// Idle timeout while waiting for new message
	c.conn.SetReadDeadline(time.Now().Add(maxIdleTimeout))

	// Read the size of the message (first 4 bytes)
	msgSizeBuf := make([]byte, msgSizeFieldLen)
	if n, err := io.ReadFull(c.reader, msgSizeBuf); err != nil || n != 4 {
		// could not read message size
		return nil, fmt.Errorf("failed to read message size: %w", err)
	}

	// Convert from network order to host order
	msgSize := binary.BigEndian.Uint32(msgSizeBuf)

	if msgSize > c.maxMessageSize {
		// message size exceeds maximum allowed size
		return nil, fmt.Errorf(
			"message size %d exceeded maximum allowed message size of %d bytes",
			msgSize,
			c.maxMessageSize,
		)
	}

	// Allocate buffer for the message
	msgBuf := make([]byte, msgSize)

	// Start per message tiemeout to avoid slow senders and DOS attacks
	c.conn.SetReadDeadline(time.Now().Add(maxMessageTimeout))

	// Read the actual message payload
	if n, err := io.ReadFull(c.reader, msgBuf); err != nil || uint32(n) != msgSize {
		// could not read full message payload
		return nil, fmt.Errorf("failed to read message payload: %w", err)
	}

	// Clear read deadline
	c.conn.SetReadDeadline(time.Time{})

	return msgBuf, nil
}

func (c *Connection) write(msg []byte) error {
	if uint32(len(msg)) > c.maxMessageSize {
		return fmt.Errorf(
			"message size %d exceeded maximum allowed message size of %d bytes",
			len(msg),
			c.maxMessageSize,
		)
	}

	// Set write deadline
	c.conn.SetWriteDeadline(time.Now().Add(maxMessageTimeout))

	// Prepare message size buffer
	msgSizeBuf := make([]byte, msgSizeFieldLen)
	binary.BigEndian.PutUint32(msgSizeBuf, uint32(len(msg)))

	// Write message size
	if n, err := c.writer.Write(msgSizeBuf); err != nil || n != msgSizeFieldLen {
		return fmt.Errorf("failed to write message size: %w", err)
	}

	// Write message payload
	if n, err := c.writer.Write(msg); err != nil || n != len(msg) {
		return fmt.Errorf("failed to write message payload: %w", err)
	}

	if err := c.writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush message to connection: %w", err)
	}

	// Clear write deadline
	c.conn.SetWriteDeadline(time.Time{})

	return nil
}

// How long does server wait for data from idle connection before closing it?
const maxIdleTimeout = 5 * time.Minute

// How long oes the server wait for a full message to be received?
const maxMessageTimeout = 3 * time.Second

// Message size field length (in bytes).
const msgSizeFieldLen = 4

// How many messages can be queued for reading/writing per connection.
const maxMessages = 10
