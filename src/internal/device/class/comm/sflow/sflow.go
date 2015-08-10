package sflow

import (
	"net"

	"github.com/Preetam/sflow"
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
				case sflow.HostCPUCounters:
					c.outbound <- &message.Message{
						Class:   "host-counters",
						Type:    "CPU",
						Content: record,
					}
				case sflow.HostMemoryCounters:
					c.outbound <- &message.Message{
						Class:   "host-counters",
						Type:    "Memory",
						Content: record,
					}
				case sflow.HostDiskCounters:
					c.outbound <- &message.Message{
						Class:   "host-counters",
						Type:    "Disk",
						Content: record,
					}
				case sflow.HostNetCounters:
					c.outbound <- &message.Message{
						Class:   "host-counters",
						Type:    "Net",
						Content: record,
					}
				default:
				}
			}
		}
	}
}
