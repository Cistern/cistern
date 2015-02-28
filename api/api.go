package api

import (
	"encoding/json"
	"net/http"

	"github.com/VividCortex/siesta"

	"github.com/PreetamJinka/cistern/device"
	"github.com/PreetamJinka/cistern/state/series"
)

type APIServer struct {
	addr           string
	deviceRegistry *device.Registry
	seriesEngine   *series.Engine
}

func NewAPIServer(address string, deviceRegistry *device.Registry, seriesEngine *series.Engine) *APIServer {
	return &APIServer{
		addr:           address,
		deviceRegistry: deviceRegistry,
		seriesEngine:   seriesEngine,
	}
}

func (s *APIServer) Run() {
	service := siesta.NewService("/")

	service.AddPost(func(c siesta.Context, w http.ResponseWriter, r *http.Request) {
		resp := c.Get(responseKey)
		err, _ := c.Get(errorKey).(string)

		enc := json.NewEncoder(w)
		enc.Encode(APIResponse{
			Data:  resp,
			Error: err,
		})
	})

	service.Route("GET", "/", "Default page", func(c siesta.Context, w http.ResponseWriter, r *http.Request) {
		c.Set(responseKey, "Welcome to the Cistern API!")
	})

	service.Route("GET", "/devices", "Lists sources", func(c siesta.Context, w http.ResponseWriter, r *http.Request) {
		type ipHostname struct {
			IP       string `json:"ip"`
			Hostname string `json:"hostname,omitempty"`
		}

		devices := []ipHostname{}

		for _, dev := range s.deviceRegistry.Devices() {
			devices = append(devices, ipHostname{
				IP:       dev.IP().String(),
				Hostname: dev.Hostname(),
			})
		}

		c.Set(responseKey, devices)
	})

	http.Handle("/", service)

	go http.ListenAndServe(s.addr, nil)
}
