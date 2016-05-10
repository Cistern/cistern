package metrics

import (
	"time"
)

type MetricState interface {
	Type() MetricType
	Update(interface{}) MetricState
	Value() float32
}

type DerivativeState struct {
	lastUpdated time.Time
	// This is a uint64 because we want calculate
	// derivatives accurately.
	prev  uint64
	value float32
}

type GaugeState struct {
	lastUpdated time.Time
	value       float32
}

func (g GaugeState) Update(value interface{}) MetricState {
	g.value = getFloat32Value(value)
	g.lastUpdated = time.Now()
	return g
}

func (g GaugeState) Type() MetricType {
	return TypeGauge
}

func (g GaugeState) Value() float32 {
	return g.value
}

func (d DerivativeState) Update(value interface{}) MetricState {
	now := time.Now()
	timeDelta := now.Sub(d.lastUpdated)
	currentValue := getUint64Value(value)
	if d.prev > currentValue {
		// Rollover? Keep the value we have.
		d.lastUpdated = now
		d.prev = currentValue
		d.value = 0
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

func (d DerivativeState) Value() float32 {
	return d.value
}
