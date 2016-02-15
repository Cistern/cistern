package source

import (
	"net"

	"github.com/Cistern/sflow"
	"internal/clock"
	"internal/message"
)

const CommSFlowClassName = "sflow"

type CommSFlowClass struct {
	sourceAddress net.IP
	inbound       chan *sflow.Datagram
	outbound      chan *message.Message
}

func NewCommSFlowClass(
	sourceAddress net.IP,
	inbound chan *sflow.Datagram,
	outbound chan *message.Message) *CommSFlowClass {
	c := &CommSFlowClass{
		sourceAddress: sourceAddress,
		inbound:       inbound,
		outbound:      outbound,
	}
	go c.generateMessages()
	return c
}

func (c *CommSFlowClass) Name() string {
	return CommSFlowClassName
}

func (c *CommSFlowClass) Category() string {
	return "comm"
}

func (c *CommSFlowClass) OutboundMessages() chan *message.Message {
	return c.outbound
}

func (c *CommSFlowClass) generateMessages() {
	for dgram := range c.inbound {
		for _, sample := range dgram.Samples {
			for _, record := range sample.GetRecords() {
				switch record.(type) {
				case sflow.HostCPUCounters,
					sflow.HostMemoryCounters,
					sflow.HostDiskCounters,
					sflow.HostNetCounters:
					c.handleHostCounters(record)
				case sflow.GenericInterfaceCounters:
					c.handleSwitchCounters(record)
				case sflow.RawPacketFlow:
					c.handleRawPacketFlow(record)
				default:
					// Unknown type. Drop.
				}
			}
		}
	}
}

func (c *CommSFlowClass) handleHostCounters(record sflow.Record) {
	m := &message.Message{
		Class:     "host-counters",
		Timestamp: clock.Time(),
		Content:   record,
	}
	switch record.(type) {
	case sflow.HostCPUCounters:
		m.Type = "CPU"
	case sflow.HostMemoryCounters:
		m.Type = "Memory"
	case sflow.HostDiskCounters:
		m.Type = "Disk"
	case sflow.HostNetCounters:
		m.Type = "Net"
	default:
		return
	}
	c.outbound <- m
}

func (c *CommSFlowClass) handleSwitchCounters(record sflow.Record) {
	m := &message.Message{
		Class:   "switch-counters",
		Content: record,
	}
	switch record.(type) {
	case sflow.GenericInterfaceCounters:
		m.Type = "GenericInterface"
	default:
		return
	}
	c.outbound <- m
}

func (c *CommSFlowClass) handleRawPacketFlow(record sflow.Record) {
	m := &message.Message{
		Class:   "packet-flow",
		Content: record,
	}
	c.outbound <- m
}
