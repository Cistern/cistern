package metrics

import (
	"errors"
	"sync"
)

var (
	ErrUnknownHost = errors.New("metrics: unknown host")
)

type HostRegistry struct {
	lock  sync.RWMutex
	hosts map[string]*MetricRegistry
}

func NewHostRegistry() *HostRegistry {
	return &HostRegistry{
		lock:  sync.RWMutex{},
		hosts: make(map[string]*MetricRegistry),
	}
}

func (h *HostRegistry) Insert(host string, metric string, metricType MetricType, value interface{}) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	metricRegistry, present := h.hosts[host]
	if !present {
		h.hosts[host] = NewMetricRegistry()
		metricRegistry = h.hosts[host]
	}

	return metricRegistry.Update(metric, metricType, value)
}

// Hosts returns a slice of all of the hosts listed
// in the registry
func (h *HostRegistry) Hosts() []string {
	h.lock.RLock()
	defer h.lock.RUnlock()

	hosts := []string{}

	for host := range h.hosts {
		hosts = append(hosts, host)
	}

	return hosts
}

// Metrics returns a slice of all of the metrics
// for a specific host
func (h *HostRegistry) Metrics(host string) []string {
	h.lock.RLock()
	defer h.lock.RUnlock()

	metrics := []string{}

	if metricReg := h.hosts[host]; metricReg != nil {
		for metric := range h.hosts[host].metrics {
			metrics = append(metrics, metric)
		}
	}

	return metrics
}

func (h *HostRegistry) MetricStates(host string, metrics ...string) []float32 {
	h.lock.RLock()
	defer h.lock.RUnlock()

	values := []float32{}

	if metricReg := h.hosts[host]; metricReg != nil {
		for _, metric := range metrics {
			values = append(values, metricReg.Get(metric))
		}
	}

	return values
}
