package host_counters

import (
	"fmt"
	"net"

	"github.com/Preetam/sflow"
	"internal/message"
)

const ClassName = "host-counters"

type Class struct {
	sourceAddress net.IP
	inbound       chan *message.Message
	outbound      chan *message.Message
}

func NewClass(sourceAddress net.IP, outbound chan *message.Message) *Class {
	c := &Class{
		sourceAddress: sourceAddress,
		inbound:       make(chan *message.Message),
		outbound:      outbound,
	}
	go c.handleMessages()
	return c
}

func (c *Class) Name() string {
	return ClassName
}

func (c *Class) Category() string {
	return "info"
}

func (c *Class) InboundMessages() chan *message.Message {
	return c.inbound
}

func (c *Class) OutboundMessages() chan *message.Message {
	return c.outbound
}

func (c *Class) handleMessages() {
	for m := range c.inbound {
		fmt.Println("host_counters")
		switch m.Type {
		case "CPU":
			cpuCounters := m.Content.(sflow.HostCPUCounters)
			c.handleCPUCounters(cpuCounters)
		}
	}
}

func (c *Class) handleCPUCounters(counters sflow.HostCPUCounters) {
	str := fmt.Sprintf("Load: %0.2f %0.2f %0.2f\n"+
		"Procs: %d %d\n"+
		"CPU: %d %d\n"+
		"Uptime: %d\n"+
		"CPU: %dus %dni %dsy %did %dio %din %dsi %di %dcw %dst %dgu %dgn\n",
		counters.Load1m,
		counters.Load5m,
		counters.Load15m,
		counters.ProcessesRunning,
		counters.ProcessesTotal,
		counters.NumCPU,
		counters.SpeedCPU,
		counters.Uptime,
		counters.CPUUser,
		counters.CPUNice,
		counters.CPUSys,
		counters.CPUIdle,
		counters.CPUWio,
		counters.CPUIntr,
		counters.CPUSoftIntr,
		counters.Interrupts,
		counters.ContextSwitches,
		counters.CPUSteal,
		counters.CPUGuest,
		counters.CPUGuestNice,
	)
	c.outbound <- &message.Message{
		Class:   "debug",
		Content: str,
	}
}
