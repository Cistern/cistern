package metrics

import (
	"math"
)

type MetricRegistry struct {
	metrics map[string]MetricState
}

func NewMetricRegistry() *MetricRegistry {
	return &MetricRegistry{
		metrics: make(map[string]MetricState),
	}
}

func (m *MetricRegistry) Update(metric string, metricType MetricType, value interface{}) float32 {
	state, present := m.metrics[metric]
	if !present {
		switch metricType {
		case TypeDerivative:
			state = DerivativeState{}
		case TypeGauge:
			state = GaugeState{}
		}
	}

	m.metrics[metric] = state.Update(value)

	return state.Value()
}

func (m *MetricRegistry) Get(metric string) float32 {
	state, present := m.metrics[metric]
	if !present {
		return float32(math.NaN())
	}

	return state.Value()
}

func (m *MetricRegistry) Metrics() []string {
	metrics := []string{}

	for metric := range m.metrics {
		metrics = append(metrics, metric)
	}

	return metrics
}
