package source

import (
	"github.com/Cistern/appflow"

	"github.com/Cistern/cistern/clock"
	"github.com/Cistern/cistern/message"
)

const CommAppFlowClassName = "appflow"

type CommAppFlowClass struct {
	inbound  chan *appflow.HTTPFlowData
	outbound chan *message.Message
}

func NewCommAppFlowClass(
	inbound chan *appflow.HTTPFlowData,
	outbound chan *message.Message) *CommAppFlowClass {
	c := &CommAppFlowClass{
		inbound:  inbound,
		outbound: outbound,
	}
	go c.generateMessages()
	return c
}

func (c *CommAppFlowClass) Name() string {
	return CommAppFlowClassName
}

func (c *CommAppFlowClass) Category() string {
	return "comm"
}

func (c *CommAppFlowClass) OutboundMessages() chan *message.Message {
	return c.outbound
}

func (c *CommAppFlowClass) generateMessages() {
	for flowData := range c.inbound {
		m := &message.Message{
			Class:     "http-flow",
			Timestamp: clock.Time(),
			Content:   flowData,
		}
		c.outbound <- m
	}
}
