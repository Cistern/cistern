package main

import (
	"github.com/PreetamJinka/siesta"

	"database/sql"
	"encoding/json"
	"math"
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

func ServeMetrics(db *sql.DB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var params siesta.Params

		host := params.String("host", "", "")
		metric := params.String("metric", "", "")
		start := params.Int64("start", 0, "")
		end := params.Int64("end", math.MaxInt64, "")

		err := params.Parse(r.Form)

		if err != nil {
			panic(err)
		}

		type point struct {
			Ts    time.Time `json:"ts"`
			Value float32   `json:"value"`
		}

		points := []point{}

		rows, err := db.Query(`SELECT ts, value FROM metricdata WHERE host = ?
				AND metric = ? AND ts BETWEEN ? AND ?`, *host, *metric, *start, *end)

		if err != nil {
			panic(err)
		}

		defer rows.Close()

		for rows.Next() {
			p := point{}
			t := int64(0)
			rows.Scan(&t, &p.Value)

			p.Ts = time.Unix(t, 0)

			points = append(points, p)
		}

		enc := json.NewEncoder(w)
		enc.Encode(points)
	}
}

func ListHosts(db *sql.DB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT DISTINCT host FROM metricdata")
		if err != nil {
			panic(err)
		}
		defer rows.Close()

		hosts := []string{}
		for rows.Next() {
			var host string
			err = rows.Scan(&host)
			if err == nil {
				hosts = append(hosts, host)
			}
		}

		enc := json.NewEncoder(w)
		enc.Encode(hosts)
	}
}

func ListMetrics(db *sql.DB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		var params siesta.Params
		host := params.String("host", "", "")
		err := params.Parse(r.Form)

		if err != nil {
			panic(err)
		}

		rows, err := db.Query("SELECT DISTINCT metric FROM metricdata WHERE host = ?", *host)
		if err != nil {
			panic(err)
		}
		defer rows.Close()

		metrics := []string{}
		for rows.Next() {
			var metric string
			err = rows.Scan(&metric)
			if err == nil {
				metrics = append(metrics, metric)
			}
		}

		enc := json.NewEncoder(w)
		enc.Encode(metrics)
	}
}

func RunHTTP(address string, registry *HostRegistry, db *sql.DB) {
	service := siesta.NewService("/")
	service.AddPre(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
	})

	service.Route("GET", "/", "", ServeHostsList(registry))
	service.Route("GET", "/metrics/<host>/<metric>", "", ServeMetrics(db))
	service.Route("GET", "/metrics/listhosts", "", ListHosts(db))
	service.Route("GET", "/metrics/listmetrics", "", ListMetrics(db))

	http.ListenAndServe(address, service)
}
