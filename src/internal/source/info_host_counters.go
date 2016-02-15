package source

import (
	"github.com/Cistern/sflow"
	"internal/message"
	"internal/state/metrics"
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
