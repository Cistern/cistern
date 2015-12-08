package source

import (
	"net"

	"internal/message"
	metricsPackage "internal/state/metrics"
)

const InfoMetricsClassName = "metrics"

type InfoMetricsClass struct {
	registry      *metricsPackage.MetricRegistry
	sourceAddress net.IP
	outbound      chan *message.Message
}

func NewInfoMetricsClass(registry *metricsPackage.MetricRegistry,
	sourceAddress net.IP, outbound chan *message.Message) *InfoMetricsClass {
	c := &InfoMetricsClass{
		registry:      registry,
		sourceAddress: sourceAddress,
		outbound:      outbound,
	}
	return c
}

func (c *InfoMetricsClass) Name() string {
	return InfoMetricsClassName
}

func (c *InfoMetricsClass) Category() string {
	return "info"
}

func (c *InfoMetricsClass) OutboundMessages() chan *message.Message {
	return c.outbound
}

func (c *InfoMetricsClass) Process(m *message.Message) {
	metricsData := m.Content.(metricsPackage.MessageContent)
	c.registry.Lock()
	defer c.registry.Unlock()
	for name, v := range metricsData {
		c.registry.Update(name, v.Type, v.Value)
		//log.Printf("%s,source=%s value=%f %d", name, c.sourceAddress, updatedState, m.Timestamp*1e9)
	}
}
