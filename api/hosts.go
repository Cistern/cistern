package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PreetamJinka/catena"
	"github.com/PreetamJinka/cistern/state/metrics"
	"github.com/PreetamJinka/cistern/state/series"
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

func metricSeries(engine *series.Engine) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Content-Type", "application/json")

		log.Printf("[http] serving %s", r.URL)

		enc := json.NewEncoder(w)

		hostsString := r.URL.Query().Get("hosts")
		metricsString := r.URL.Query().Get("metrics")
		startString := r.URL.Query().Get("start")
		endString := r.URL.Query().Get("end")

		hosts := strings.Split(hostsString, ",")
		metrics := strings.Split(metricsString, ",")

		start := int64(0)
		end := int64(0)

		now := time.Now().Unix()

		var err error

		if startString != "" {
			start, err = strconv.ParseInt(startString, 10, 64)
			if err != nil {
				panic(err)
			}
		} else {
			start = now - 3600
		}

		if start < 0 {
			start = start + now
		}

		if endString != "" {
			end, err = strconv.ParseInt(endString, 10, 64)
			if err != nil {
				panic(err)
			}
		} else {
			end = now
		}

		if end < 0 {
			end = end + now
		}

		queryDescs := []catena.QueryDesc{}

		for _, host := range hosts {
			for _, metric := range metrics {
				queryDescs = append(queryDescs, catena.QueryDesc{
					Source: host,
					Metric: metric,
					Start:  start,
					End:    end,
				})
			}
		}

		resp := engine.Query(queryDescs)

		enc.Encode(resp)
	})
}
