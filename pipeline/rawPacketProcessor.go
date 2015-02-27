package pipeline

import (
	"log"
	"net"

	"github.com/PreetamJinka/proto"
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
		case sflow.RawPacketFlow:
			log.Println("received raw packet flow record")

			rawFlow := record.(sflow.RawPacketFlow)
			sampleBytes := rawFlow.Header

			ethernetPacket, err := proto.DecodeEthernet(sampleBytes)
			if err != nil {
				log.Println("DecodeEthernet:", err)
				continue
			}

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
				ipv4Packet, err := proto.DecodeIPv4(ethernetPacket.Payload)
				if err != nil {
					log.Println("DecodeIPv4:", err)
					continue
				}

				sourceAddr = ipv4Packet.Source
				destAddr = ipv4Packet.Destination
				ipPayload = ipv4Packet.Payload

				protocol = ipv4Packet.Protocol

				length = ipv4Packet.Length

			case 0x86dd:
				ipv6Packet, err := proto.DecodeIPv6(ethernetPacket.Payload)
				if err != nil {
					log.Println("DecodeIPv6:", err)
					continue
				}

				sourceAddr = ipv6Packet.Source
				destAddr = ipv6Packet.Destination
				ipPayload = ipv6Packet.Payload

				protocol = ipv6Packet.NextHeader

				length = ipv6Packet.Length
			}

			switch protocol {
			case 0x6:
				tcpPacket, err := proto.DecodeTCP(ipPayload)
				if err != nil {
					log.Println("DecodeTCP:", err)
					continue
				}

				sourcePort = tcpPacket.SourcePort
				destPort = tcpPacket.DestinationPort

				protocolStr = "TCP"

			case 0x11:
				udpPacket, err := proto.DecodeUDP(ipPayload)
				if err != nil {
					log.Println("DecodeUDP:", err)
					continue
				}

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
