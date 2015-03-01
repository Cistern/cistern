package device

import (
	"fmt"
	"log"
	"net"

	"github.com/PreetamJinka/proto"
	"github.com/PreetamJinka/sflow"

	"github.com/PreetamJinka/cistern/state/metrics"
)

func (d *Device) processHostCPUCounters(c sflow.HostCpuCounters) {
	d.updateAndEmit("cpu.user", metrics.TypeDerivative, c.CpuUser)
	d.updateAndEmit("cpu.nice", metrics.TypeDerivative, c.CpuNice)
	d.updateAndEmit("cpu.sys", metrics.TypeDerivative, c.CpuSys)
	d.updateAndEmit("cpu.idle", metrics.TypeDerivative, c.CpuIdle)
	d.updateAndEmit("cpu.wio", metrics.TypeDerivative, c.CpuWio)
	d.updateAndEmit("cpu.intr", metrics.TypeDerivative, c.CpuIntr)
	d.updateAndEmit("cpu.softintr", metrics.TypeDerivative, c.CpuSoftIntr)
}

func (d *Device) processHostMemoryCounters(c sflow.HostMemoryCounters) {
	d.updateAndEmit("mem.total", metrics.TypeGauge, c.Total)
	d.updateAndEmit("mem.free", metrics.TypeGauge, c.Free)
	d.updateAndEmit("mem.shared", metrics.TypeGauge, c.Shared)
	d.updateAndEmit("mem.buffers", metrics.TypeGauge, c.Buffers)
	d.updateAndEmit("mem.cached", metrics.TypeGauge, c.Cached)
	d.updateAndEmit("mem.swap_total", metrics.TypeGauge, c.SwapTotal)
	d.updateAndEmit("mem.swap_free", metrics.TypeGauge, c.SwapFree)

	d.updateAndEmit("mem.page_in", metrics.TypeDerivative, c.PageIn)
	d.updateAndEmit("mem.page_out", metrics.TypeDerivative, c.PageOut)
	d.updateAndEmit("mem.swap_in", metrics.TypeDerivative, c.SwapIn)
	d.updateAndEmit("mem.swap_out", metrics.TypeDerivative, c.SwapOut)
}

func (d *Device) processHostDiskCounters(c sflow.HostDiskCounters) {
	d.updateAndEmit("disk.total", metrics.TypeGauge, c.Total)
	d.updateAndEmit("disk.free", metrics.TypeGauge, c.Free)
	d.updateAndEmit("disk.max_used", metrics.TypeGauge, c.MaxUsedPercent)

	d.updateAndEmit("disk.reads", metrics.TypeDerivative, c.Reads)
	d.updateAndEmit("disk.bytes_read", metrics.TypeDerivative, c.BytesRead)
	d.updateAndEmit("disk.read_time", metrics.TypeDerivative, c.ReadTime)

	d.updateAndEmit("disk.writes", metrics.TypeDerivative, c.Writes)
	d.updateAndEmit("disk.bytes_written", metrics.TypeDerivative, c.BytesWritten)
	d.updateAndEmit("disk.write_time", metrics.TypeDerivative, c.WriteTime)
}

func (d *Device) processHostNetCounters(c sflow.HostNetCounters) {
	d.updateAndEmit("net.bytes_in", metrics.TypeDerivative, c.BytesIn)
	d.updateAndEmit("net.packets_in", metrics.TypeDerivative, c.PacketsIn)
	d.updateAndEmit("net.errs_in", metrics.TypeDerivative, c.ErrsIn)
	d.updateAndEmit("net.drops_in", metrics.TypeDerivative, c.DropsIn)

	d.updateAndEmit("net.bytes_out", metrics.TypeDerivative, c.BytesOut)
	d.updateAndEmit("net.packets_out", metrics.TypeDerivative, c.PacketsOut)
	d.updateAndEmit("net.errs_out", metrics.TypeDerivative, c.ErrsOut)
	d.updateAndEmit("net.drops_out", metrics.TypeDerivative, c.DropsOut)
}

func (d *Device) processGenericInterfaceCounters(c sflow.GenericInterfaceCounters) {
	d.updateAndEmit(fmt.Sprintf("if%d.%s", c.Index, "octets_in"), metrics.TypeDerivative, c.InOctets)
	d.updateAndEmit(fmt.Sprintf("if%d.%s", c.Index, "unicast_packets_in"), metrics.TypeDerivative, c.InUcastPkts)
	d.updateAndEmit(fmt.Sprintf("if%d.%s", c.Index, "multicast_packets_in"), metrics.TypeDerivative, c.InMulticastPkts)
	d.updateAndEmit(fmt.Sprintf("if%d.%s", c.Index, "broadcast_packets_in"), metrics.TypeDerivative, c.InBroadcastPkts)
	d.updateAndEmit(fmt.Sprintf("if%d.%s", c.Index, "errors_in"), metrics.TypeDerivative, c.InErrors)
	d.updateAndEmit(fmt.Sprintf("if%d.%s", c.Index, "unknown_proto_in"), metrics.TypeDerivative, c.InUnknownProtos)

	d.updateAndEmit(fmt.Sprintf("if%d.%s", c.Index, "octets_out"), metrics.TypeDerivative, c.OutOctets)
	d.updateAndEmit(fmt.Sprintf("if%d.%s", c.Index, "unicast_packets_out"), metrics.TypeDerivative, c.OutUcastPkts)
	d.updateAndEmit(fmt.Sprintf("if%d.%s", c.Index, "multicast_packets_out"), metrics.TypeDerivative, c.OutMulticastPkts)
	d.updateAndEmit(fmt.Sprintf("if%d.%s", c.Index, "broadcast_packets_out"), metrics.TypeDerivative, c.OutBroadcastPkts)
	d.updateAndEmit(fmt.Sprintf("if%d.%s", c.Index, "errors_out"), metrics.TypeDerivative, c.OutErrors)
}

func (d *Device) processRawPacketFlow(c sflow.RawPacketFlow) {
	sampleBytes := c.Header

	ethernetPacket, err := proto.DecodeEthernet(sampleBytes)
	if err != nil {
		log.Println("DecodeEthernet:", err)
		return
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
			log.Println("DecodeIPv6:", err)
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
			log.Println("DecodeTCP:", err)
			return
		}

		sourcePort = tcpPacket.SourcePort
		destPort = tcpPacket.DestinationPort

		protocolStr = "TCP"

	case 0x11:
		udpPacket, err := proto.DecodeUDP(ipPayload)
		if err != nil {
			log.Println("DecodeUDP:", err)
			return
		}

		sourcePort = udpPacket.SourcePort
		destPort = udpPacket.DestinationPort

		protocolStr = "UDP"
	}

	if sourcePort+destPort > 0 {
		d.topTalkers.Update(protocolStr, sourceAddr, destAddr, int(sourcePort), int(destPort), int(length))
		log.Printf("[Packet flow] [%s] %v:%d -> %v:%d (%d bytes)",
			protocolStr, sourceAddr, sourcePort, destAddr, destPort, length)
	}
}
