package message

type Message struct {
	Global    bool        // A message in the global scope. False corresponds to a device scope.
	Class     string      // Class name
	Type      string      // Type within class
	Timestamp int64       // When the message was sent (as a Unix timestamp)
	Content   interface{} // Arbitrary content
}

func NewMessageChannel() chan *Message {
	return make(chan *Message, 1)
}
