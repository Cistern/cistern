package device

import (
	"fmt"

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
