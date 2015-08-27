package switch_counters

import (
	"fmt"
	"net"

	"github.com/Preetam/sflow"

	"internal/message"
	"internal/state/metrics"
)

const ClassName = "switch-counters"

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
	case "GenericInterface":
		counters := m.Content.(sflow.GenericInterfaceCounters)
		c.handleGenericInterfaceCounters(counters)
	default:
		// Drop.
	}
}

func (c *Class) handleGenericInterfaceCounters(counters sflow.GenericInterfaceCounters) {
	prefix := fmt.Sprintf("if%d.", counters.Index)
	c.outbound <- &message.Message{
		Class: "metrics",
		Content: metrics.MessageContent{
			prefix + "octets_in": {
				Type:  metrics.TypeDerivative,
				Value: counters.InOctets,
			},
			prefix + "unicast_packets_in": {
				Type:  metrics.TypeDerivative,
				Value: counters.InUnicastPackets,
			},
			prefix + "multicast_packets_in": {
				Type:  metrics.TypeDerivative,
				Value: counters.InMulticastPackets,
			},
			prefix + "broadcast_packets_in": {
				Type:  metrics.TypeDerivative,
				Value: counters.InBroadcastPackets,
			},
			prefix + "discards_in": {
				Type:  metrics.TypeDerivative,
				Value: counters.InDiscards,
			},
			prefix + "errors_in": {
				Type:  metrics.TypeDerivative,
				Value: counters.InErrors,
			},
			prefix + "unknown_protocols_in": {
				Type:  metrics.TypeDerivative,
				Value: counters.InUnknownProtocols,
			},
			prefix + "octets_out": {
				Type:  metrics.TypeDerivative,
				Value: counters.OutOctets,
			},
			prefix + "unicast_packets_out": {
				Type:  metrics.TypeDerivative,
				Value: counters.OutUnicastPackets,
			},
			prefix + "multicast_packets_out": {
				Type:  metrics.TypeDerivative,
				Value: counters.OutMulticastPackets,
			},
			prefix + "broadcast_packets_out": {
				Type:  metrics.TypeDerivative,
				Value: counters.OutBroadcastPackets,
			},
			prefix + "discards_out": {
				Type:  metrics.TypeDerivative,
				Value: counters.OutDiscards,
			},
			prefix + "errors_out": {
				Type:  metrics.TypeDerivative,
				Value: counters.OutErrors,
			},
			prefix + "promiscuous_mode": {
				Type:  metrics.TypeGauge,
				Value: counters.PromiscuousMode,
			},
		},
	}
}
