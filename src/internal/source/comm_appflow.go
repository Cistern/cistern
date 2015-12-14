package source

import (
	"github.com/Preetam/appflow"

	"internal/clock"
	"internal/message"
)

const CommAppflowClassName = "appflow"

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
	for flowData := range c.inbound {
		m := &message.Message{
			Class:     "appflow",
			Timestamp: clock.Time(),
			Content:   flowData,
		}
		c.outbound <- m
	}
}
