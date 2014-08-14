package main

import (
	"net/http"
)

func registryPrinter(reg *HostRegistry, stor *MetricStorage) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(stor.String()))
	})
}

func RunHTTP(addr string, r *HostRegistry, m *MetricStorage) {
	http.ListenAndServe(addr, registryPrinter(r, m))
}
