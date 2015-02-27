package pipeline

import (
	"github.com/PreetamJinka/sflow"

	"github.com/PreetamJinka/cistern/state/metrics"
)

type HostProcessor struct {
	reg      *metrics.HostRegistry
	inbound  chan Message
	outbound chan Message
}

func NewHostProcessor(reg *metrics.HostRegistry) *HostProcessor {
	return &HostProcessor{
		reg:      reg,
		outbound: make(chan Message, 4),
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

		switch record.(type) {
		case sflow.HostCpuCounters:
			c := record.(sflow.HostCpuCounters)

			h.reg.Insert(registryKey, "cpu.user", metrics.TypeDerivative, c.CpuUser)
			h.reg.Insert(registryKey, "cpu.nice", metrics.TypeDerivative, c.CpuNice)
			h.reg.Insert(registryKey, "cpu.sys", metrics.TypeDerivative, c.CpuSys)
			h.reg.Insert(registryKey, "cpu.idle", metrics.TypeDerivative, c.CpuIdle)
			h.reg.Insert(registryKey, "cpu.wio", metrics.TypeDerivative, c.CpuWio)
			h.reg.Insert(registryKey, "cpu.intr", metrics.TypeDerivative, c.CpuIntr)
			h.reg.Insert(registryKey, "cpu.softintr", metrics.TypeDerivative, c.CpuSoftIntr)

		case sflow.HostMemoryCounters:
			m := record.(sflow.HostMemoryCounters)

			h.reg.Insert(registryKey, "mem.total", metrics.TypeGauge, m.Total)
			h.reg.Insert(registryKey, "mem.free", metrics.TypeGauge, m.Free)
			h.reg.Insert(registryKey, "mem.shared", metrics.TypeGauge, m.Shared)
			h.reg.Insert(registryKey, "mem.buffers", metrics.TypeGauge, m.Buffers)
			h.reg.Insert(registryKey, "mem.cached", metrics.TypeGauge, m.Cached)
			h.reg.Insert(registryKey, "mem.swap_total", metrics.TypeGauge, m.SwapTotal)
			h.reg.Insert(registryKey, "mem.swap_free", metrics.TypeGauge, m.SwapFree)

			h.reg.Insert(registryKey, "mem.page_in", metrics.TypeDerivative, m.PageIn)
			h.reg.Insert(registryKey, "mem.page_out", metrics.TypeDerivative, m.PageOut)
			h.reg.Insert(registryKey, "mem.swap_in", metrics.TypeDerivative, m.SwapIn)
			h.reg.Insert(registryKey, "mem.swap_out", metrics.TypeDerivative, m.SwapOut)

		case sflow.HostDiskCounters:
			d := record.(sflow.HostDiskCounters)

			h.reg.Insert(registryKey, "disk.total", metrics.TypeGauge, d.Total)
			h.reg.Insert(registryKey, "disk.free", metrics.TypeGauge, d.Free)
			h.reg.Insert(registryKey, "disk.max_used", metrics.TypeGauge, d.MaxUsedPercent)

			h.reg.Insert(registryKey, "disk.reads", metrics.TypeDerivative, d.Reads)
			h.reg.Insert(registryKey, "disk.bytes_read", metrics.TypeDerivative, d.BytesRead)
			h.reg.Insert(registryKey, "disk.read_time", metrics.TypeDerivative, d.ReadTime)

			h.reg.Insert(registryKey, "disk.writes", metrics.TypeDerivative, d.Writes)
			h.reg.Insert(registryKey, "disk.bytes_written", metrics.TypeDerivative, d.BytesWritten)
			h.reg.Insert(registryKey, "disk.write_time", metrics.TypeDerivative, d.WriteTime)

		case sflow.HostNetCounters:
			n := record.(sflow.HostNetCounters)

			h.reg.Insert(registryKey, "net.bytes_in", metrics.TypeDerivative, n.BytesIn)
			h.reg.Insert(registryKey, "net.packets_in", metrics.TypeDerivative, n.PacketsIn)
			h.reg.Insert(registryKey, "net.errs_in", metrics.TypeDerivative, n.ErrsIn)
			h.reg.Insert(registryKey, "net.drops_in", metrics.TypeDerivative, n.DropsIn)

			h.reg.Insert(registryKey, "net.bytes_out", metrics.TypeDerivative, n.BytesOut)
			h.reg.Insert(registryKey, "net.packets_out", metrics.TypeDerivative, n.PacketsOut)
			h.reg.Insert(registryKey, "net.errs_out", metrics.TypeDerivative, n.ErrsOut)
			h.reg.Insert(registryKey, "net.drops_out", metrics.TypeDerivative, n.DropsOut)

		default:
			select {
			case h.outbound <- message:
			default:
			}
		}
	}
}
