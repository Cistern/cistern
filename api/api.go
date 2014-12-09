package api

import (
	"net/http"

	"github.com/PreetamJinka/cistern/state/metrics"
	"github.com/PreetamJinka/cistern/state/series"
)

type ApiServer struct {
	addr         string
	hostRegistry *metrics.HostRegistry
	engine       *series.Engine
}

func NewApiServer(address string, reg *metrics.HostRegistry, engine *series.Engine) *ApiServer {
	return &ApiServer{
		addr:         address,
		hostRegistry: reg,
		engine:       engine,
	}
}

func (s *ApiServer) Run() {
	http.Handle("/hosts", hostStatus(s.hostRegistry))
	http.Handle("/metrics", hostMetrics(s.hostRegistry))
	http.Handle("/metricstates", metricStates(s.hostRegistry))
	http.Handle("/metricseries", metricSeries(s.engine))
	go http.ListenAndServe(s.addr, nil)
}
