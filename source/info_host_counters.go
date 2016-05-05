package source

import (
	"github.com/Cistern/cistern/message"
	"github.com/Cistern/cistern/state/metrics"
	"github.com/Cistern/sflow"
)

const InfoHostCountersClassName = "host-counters"

type InfoHostCountersClass struct {
	sourceAddress string
	outbound      chan *message.Message
}

func NewInfoHostCountersClass(sourceAddress string, outbound chan *message.Message) *InfoHostCountersClass {
	c := &InfoHostCountersClass{
		sourceAddress: sourceAddress,
		outbound:      outbound,
	}
	return c
}

func (c *InfoHostCountersClass) Name() string {
	return InfoHostCountersClassName
}

func (c *InfoHostCountersClass) Category() string {
	return "info"
}

func (c *InfoHostCountersClass) OutboundMessages() chan *message.Message {
	return c.outbound
}

func (c *InfoHostCountersClass) Process(m *message.Message) {
	switch m.Type {
	case "CPU":
		cpuCounters := m.Content.(sflow.HostCPUCounters)
		c.handleCPUCounters(cpuCounters)
	case "Memory":
		memCounters := m.Content.(sflow.HostMemoryCounters)
		c.handleMemoryCounters(memCounters)
	case "Disk":
		diskCounters := m.Content.(sflow.HostDiskCounters)
		c.handleDiskCounters(diskCounters)
	case "Net":
		netCounters := m.Content.(sflow.HostNetCounters)
		c.handleNetCounters(netCounters)
	default:
		// Drop.
	}
}

func (c *InfoHostCountersClass) handleCPUCounters(counters sflow.HostCPUCounters) {
	c.outbound <- &message.Message{
		Class: "metrics",
		Content: metrics.MessageContent{
			"load.1m": {
				Type:  metrics.TypeGauge,
				Value: counters.Load1m,
			},
			"load.5m": {
				Type:  metrics.TypeGauge,
				Value: counters.Load5m,
			},
			"load.15m": {
				Type:  metrics.TypeGauge,
				Value: counters.Load15m,
			},
			"processes.running": {
				Type:  metrics.TypeGauge,
				Value: counters.ProcessesRunning,
			},
			"processes.total": {
				Type:  metrics.TypeGauge,
				Value: counters.ProcessesTotal,
			},
			"uptime": {
				Type:  metrics.TypeGauge,
				Value: counters.Uptime,
			},
			"cpu.user": {
				Type:  metrics.TypeDerivative,
				Value: counters.CPUUser,
			},
			"cpu.nice": {
				Type:  metrics.TypeDerivative,
				Value: counters.CPUNice,
			},
			"cpu.sys": {
				Type:  metrics.TypeDerivative,
				Value: counters.CPUSys,
			},
			"cpu.idle": {
				Type:  metrics.TypeDerivative,
				Value: counters.CPUIdle,
			},
			"cpu.wio": {
				Type:  metrics.TypeDerivative,
				Value: counters.CPUWio,
			},
			"cpu.intr": {
				Type:  metrics.TypeDerivative,
				Value: counters.CPUIntr,
			},
			"cpu.softintr": {
				Type:  metrics.TypeDerivative,
				Value: counters.CPUSoftIntr,
			},
			"cpu.interrupts": {
				Type:  metrics.TypeDerivative,
				Value: counters.Interrupts,
			},
			"cpu.contextswitches": {
				Type:  metrics.TypeDerivative,
				Value: counters.ContextSwitches,
			},
			"cpu.steal": {
				Type:  metrics.TypeDerivative,
				Value: counters.CPUSteal,
			},
			"cpu.guest": {
				Type:  metrics.TypeDerivative,
				Value: counters.CPUGuest,
			},
			"cpu.guestnice": {
				Type:  metrics.TypeDerivative,
				Value: counters.CPUGuestNice,
			},
		},
	}
}

