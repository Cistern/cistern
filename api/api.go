package api

import (
	"net/http"

	"github.com/PreetamJinka/cistern/state/metrics"
)

type ApiServer struct {
	addr         string
	hostRegistry *metrics.HostRegistry
}

func NewApiServer(address string, reg *metrics.HostRegistry) *ApiServer {
	return &ApiServer{
		addr:         address,
		hostRegistry: reg,
	}
}

func (s *ApiServer) Run() {
	http.Handle("/hosts", hostStatus(s.hostRegistry))
	http.Handle("/metrics", hostMetrics(s.hostRegistry))
	http.Handle("/metricstates", metricStates(s.hostRegistry))
	go http.ListenAndServe(s.addr, nil)
}
