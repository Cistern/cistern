package message

type Class interface {
	Name() string
	Category() string
}

type Emitter interface {
	OutboundMessages() chan *Message
}

type Collector interface {
	InboundMessages() chan *Message
}