func (c *InfoHostCountersClass) handleMemoryCounters(counters sflow.HostMemoryCounters) {
	c.outbound <- &message.Message{
		Class: "metrics",
		Content: metrics.MessageContent{
			"mem.total": {
				Type:  metrics.TypeGauge,
				Value: counters.Total,
			},
			"mem.free": {
				Type:  metrics.TypeGauge,
				Value: counters.Free,
			},
			"mem.shared": {
				Type:  metrics.TypeGauge,
				Value: counters.Shared,
			},
			"mem.buffers": {
				Type:  metrics.TypeGauge,
				Value: counters.Buffers,
			},
			"mem.cached": {
				Type:  metrics.TypeGauge,
				Value: counters.Cached,
			},
			"mem.used": {
				Type:  metrics.TypeGauge,
				Value: counters.Total - (counters.Free + counters.Shared + counters.Buffers + counters.Cached),
			},
			"mem.swap_total": {
				Type:  metrics.TypeGauge,
				Value: counters.SwapTotal,
			},
			"mem.swap_free": {
				Type:  metrics.TypeGauge,
				Value: counters.SwapFree,
			},
			"mem.swap_used": {
				Type:  metrics.TypeGauge,
				Value: counters.SwapTotal - counters.SwapFree,
			},
			"mem.page_in": {
				Type:  metrics.TypeDerivative,
				Value: counters.PageIn,
			},
			"mem.page_out": {
				Type:  metrics.TypeDerivative,
				Value: counters.PageOut,
			},
			"mem.swap_in": {
				Type:  metrics.TypeDerivative,
				Value: counters.SwapIn,
			},
			"mem.swap_out": {
				Type:  metrics.TypeDerivative,
				Value: counters.SwapOut,
			},
		},
	}
}

func (c *InfoHostCountersClass) handleDiskCounters(counters sflow.HostDiskCounters) {
	c.outbound <- &message.Message{
		Class: "metrics",
		Content: metrics.MessageContent{
			"disk.total": {
				Type:  metrics.TypeGauge,
				Value: counters.Total,
			},
			"disk.free": {
				Type:  metrics.TypeGauge,
				Value: counters.Free,
			},
			"disk.used": {
				Type:  metrics.TypeGauge,
				Value: counters.Total - counters.Free,
			},
			"disk.max_used_percent": {
				Type:  metrics.TypeGauge,
				Value: counters.MaxUsedPercent,
			},
			"disk.reads": {
				Type:  metrics.TypeDerivative,
				Value: counters.Reads,
			},
			"disk.bytes_read": {
				Type:  metrics.TypeDerivative,
				Value: counters.BytesRead,
			},
			"disk.read_time": {
				Type:  metrics.TypeDerivative,
				Value: counters.ReadTime,
			},
			"disk.writes": {
				Type:  metrics.TypeDerivative,
				Value: counters.Writes,
			},
			"disk.bytes_written": {
				Type:  metrics.TypeDerivative,
				Value: counters.BytesWritten,
			},
			"disk.write_time": {
				Type:  metrics.TypeDerivative,
				Value: counters.WriteTime,
			},
		},
	}
}

func (c *InfoHostCountersClass) handleNetCounters(counters sflow.HostNetCounters) {
	c.outbound <- &message.Message{
		Class: "metrics",
		Content: metrics.MessageContent{
			"net.bytes_in": {
				Type:  metrics.TypeDerivative,
				Value: counters.BytesIn,
			},
			"net.packets_in": {
				Type:  metrics.TypeDerivative,
				Value: counters.PacketsIn,
			},
			"net.errors_in": {
				Type:  metrics.TypeDerivative,
				Value: counters.ErrorsIn,
			},
			"net.drops_in": {
				Type:  metrics.TypeDerivative,
				Value: counters.DropsIn,
			},
			"net.bytes_out": {
				Type:  metrics.TypeDerivative,
				Value: counters.BytesOut,
			},
			"net.packets_out": {
				Type:  metrics.TypeDerivative,
				Value: counters.PacketsOut,
			},
			"net.errors_out": {
				Type:  metrics.TypeDerivative,
				Value: counters.ErrorsOut,
			},
			"net.drops_out": {
				Type:  metrics.TypeDerivative,
				Value: counters.DropsOut,
			},
		},
	}
}
