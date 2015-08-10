package snmp

import (
	"github.com/Preetam/snmp"
	"internal/message"
)

const ClassName = "snmp"

type Class struct {
	session  *snmp.Session
	inbound  chan *message.Message
	outbound chan *message.Message
}

func (c *Class) Name() string {
	return ClassName
}

func (c *Class) Category() string {
	return "comm"
}

func (c *Class) InboundMessages() chan *message.Message {
	return c.inbound
}

func (c *Class) OutboundMessages() chan *message.Message {
	return c.outbound
}
