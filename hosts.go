package main

import (
	"errors"
	"fmt"
	"sync"
)

var (
	ErrUnknownHost = errors.New("host registry: unknown host")
)

// This is basically a structure to hold
// states which are organized by a host string.
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

func (h *HostRegistry) GetHosts() []string {
	h.lock.RLock()
	defer h.lock.RUnlock()

	var hosts []string

	for host := range h.hosts {
		hosts = append(hosts, host)
	}

	return hosts
}

func (h *HostRegistry) Query(host string, metrics ...string) ([]float32, error) {
	h.lock.RLock()
	defer h.lock.RUnlock()

	metricRegistry, present := h.hosts[host]
	if !present {
		return nil, ErrUnknownHost
	}

	var result = make([]float32, 0, len(metrics))
	for _, metric := range metrics {
		result = append(result, metricRegistry.Get(metric))
	}

	return result, nil
}

func (h *HostRegistry) String() string {
	h.lock.RLock()
	defer h.lock.RUnlock()

	result := ""
	for host, metricRegistry := range h.hosts {
		result += fmt.Sprintf("Host: %s\n%s\n", host, metricRegistry.String())
	}

	return result
}
