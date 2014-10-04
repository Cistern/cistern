package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

type stats interface{}

type hostStats struct {
	Name  string           `json:"name"`
	Stats map[string]stats `json:"stats"`
}

type cpuStats struct {
	User     float32 `json:"user"`
	Sys      float32 `json:"sys"`
	Nice     float32 `json:"nice"`
	Wio      float32 `json:"wio"`
	Intr     float32 `json:"intr"`
	SoftIntr float32 `json:"softintr"`
	Idle     float32 `json:"idle"`
}

type memStats struct {
	Total   float32 `json:"total"`
	Free    float32 `json:"free"`
	Shared  float32 `json:"shared"`
	Buffers float32 `json:"buffers"`
	Cached  float32 `json:"cached"`
}

type netStats struct {
	BytesIn    float32 `json:"bytesIn"`
	PacketsIn  float32 `json:"packetsIn"`
	BytesOut   float32 `json:"bytesOut"`
	PacketsOut float32 `json:"packetsOut"`
}

func ServeAllHostStats(registry *HostRegistry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")

		hostsSlice := make([]hostStats, 0)

		hosts := registry.GetHosts()

		for _, host := range hosts {

			statsMap := make(map[string]stats)
			metricRegistry := registry.hosts[host]
			for metric, metricState := range metricRegistry.metrics {
				statsMap[metric] = metricState.Value()
			}

			hostsSlice = append(hostsSlice, hostStats{
				Name:  host,
				Stats: statsMap,
			})
		}

		enc := json.NewEncoder(w)
		enc.Encode(hostsSlice)
	})
}

func ServeHostStats(registry *HostRegistry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		host := strings.TrimLeft(r.URL.Path, "/")

		if host == "" {
			ServeAllHostStats(registry).ServeHTTP(w, r)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")

		statsMap := make(map[string]stats)
		metricRegistry := registry.hosts[host]
		for metric, metricState := range metricRegistry.metrics {
			statsMap[metric] = metricState.Value()
		}

		enc := json.NewEncoder(w)
		enc.Encode(hostStats{
			Name:  host,
			Stats: statsMap,
		})
	})
}

func ServeHostsList(registry *HostRegistry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")

		enc := json.NewEncoder(w)
		enc.Encode(registry.GetHosts())
	})
}
