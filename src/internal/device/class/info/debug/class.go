package debug

import (
	"log"
	"net"

	"internal/message"
)

const ClassName = "debug"

type Class struct {
	sourceAddress net.IP
	inbound       chan *message.Message
}

func NewClass(sourceAddress net.IP) *Class {
	c := &Class{
		sourceAddress: sourceAddress,
		inbound:       make(chan *message.Message),
	}
	go c.handleMessages()
	return c
}

func (c *Class) Name() string {
	return ClassName
}

func (c *Class) Category() string {
	return "info"
}

func (c *Class) InboundMessages() chan *message.Message {
	return c.inbound
}

func (c *Class) handleMessages() {
	for m := range c.inbound {
		log.Println(m)
	}
}
