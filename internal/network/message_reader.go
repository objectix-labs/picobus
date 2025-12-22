package network

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)

type MessageReader struct {
	conn           net.Conn
	reader         *bufio.Reader
	maxMessageSize uint32
}

func NewMessageReader(conn net.Conn, maxMessageSize uint32) *MessageReader {
	return &MessageReader{
		conn:           conn,
		reader:         bufio.NewReader(conn),
		maxMessageSize: maxMessageSize,
	}
}

func (r *MessageReader) Read() ([]byte, error) {
	// Idle timeout while waiting for new message
	r.conn.SetReadDeadline(time.Now().Add(maxIdleTimeout))

	// Read the size of the message (first 4 bytes)
	msgSizeBuf := make([]byte, msgSizeFieldLen)
	if n, err := io.ReadFull(r.reader, msgSizeBuf); err != nil || n != 4 {
		// could not read message size
		return nil, fmt.Errorf("failed to read message size: %w", err)
	}

	// Convert from network order to host order
	msgSize := binary.BigEndian.Uint32(msgSizeBuf)

	if msgSize > r.maxMessageSize {
		// message size exceeds maximum allowed size
		return nil, fmt.Errorf(
			"message size %d exceeded maximum allowed message size of %d bytes",
			msgSize,
			r.maxMessageSize,
		)
	}

	// Allocate buffer for the message
	msgBuf := make([]byte, msgSize)

	// Start per message tiemeout to avoid slow senders and DOS attacks
	r.conn.SetReadDeadline(time.Now().Add(maxMessageTimeout))

	// Read the actual message payload
	if n, err := io.ReadFull(r.reader, msgBuf); err != nil || uint32(n) != msgSize {
		// could not read full message payload
		return nil, fmt.Errorf("failed to read message payload: %w", err)
	}

	// Clear read deadline
	r.conn.SetReadDeadline(time.Time{})

	return msgBuf, nil
}

// How long does server wait for data from idle connection before closing it?
const maxIdleTimeout = 5 * time.Minute

// How long oes the server wait for a full message to be received?
const maxMessageTimeout = 3 * time.Second

// Message size field length (in bytes).
const msgSizeFieldLen = 4
