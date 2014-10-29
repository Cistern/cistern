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

func (h *HostRegistry) Hosts() []string {
	h.lock.RLock()
	defer h.lock.RUnlock()

	var hosts []string

	for host := range h.hosts {
		hosts = append(hosts, host)
	}

	return hosts
}
