package sflow

import (
	"net"

	"github.com/Preetam/sflow"
	"internal/clock"
	"internal/message"
)

const ClassName = "sflow"

type Class struct {
	sourceAddress net.IP
	inbound       chan *sflow.Datagram
	outbound      chan *message.Message
}

func NewClass(sourceAddress net.IP, inbound chan *sflow.Datagram, outbound chan *message.Message) *Class {
	c := &Class{
		sourceAddress: sourceAddress,
		inbound:       inbound,
		outbound:      outbound,
	}
	go c.generateMessages()
	return c
}

func (c *Class) Name() string {
	return ClassName
}

func (c *Class) Category() string {
	return "comm"
}

func (c *Class) OutboundMessages() chan *message.Message {
	return c.outbound
}

func (c *Class) generateMessages() {
	for dgram := range c.inbound {
		for _, sample := range dgram.Samples {
			for _, record := range sample.GetRecords() {
				switch record.(type) {
				case sflow.HostCPUCounters, sflow.HostMemoryCounters, sflow.HostDiskCounters,
					sflow.HostNetCounters:
					c.handleHostCounters(record)
				case sflow.GenericInterfaceCounters:
					c.handleSwitchCounters(record)
				default:
					// Unknown type. Drop.
				}
			}
		}
	}
}

func (c *Class) handleHostCounters(record sflow.Record) {
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

func (c *Class) handleSwitchCounters(record sflow.Record) {
	m := &message.Message{
		Class:     "switch-counters",
		Timestamp: clock.Time(),
		Content:   record,
	}
	switch record.(type) {
	case sflow.GenericInterfaceCounters:
		m.Type = "GenericInterface"
	default:
		return
	}
	c.outbound <- m
}
