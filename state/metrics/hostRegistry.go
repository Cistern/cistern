package metrics

import (
	"errors"
	"sync"
	"time"

	"github.com/PreetamJinka/catena"
	"github.com/PreetamJinka/cistern/state/series"
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

func (h *HostRegistry) RunSnapshotter(engine *series.Engine) {
	now := time.Now()

	<-time.After(now.Add(time.Minute).Truncate(time.Minute).Sub(now))

	for now := range time.Tick(time.Minute) {
		h.lock.RLock()

		rows := catena.Rows{}

		for host, metricReg := range h.hosts {
			for metric, metricState := range metricReg.metrics {
				metricVal := metricState.Value()
				if metricVal == metricVal {
					rows = append(rows, catena.Row{
						Source:    host,
						Metric:    metric,
						Timestamp: now.Unix(),
						Value:     float64(metricVal),
					})
				}
			}
		}

		h.lock.RUnlock()

		engine.InsertRows(rows)
	}
}
