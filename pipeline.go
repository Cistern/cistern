package main

import (
	"github.com/PreetamJinka/sflow-go"

	"fmt"
)

type Pipeline struct {
	processors []PipelineProcessor
}

func (p *Pipeline) Add(proc PipelineProcessor) {
	p.processors = append(p.processors, proc)
}

func (p *Pipeline) Run(inbound chan Message) {
	for _, proc := range p.processors {
		proc.SetInbound(inbound)
		inbound = proc.Outbound()
		go proc.Process()
	}

	go (&BlackholeProcessor{inbound: inbound}).Process()
}

type Message struct {
	Source string
	Record sflow.Record
}

type PipelineProcessor interface {
	Process()
	SetInbound(chan Message)
	Outbound() chan Message
}

type BlackholeProcessor struct {
	inbound chan Message
}

func (b *BlackholeProcessor) SetInbound(inbound chan Message) {
	b.inbound = inbound
}

func (b *BlackholeProcessor) Process() {
	for _ = range b.inbound {
	}
}

func (b *BlackholeProcessor) Outbound() chan Message {
	return nil
}

type HostProcessor struct {
	reg      *HostRegistry
	inbound  chan Message
	outbound chan Message
}

func NewHostProcessor(reg *HostRegistry) *HostProcessor {
	return &HostProcessor{
		reg:      reg,
		outbound: make(chan Message, 100),
	}
}

func (h *HostProcessor) SetInbound(inbound chan Message) {
	h.inbound = inbound
}

func (h *HostProcessor) Outbound() chan Message {
	return h.outbound
}

func (h *HostProcessor) Process() {
	for message := range h.inbound {
		record := message.Record
		registryKey := message.Source

		switch record.RecordType() {
		case sflow.TypeHostCpuCounter:
			c := record.(sflow.HostCpuCounters)

			h.reg.Insert(registryKey, "cpu.user", TypeDerivative, c.CpuUser)
			h.reg.Insert(registryKey, "cpu.nice", TypeDerivative, c.CpuNice)
			h.reg.Insert(registryKey, "cpu.sys", TypeDerivative, c.CpuSys)
			h.reg.Insert(registryKey, "cpu.idle", TypeDerivative, c.CpuIdle)
			h.reg.Insert(registryKey, "cpu.wio", TypeDerivative, c.CpuWio)
			h.reg.Insert(registryKey, "cpu.intr", TypeDerivative, c.CpuIntr)
			h.reg.Insert(registryKey, "cpu.softintr", TypeDerivative, c.CpuSoftIntr)

		case sflow.TypeHostMemoryCounter:
			m := record.(sflow.HostMemoryCounters)

			h.reg.Insert(registryKey, "mem.total", TypeGauge, m.Total)
			h.reg.Insert(registryKey, "mem.free", TypeGauge, m.Free)
			h.reg.Insert(registryKey, "mem.shared", TypeGauge, m.Shared)
			h.reg.Insert(registryKey, "mem.buffers", TypeGauge, m.Buffers)
			h.reg.Insert(registryKey, "mem.cached", TypeGauge, m.Cached)
			h.reg.Insert(registryKey, "mem.swap_total", TypeGauge, m.SwapTotal)
			h.reg.Insert(registryKey, "mem.swap_free", TypeGauge, m.SwapFree)

			h.reg.Insert(registryKey, "mem.page_in", TypeDerivative, m.PageIn)
			h.reg.Insert(registryKey, "mem.page_out", TypeDerivative, m.PageOut)
			h.reg.Insert(registryKey, "mem.swap_in", TypeDerivative, m.SwapIn)
			h.reg.Insert(registryKey, "mem.swap_out", TypeDerivative, m.SwapOut)

		case sflow.TypeHostDiskCounter:
			d := record.(sflow.HostDiskCounters)

			h.reg.Insert(registryKey, "disk.total", TypeGauge, d.Total)
			h.reg.Insert(registryKey, "disk.free", TypeGauge, d.Free)
			h.reg.Insert(registryKey, "disk.max_used", TypeGauge, d.MaxUsedPercent)

			h.reg.Insert(registryKey, "disk.reads", TypeDerivative, d.Reads)
			h.reg.Insert(registryKey, "disk.bytes_read", TypeDerivative, d.BytesRead)
			h.reg.Insert(registryKey, "disk.read_time", TypeDerivative, d.ReadTime)

			h.reg.Insert(registryKey, "disk.writes", TypeDerivative, d.Writes)
			h.reg.Insert(registryKey, "disk.bytes_written", TypeDerivative, d.BytesWritten)
			h.reg.Insert(registryKey, "disk.write_time", TypeDerivative, d.WriteTime)

		case sflow.TypeHostNetCounter:
			n := record.(sflow.HostNetCounters)

			h.reg.Insert(registryKey, "net.bytes_in", TypeDerivative, n.BytesIn)
			h.reg.Insert(registryKey, "net.packets_in", TypeDerivative, n.PacketsIn)
			h.reg.Insert(registryKey, "net.errs_in", TypeDerivative, n.ErrsIn)
			h.reg.Insert(registryKey, "net.drops_in", TypeDerivative, n.DropsIn)

			h.reg.Insert(registryKey, "net.bytes_out", TypeDerivative, n.BytesOut)
			h.reg.Insert(registryKey, "net.packets_out", TypeDerivative, n.PacketsOut)
			h.reg.Insert(registryKey, "net.errs_out", TypeDerivative, n.ErrsOut)
			h.reg.Insert(registryKey, "net.drops_out", TypeDerivative, n.DropsOut)

		default:
			h.outbound <- message
		}
	}
}

