package main

import (
	"github.com/PreetamJinka/metricstore"
	"github.com/PreetamJinka/siesta"

	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
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

func ServeMetrics(store *metricstore.MetricStore) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("foo")

		var params siesta.Params

		host := params.String("host", "", "")
		metric := params.String("metric", "", "")
		start := params.Int64("start", 0, "")
		end := params.Int64("end", 9999999999999, "")

		err := params.Parse(r.Form)

		if err != nil {
			panic(err)
		}

		startTime := time.Unix(*start, 0)
		endTime := time.Unix(*end, 0)

		points := store.Retrieve(*host, *metric, startTime, endTime)

		enc := json.NewEncoder(w)
		enc.Encode(points)
	}
}

func RunHTTP(address string, registry *HostRegistry, store *metricstore.MetricStore) {
	service := siesta.NewService("/")
	service.Route("GET", "/", "", ServeHostsList(registry))
	service.Route("GET", "/metrics/<host>/<metric>", "", ServeMetrics(store))
	service.Route("GET", "/asdf", "", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hi"))
	})

	http.ListenAndServe(address, service)
}
