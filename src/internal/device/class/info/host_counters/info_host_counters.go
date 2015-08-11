package host_counters

import (
	"net"

	"github.com/Preetam/sflow"
	"internal/message"
)

const ClassName = "host-counters"

type Class struct {
	sourceAddress net.IP
	outbound      chan *message.Message
}

func NewClass(sourceAddress net.IP, outbound chan *message.Message) *Class {
	c := &Class{
		sourceAddress: sourceAddress,
		outbound:      outbound,
	}
	return c
}

func (c *Class) Name() string {
	return ClassName
}

func (c *Class) Category() string {
	return "info"
}

func (c *Class) OutboundMessages() chan *message.Message {
	return c.outbound
}

func (c *Class) Process(m *message.Message) {
	switch m.Type {
	case "CPU":
		cpuCounters := m.Content.(sflow.HostCPUCounters)
		c.handleCPUCounters(cpuCounters)
	default:
		// Drop.
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
