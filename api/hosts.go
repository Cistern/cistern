package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/PreetamJinka/cistern/state/metrics"
)

func hostStatus(reg *metrics.HostRegistry) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[http] serving %s", r.URL)

		enc := json.NewEncoder(w)

		enc.Encode(reg.Hosts())
	})
}

func hostMetrics(reg *metrics.HostRegistry) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[http] serving %s", r.URL)

		enc := json.NewEncoder(w)

		host := r.URL.Query().Get("host")
		enc.Encode(reg.Metrics(host))
	})
}

func metricStates(reg *metrics.HostRegistry) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[http] serving %s", r.URL)

		enc := json.NewEncoder(w)

		host := r.URL.Query().Get("host")
		metrics := strings.Split(r.URL.Query().Get("metrics"), ",")

		enc.Encode(reg.MetricStates(host, metrics...))
	})
}
