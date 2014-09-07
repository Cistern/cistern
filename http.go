package main

import (
	"encoding/json"
	"net/http"
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

func ServeHostCpuStats(registry *HostRegistry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")

		hostsSlice := make([]hostStats, 0)

		hosts := registry.GetHosts()

	HOST_LOOP:
		for _, host := range hosts {

			// These are the metrics we're interested in.
			cpuMetrics := []string{
				"cpu.user",
				"cpu.sys",
				"cpu.nice",
				"cpu.wio",
				"cpu.intr",
				"cpu.softintr",
				"cpu.idle",
			}

			// Get values. Some, or all, of these could be NaN.
			metrics, err := registry.Query(host, cpuMetrics...)
			if err != nil {
				continue
			}

			var totalTime float32

			for _, metric := range metrics {
				// NaN != NaN according to the IEEE standard.
				if metric != metric {
					continue HOST_LOOP
				}
				totalTime += metric
			}

			// We want percentages.
			totalTime /= 100

			h := hostStats{
				Name:  host,
				Stats: make(map[string]stats),
			}

			h.Stats["cpu"] = cpuStats{
				metrics[0] / totalTime,
				metrics[1] / totalTime,
				metrics[2] / totalTime,
				metrics[3] / totalTime,
				metrics[4] / totalTime,
				metrics[5] / totalTime,
				metrics[6] / totalTime,
			}

			metrics, err = registry.Query(host, "mem.total", "mem.free",
				"mem.shared", "mem.buffers", "mem.cached")

			if err == nil {
				h.Stats["mem"] = memStats{
					metrics[0],
					metrics[1],
					metrics[2],
					metrics[3],
					metrics[4],
				}
			}

			metrics, err = registry.Query(host, "net.bytes_in", "net.packets_in",
				"net.bytes_out", "net.packets_out")

			if err == nil {
				h.Stats["net"] = netStats{
					metrics[0],
					metrics[1],
					metrics[2],
					metrics[3],
				}
			}

			hostsSlice = append(hostsSlice, h)
		}

		enc := json.NewEncoder(w)
		enc.Encode(hostsSlice)
	})
}
