package protocol

type SimpleProtocolMessage struct {
	data []byte
}

func (m *SimpleProtocolMessage) Bytes() []byte {
	return m.data
}
