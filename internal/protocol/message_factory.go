package protocol

type Message interface {
	Bytes() []byte
}

type SimpleProtocolMessage struct {
	data []byte
}

func (m *SimpleProtocolMessage) Bytes() []byte {
	return m.data
}

type MessageFactory struct{}

func NewMessageFactory() *MessageFactory {
	return &MessageFactory{}
}

func (f *MessageFactory) CreateMessage(data []byte) Message {
	return NewMessageFromBytes(data)
}

func NewMessageFromBytes(data []byte) Message {
	// For demonstration purposes, we return a simple implementation
	return &SimpleProtocolMessage{data: data}
}
