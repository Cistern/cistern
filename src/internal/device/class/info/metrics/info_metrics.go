package metrics

import (
	"log"
	"net"

	"internal/message"
	metricsPackage "internal/state/metrics"
)

const ClassName = "metrics"

type Class struct {
	registry      *metricsPackage.MetricRegistry
	sourceAddress net.IP
	outbound      chan *message.Message
}

func NewClass(registry *metricsPackage.MetricRegistry,
	sourceAddress net.IP, outbound chan *message.Message) *Class {
	c := &Class{
		registry:      registry,
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
	metricsData := m.Content.(metricsPackage.MessageContent)
	c.registry.Lock()
	defer c.registry.Unlock()
	for name, v := range metricsData {
		updatedState := c.registry.Update(name, v.Type, v.Value)
		log.Printf("%s,device=%s value=%f %d", name, c.sourceAddress, updatedState, m.Timestamp*1e9)
	}
}
