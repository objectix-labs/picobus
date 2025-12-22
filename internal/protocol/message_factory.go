package protocol

type Message interface {
	Bytes() []byte
}

type MessageCodec struct{}

func NewMessageCodec() *MessageCodec {
	return &MessageCodec{}
}

func (f *MessageCodec) Decode(data []byte) (Message, error) {
	return &SimpleProtocolMessage{data: data}, nil
}

func (f *MessageCodec) Encode(msg Message) []byte {
	return msg.Bytes()
}
