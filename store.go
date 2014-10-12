package main

import (
	"github.com/PreetamJinka/metricstore"

	"time"
)

func SnapshotMetrics(r *HostRegistry, interval time.Duration, baseDir string) {
	s := metricstore.NewMetricStore(baseDir)

	for now := range time.Tick(interval) {
		for host, hostRegistry := range r.hosts {
			for metric, metricState := range hostRegistry.metrics {
				s.Insert(host, metric, now, float64(metricState.Value()))
			}
		}
	}
}
