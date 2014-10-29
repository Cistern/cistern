package main

import (
	"log"
	"time"

	"github.com/PreetamJinka/cistern/state/metrics"
)

func LogDiagnostics(hostRegistry *metrics.HostRegistry) {
	log.Println("logging diagnostics")

	for _ = range time.Tick(30 * time.Second) {
		hosts := hostRegistry.Hosts()

		log.Printf(`[DIAGNOSTIC] Num hosts: %d
  %v`, len(hosts), hosts)

	}
}
