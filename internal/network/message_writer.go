package network

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

type MessageWriter struct {
	conn           net.Conn
	writer         *bufio.Writer
	maxMessageSize uint32
}

func NewMessageWriter(conn net.Conn, maxMessageSize uint32) *MessageWriter {
	return &MessageWriter{
		conn:           conn,
		writer:         bufio.NewWriter(conn),
		maxMessageSize: maxMessageSize,
	}
}

func (w *MessageWriter) Write(msg []byte) error {
	if uint32(len(msg)) > w.maxMessageSize {
		return fmt.Errorf(
			"message size %d exceeded maximum allowed message size of %d bytes",
			len(msg),
			w.maxMessageSize,
		)
	}

	// Set write deadline
	w.conn.SetWriteDeadline(time.Now().Add(maxMessageTimeout))

	// Prepare message size buffer
	msgSizeBuf := make([]byte, msgSizeFieldLen)
	binary.BigEndian.PutUint32(msgSizeBuf, uint32(len(msg)))

	// Write message size
	if n, err := w.writer.Write(msgSizeBuf); err != nil || n != msgSizeFieldLen {
		return fmt.Errorf("failed to write message size: %w", err)
	}

	// Write message payload
	if n, err := w.writer.Write(msg); err != nil || n != len(msg) {
		return fmt.Errorf("failed to write message payload: %w", err)
	}

	if err := w.writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush message to connection: %w", err)
	}

	// Clear write deadline
	w.conn.SetWriteDeadline(time.Time{})

	return nil
}
