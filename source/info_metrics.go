package source

import (
	"github.com/Cistern/cistern/message"
	metricsPackage "github.com/Cistern/cistern/state/metrics"
	"github.com/Cistern/cistern/state/series"
)

const InfoMetricsClassName = "metrics"

type InfoMetricsClass struct {
	registry      *metricsPackage.MetricRegistry
	sourceAddress string
	outbound      chan *message.Message
}

func NewInfoMetricsClass(registry *metricsPackage.MetricRegistry,
	sourceAddress string, outbound chan *message.Message) *InfoMetricsClass {
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

	observations := []series.Observation{}
	for name, v := range metricsData {
		updatedVal := c.registry.Update(name, v.Type, v.Value)
		observations = append(observations, series.Observation{
			Source:    c.sourceAddress,
			Metric:    name,
			Timestamp: m.Timestamp,
			Value:     float64(updatedVal),
		})
	}

	c.outbound <- &message.Message{
		Class:   series.SeriesEngineClassName,
		Global:  true,
		Content: observations,
	}
}
