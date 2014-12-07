package pipeline

import (
	"log"
	"net"

	"github.com/PreetamJinka/cistern/net/proto"
	"github.com/PreetamJinka/cistern/net/sflow"
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
		case sflow.RawPacketFlow:
			log.Println("received raw packet flow record")

			rawFlow := record.(sflow.RawPacketFlow)
			sampleBytes := rawFlow.Header

			ethernetPacket := proto.DecodeEthernet(sampleBytes)

			var (
				protocol    uint8
				protocolStr = ""

				sourceAddr net.IP
				destAddr   net.IP

				sourcePort uint16
				destPort   uint16

				length uint16
			)

			var ipPayload []byte

			switch ethernetPacket.EtherType {
			case 0x0800:
				ipv4Packet := proto.DecodeIPv4(ethernetPacket.Payload)

				sourceAddr = ipv4Packet.Source
				destAddr = ipv4Packet.Destination
				ipPayload = ipv4Packet.Payload

				protocol = ipv4Packet.Protocol

				length = ipv4Packet.Length

			case 0x86dd:
				ipv6Packet := proto.DecodeIPv6(ethernetPacket.Payload)

				sourceAddr = ipv6Packet.Source
				destAddr = ipv6Packet.Destination
				ipPayload = ipv6Packet.Payload

				protocol = ipv6Packet.NextHeader

				length = ipv6Packet.Length
			}

			switch protocol {
			case 0x6:
				tcpPacket := proto.DecodeTCP(ipPayload)

				sourcePort = tcpPacket.SourcePort
				destPort = tcpPacket.DestinationPort

				protocolStr = "TCP"

			case 0x11:
				udpPacket := proto.DecodeUDP(ipPayload)

				sourcePort = udpPacket.SourcePort
				destPort = udpPacket.DestinationPort

				protocolStr = "UDP"
			}

			if sourcePort+destPort > 0 {
				log.Printf("[%s] %v:%d -> %v:%d (%d bytes)", protocolStr, sourceAddr, sourcePort, destAddr, destPort, length)
			}

		default:
			select {
			case p.outbound <- message:
			default:
			}
		}
	}
}