type GenericIfaceProcessor struct {
	reg      *HostRegistry
	inbound  chan Message
	outbound chan Message
}

func NewGenericIfaceProcessor(reg *HostRegistry) *GenericIfaceProcessor {
	return &GenericIfaceProcessor{
		reg:      reg,
		outbound: make(chan Message, 100),
	}
}

func (p *GenericIfaceProcessor) SetInbound(inbound chan Message) {
	p.inbound = inbound
}

func (p *GenericIfaceProcessor) Outbound() chan Message {
	return p.outbound
}

func (p *GenericIfaceProcessor) Process() {
	for message := range p.inbound {
		record := message.Record
		registryKey := message.Source

		switch record.RecordType() {
		case sflow.TypeGenericIfaceCounter:
			c := record.(sflow.GenericIfaceCounters)

			p.reg.Insert(registryKey, fmt.Sprintf("if%d.%s", c.Index, "octets_in"), TypeDerivative, c.InOctets)
			p.reg.Insert(registryKey, fmt.Sprintf("if%d.%s", c.Index, "unicast_packets_in"), TypeDerivative, c.InUcastPkts)
			p.reg.Insert(registryKey, fmt.Sprintf("if%d.%s", c.Index, "multicast_packets_in"), TypeDerivative, c.InMulticastPkts)
			p.reg.Insert(registryKey, fmt.Sprintf("if%d.%s", c.Index, "broadcast_packets_in"), TypeDerivative, c.InBroadcastPkts)
			p.reg.Insert(registryKey, fmt.Sprintf("if%d.%s", c.Index, "errors_in"), TypeDerivative, c.InErrors)
			p.reg.Insert(registryKey, fmt.Sprintf("if%d.%s", c.Index, "unknown_proto_in"), TypeDerivative, c.InUnknownProtos)

			p.reg.Insert(registryKey, fmt.Sprintf("if%d.%s", c.Index, "octets_out"), TypeDerivative, c.OutOctets)
			p.reg.Insert(registryKey, fmt.Sprintf("if%d.%s", c.Index, "unicast_packets_out"), TypeDerivative, c.OutUcastPkts)
			p.reg.Insert(registryKey, fmt.Sprintf("if%d.%s", c.Index, "multicast_packets_out"), TypeDerivative, c.OutMulticastPkts)
			p.reg.Insert(registryKey, fmt.Sprintf("if%d.%s", c.Index, "broadcast_packets_out"), TypeDerivative, c.OutBroadcastPkts)
			p.reg.Insert(registryKey, fmt.Sprintf("if%d.%s", c.Index, "errors_out"), TypeDerivative, c.OutErrors)

		default:
			select {
			case p.outbound <- message:
			default:
			}
		}
	}
}
