package main

import (
	"errors"
	"fmt"
	"time"
)

type MetricType byte

const (
	TypeGauge MetricType = iota
	TypeDerivative
)

var (
	ErrUnknownMetric = errors.New("unknown metric")
)

func getFloat32Value(i interface{}) float32 {
	switch n := i.(type) {
	case int:
		return float32(n)
	case int8:
		return float32(n)
	case int16:
		return float32(n)
	case int32:
		return float32(n)
	case int64:
		return float32(n)
	case uint:
		return float32(n)
	case uint8:
		return float32(n)
	case uint16:
		return float32(n)
	case uint32:
		return float32(n)
	case uint64:
		return float32(n)
	case float64:
		return float32(n)
	}

	return i.(float32)
}

func getUint64Value(i interface{}) uint64 {
	switch n := i.(type) {
	case int:
		return uint64(n)
	case int8:
		return uint64(n)
	case int16:
		return uint64(n)
	case int32:
		return uint64(n)
	case int64:
		return uint64(n)
	case uint:
		return uint64(n)
	case uint8:
		return uint64(n)
	case uint16:
		return uint64(n)
	case uint32:
		return uint64(n)
	}

	return i.(uint64)
}

type MetricState interface {
	Type() MetricType
	Update(interface{}) MetricState
	Value() interface{}
}

type DerivativeState struct {
	lastUpdated time.Time
	prev        uint64
	value       float32
}

type GaugeState struct {
	lastUpdated time.Time
	value       float32
}

type MetricRegistry struct {
	metrics map[string]MetricState
}

func (g GaugeState) Update(value interface{}) MetricState {
	g.value = getFloat32Value(value)
	g.lastUpdated = time.Now()
	return g
}

func (g GaugeState) Type() MetricType {
	return TypeGauge
}

func (g GaugeState) Value() interface{} {
	return g.value
}

func (d DerivativeState) Update(value interface{}) MetricState {
	now := time.Now()
	timeDelta := now.Sub(d.lastUpdated)

	currentValue := getUint64Value(value)
	if d.prev >= currentValue {
		return d
	}

	d.value = float32(float64(currentValue-d.prev) / timeDelta.Seconds())
	d.lastUpdated = now
	d.prev = currentValue

	return d
}

func (d DerivativeState) Type() MetricType {
	return TypeDerivative
}

func (d DerivativeState) Value() interface{} {
	return d.value
}

func NewMetricRegistry() *MetricRegistry {
	return &MetricRegistry{
		metrics: make(map[string]MetricState),
	}
}

func (m *MetricRegistry) Update(metric string, metricType MetricType, value interface{}) error {
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

	return nil
}

func (m *MetricRegistry) String() string {
	result := ""
	for metric, state := range m.metrics {
		result += fmt.Sprintf("  %s = %v\n", metric, state.Value())
	}

	return result
}
