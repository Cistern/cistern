package pipeline

import (
	"fmt"

	"github.com/PreetamJinka/cistern/net/sflow"
	"github.com/PreetamJinka/cistern/state/metrics"
)

type GenericInterfaceCountersProcessor struct {
	reg      *metrics.HostRegistry
	inbound  chan Message
	outbound chan Message
}

func NewGenericInterfaceCountersProcessor(reg *metrics.HostRegistry) *GenericInterfaceCountersProcessor {
	return &GenericInterfaceCountersProcessor{
		reg:      reg,
		outbound: make(chan Message, 4),
	}
}

func (p *GenericInterfaceCountersProcessor) SetInbound(inbound chan Message) {
	p.inbound = inbound
}

func (p *GenericInterfaceCountersProcessor) Outbound() chan Message {
	return p.outbound
}

func (p *GenericInterfaceCountersProcessor) Process() {
	for message := range p.inbound {

		record := message.Record
		registryKey := message.Source

		switch record.(type) {
		case sflow.GenericInterfaceCounters:
			c := record.(sflow.GenericInterfaceCounters)

			p.reg.Insert(registryKey, fmt.Sprintf("if%d.%s", c.Index, "octets_in"), metrics.TypeDerivative, c.InOctets)
			p.reg.Insert(registryKey, fmt.Sprintf("if%d.%s", c.Index, "unicast_packets_in"), metrics.TypeDerivative, c.InUcastPkts)
			p.reg.Insert(registryKey, fmt.Sprintf("if%d.%s", c.Index, "multicast_packets_in"), metrics.TypeDerivative, c.InMulticastPkts)
			p.reg.Insert(registryKey, fmt.Sprintf("if%d.%s", c.Index, "broadcast_packets_in"), metrics.TypeDerivative, c.InBroadcastPkts)
			p.reg.Insert(registryKey, fmt.Sprintf("if%d.%s", c.Index, "errors_in"), metrics.TypeDerivative, c.InErrors)
			p.reg.Insert(registryKey, fmt.Sprintf("if%d.%s", c.Index, "unknown_proto_in"), metrics.TypeDerivative, c.InUnknownProtos)

			p.reg.Insert(registryKey, fmt.Sprintf("if%d.%s", c.Index, "octets_out"), metrics.TypeDerivative, c.OutOctets)
			p.reg.Insert(registryKey, fmt.Sprintf("if%d.%s", c.Index, "unicast_packets_out"), metrics.TypeDerivative, c.OutUcastPkts)
			p.reg.Insert(registryKey, fmt.Sprintf("if%d.%s", c.Index, "multicast_packets_out"), metrics.TypeDerivative, c.OutMulticastPkts)
			p.reg.Insert(registryKey, fmt.Sprintf("if%d.%s", c.Index, "broadcast_packets_out"), metrics.TypeDerivative, c.OutBroadcastPkts)
			p.reg.Insert(registryKey, fmt.Sprintf("if%d.%s", c.Index, "errors_out"), metrics.TypeDerivative, c.OutErrors)

		default:
			select {
			case p.outbound <- message:
			default:
			}
		}
	}
}
