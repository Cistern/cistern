package metrics

import (
	"errors"
	"time"
)

type MetricType byte

const (
	TypeGauge MetricType = iota
	TypeDerivative
)

var (
	ErrUnknownMetric = errors.New("metrics: unknown metric")
)

// We want to go from some type of number to a float32.
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

// We want to go from some type of number to a uint64.
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
