package source

import (
	"github.com/Preetam/appflow"
	"internal/message"
)

const CommAppflowClassName = "sflow"

type CommAppflowClass struct {
	inbound  chan *appflow.HTTPFlowData
	outbound chan *message.Message
}

func NewAppflowClass(
	inbound chan *appflow.HTTPFlowData,
	outbound chan *message.Message) *CommAppflowClass {
	c := &CommAppflowClass{
		inbound:  inbound,
		outbound: outbound,
	}
	go c.generateMessages()
	return c
}

func (c *CommAppflowClass) Name() string {
	return CommSFlowClassName
}

func (c *CommAppflowClass) Category() string {
	return "comm"
}

func (c *CommAppflowClass) OutboundMessages() chan *message.Message {
	return c.outbound
}

func (c *CommAppflowClass) generateMessages() {
	for _ = range c.inbound {
		// TODO: use HTTP flow data
	}
}
