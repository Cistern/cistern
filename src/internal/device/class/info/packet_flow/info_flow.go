package packet_flow

import (
	"log"
	"net"

	"github.com/Preetam/proto"
	"github.com/Preetam/sflow"

	"internal/message"
)

const ClassName = "packet-flow"

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
	record := m.Content.(sflow.RawPacketFlow)
	ethernetPacket, err := proto.DecodeEthernet(record.Header)
	if err != nil {
		return
	}

	var (
		protocol    uint8
		protocolStr = ""
		sourceAddr  net.IP
		destAddr    net.IP
		sourcePort  uint16
		destPort    uint16
		ipPayload   []byte
		length      uint16
	)

	switch ethernetPacket.EtherType {
	case 0x0800:
		ipv4Packet, err := proto.DecodeIPv4(ethernetPacket.Payload)
		if err != nil {
			return
		}
		sourceAddr = ipv4Packet.Source
		destAddr = ipv4Packet.Destination
		ipPayload = ipv4Packet.Payload
		protocol = ipv4Packet.Protocol
		length = ipv4Packet.Length
	case 0x86dd:
		ipv6Packet, err := proto.DecodeIPv6(ethernetPacket.Payload)
		if err != nil {
			return
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
			return
		}
		sourcePort = tcpPacket.SourcePort
		destPort = tcpPacket.DestinationPort
		protocolStr = "TCP"
	case 0x11:
		udpPacket, err := proto.DecodeUDP(ipPayload)
		if err != nil {
			return
		}
		sourcePort = udpPacket.SourcePort
		destPort = udpPacket.DestinationPort
		protocolStr = "UDP"
	}

	log.Println(protocolStr, sourceAddr, sourcePort,
		"=>", destAddr, destPort, length, "bytes")
}
