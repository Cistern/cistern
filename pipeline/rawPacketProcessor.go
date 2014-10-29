package pipeline

import (
	"log"

	"github.com/PreetamJinka/sflow"

	"github.com/PreetamJinka/cistern/state/metrics"
)

type RawPacketProcessor struct {
	reg      *metrics.HostRegistry
	inbound  chan Message
	outbound chan Message
}

func NewRawPacketProcessor(reg *metrics.HostRegistry) *RawPacketProcessor {
	return &RawPacketProcessor{
		reg:      reg,
		outbound: make(chan Message, 16),
	}
}

func (p *RawPacketProcessor) SetInbound(inbound chan Message) {
	p.inbound = inbound
}

func (p *RawPacketProcessor) Outbound() chan Message {
	return p.outbound
}

func (p *RawPacketProcessor) Process() {
	for message := range p.inbound {
		record := message.Record

		switch record.(type) {
		case sflow.GenericIfaceCounters:
			log.Println("received raw packet flow record")

		default:
			select {
			case p.outbound <- message:
			default:
			}
		}
	}
}
