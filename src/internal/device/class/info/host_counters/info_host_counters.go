package host_counters

import (
	"net"

	"github.com/Preetam/sflow"
	"internal/message"
)

const ClassName = "host-counters"

type Class struct {
	sourceAddress net.IP
	inbound       chan *message.Message
	outbound      chan *message.Message
}

func NewClass(sourceAddress net.IP, outbound chan *message.Message) *Class {
	c := &Class{
		sourceAddress: sourceAddress,
		inbound:       message.NewMessageChannel(),
		outbound:      outbound,
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

func (c *Class) OutboundMessages() chan *message.Message {
	return c.outbound
}

func (c *Class) handleMessages() {
	for m := range c.inbound {
		switch m.Type {
		case "CPU":
			cpuCounters := m.Content.(sflow.HostCPUCounters)
			c.handleCPUCounters(cpuCounters)
		default:
			// Drop.
		}
	}
}

func (c *Class) handleCPUCounters(counters sflow.HostCPUCounters) {
	select {
	case c.outbound <- &message.Message{
		Class:   "debug",
		Content: counters,
	}:
	default:
		// Drop.
	}
}
