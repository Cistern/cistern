package main

import (
	"github.com/PreetamJinka/sflow-go"
	"github.com/PreetamJinka/udpchan"

	"fmt"
	"log"
	"net"
	"time"
)

func main() {
	c, err := udpchan.Listen(":6343", nil)
	if err != nil {
		log.Fatalln(err)
	}

	registry := NewHostRegistry()

	go func() {
		for _ = range time.Tick(time.Second) {
			fmt.Println(registry)
		}
	}()

	for buf := range c {
		dgram := sflow.Decode(buf)
		ip := dgram.Header.IpAddress
		var records []sflow.Record

		for _, sample := range dgram.Samples {
			switch sample.SampleType() {
			case sflow.TypeCounterSample, sflow.TypeExpandedCounterSample:
				records = append(records, sample.GetRecords()...)
			}
		}

		aggregateRecords(registry, ip, records)
	}
}

func aggregateRecords(r *HostRegistry, ip net.IP, records []sflow.Record) {
	registryKey := ip.String()

	for _, record := range records {
		switch record.RecordType() {
		case sflow.TypeHostCpuCounter:
			c := record.(sflow.HostCpuCounters)

			r.Insert(registryKey, "cpu.user", TypeDerivative, c.CpuUser)
			r.Insert(registryKey, "cpu.nice", TypeDerivative, c.CpuNice)
			r.Insert(registryKey, "cpu.sys", TypeDerivative, c.CpuSys)
			r.Insert(registryKey, "cpu.idle", TypeDerivative, c.CpuIdle)
			r.Insert(registryKey, "cpu.wio", TypeDerivative, c.CpuWio)
			r.Insert(registryKey, "cpu.intr", TypeDerivative, c.CpuIntr)
			r.Insert(registryKey, "cpu.softintr", TypeDerivative, c.CpuSoftIntr)

		case sflow.TypeHostMemoryCounter:
			m := record.(sflow.HostMemoryCounters)

			r.Insert(registryKey, "mem.total", TypeGauge, m.Total)
			r.Insert(registryKey, "mem.free", TypeGauge, m.Free)
			r.Insert(registryKey, "mem.shared", TypeGauge, m.Shared)
			r.Insert(registryKey, "mem.buffers", TypeGauge, m.Buffers)
			r.Insert(registryKey, "mem.cached", TypeGauge, m.Cached)
			r.Insert(registryKey, "mem.swap_total", TypeGauge, m.SwapTotal)
			r.Insert(registryKey, "mem.swap_free", TypeGauge, m.SwapFree)

			r.Insert(registryKey, "mem.page_in", TypeDerivative, m.PageIn)
			r.Insert(registryKey, "mem.page_out", TypeDerivative, m.PageOut)
			r.Insert(registryKey, "mem.swap_in", TypeDerivative, m.SwapIn)
			r.Insert(registryKey, "mem.swap_out", TypeDerivative, m.SwapOut)

		case sflow.TypeHostDiskCounter:
			d := record.(sflow.HostDiskCounters)

			r.Insert(registryKey, "disk.total", TypeGauge, d.Total)
			r.Insert(registryKey, "disk.free", TypeGauge, d.Free)
			r.Insert(registryKey, "disk.max_used", TypeGauge, d.MaxUsedPercent)

			r.Insert(registryKey, "disk.reads", TypeDerivative, d.Reads)
			r.Insert(registryKey, "disk.bytes_read", TypeDerivative, d.BytesRead)
			r.Insert(registryKey, "disk.read_time", TypeDerivative, d.ReadTime)

			r.Insert(registryKey, "disk.writes", TypeDerivative, d.Writes)
			r.Insert(registryKey, "disk.bytes_written", TypeDerivative, d.BytesWritten)
			r.Insert(registryKey, "disk.write_time", TypeDerivative, d.WriteTime)

		case sflow.TypeHostNetCounter:
			n := record.(sflow.HostNetCounters)

			r.Insert(registryKey, "net.bytes_in", TypeDerivative, n.BytesIn)
			r.Insert(registryKey, "net.packets_in", TypeDerivative, n.PacketsIn)
			r.Insert(registryKey, "net.errs_in", TypeDerivative, n.ErrsIn)
			r.Insert(registryKey, "net.drops_in", TypeDerivative, n.DropsIn)

			r.Insert(registryKey, "net.bytes_out", TypeDerivative, n.BytesOut)
			r.Insert(registryKey, "net.packets_out", TypeDerivative, n.PacketsOut)
			r.Insert(registryKey, "net.errs_out", TypeDerivative, n.ErrsOut)
			r.Insert(registryKey, "net.drops_out", TypeDerivative, n.DropsOut)

		case sflow.TypeGenericIfaceCounter:
			c := record.(sflow.GenericIfaceCounters)

			r.Insert(registryKey, fmt.Sprintf("if%d.octets_in", c.Index), TypeDerivative, c.InOctets)
			r.Insert(registryKey, fmt.Sprintf("if%d.ucast_packets_in", c.Index), TypeDerivative, c.InUcastPkts)
			r.Insert(registryKey, fmt.Sprintf("if%d.ucast_packets_in", c.Index), TypeDerivative, c.InUcastPkts)
			r.Insert(registryKey, fmt.Sprintf("if%d.multicast_packets_in", c.Index), TypeDerivative, c.InMulticastPkts)
			r.Insert(registryKey, fmt.Sprintf("if%d.broadcast_packets_in", c.Index), TypeDerivative, c.InBroadcastPkts)
			r.Insert(registryKey, fmt.Sprintf("if%d.discards_in", c.Index), TypeDerivative, c.InDiscards)
			r.Insert(registryKey, fmt.Sprintf("if%d.errors_in", c.Index), TypeDerivative, c.InErrors)
			r.Insert(registryKey, fmt.Sprintf("if%d.unknown_in", c.Index), TypeDerivative, c.InUnknownProtos)
			r.Insert(registryKey, fmt.Sprintf("if%d.octets_out", c.Index), TypeDerivative, c.OutOctets)
			r.Insert(registryKey, fmt.Sprintf("if%d.multicast_packets_out", c.Index), TypeDerivative, c.OutMulticastPkts)
			r.Insert(registryKey, fmt.Sprintf("if%d.broadcast_packets_out", c.Index), TypeDerivative, c.OutBroadcastPkts)
			r.Insert(registryKey, fmt.Sprintf("if%d.discards_out", c.Index), TypeDerivative, c.OutDiscards)
			r.Insert(registryKey, fmt.Sprintf("if%d.errors_out", c.Index), TypeDerivative, c.OutErrors)
		}
	}
}
