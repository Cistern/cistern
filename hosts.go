package main

import (
	"fmt"
)

type HostRegistry struct {
	hosts map[string]*MetricRegistry
}

func NewHostRegistry() *HostRegistry {
	return &HostRegistry{
		hosts: make(map[string]*MetricRegistry),
	}
}

func (h *HostRegistry) Insert(host string, metric string, metricType MetricType, value interface{}) error {
	metricRegistry, present := h.hosts[host]
	if !present {
		h.hosts[host] = NewMetricRegistry()
		metricRegistry = h.hosts[host]
	}

	return metricRegistry.Update(metric, metricType, value)
}

func (h *HostRegistry) String() string {
	result := ""
	for host, metricRegistry := range h.hosts {
		result += fmt.Sprintf("Host: %s\n%s\n", host, metricRegistry.String())
	}

	return result
}
