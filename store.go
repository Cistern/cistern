package main

import (
	"database/sql"
	"time"
)

func SnapshotMetrics(db *sql.DB, r *HostRegistry, interval time.Duration) {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS metricdata (
		    host   TEXT NOT NULL DEFAULT "",
		    metric TEXT NOT NULL DEFAULT "",
			ts     INT  UNSIGNED NOT NULL DEFAULT 0,
			value  FLOAT,
			PRIMARY KEY (host, metric, ts)
		)`)

	if err != nil {
		panic(err)
	}

	// clean up goroutine
	go func() {
		now := time.Now()

		db.Exec("DELETE FROM metricdata WHERE ts < ?", now.Add(-3*24*time.Hour))

		for now := range time.Tick(time.Hour * 24 * 3) {
			db.Exec("DELETE FROM metricdata WHERE ts < ?", now.Add(-3*24*time.Hour))
		}
	}()

	for now := range time.Tick(interval) {
		for host, hostRegistry := range r.hosts {
			tx, err := db.Begin()
			if err != nil {
				panic(err)
			}

			for metric, metricState := range hostRegistry.metrics {
				tx.Exec("INSERT INTO metricdata VALUES (?, ?, ?, ?)", host, metric, now.Unix(),
					metricState.Value())
			}

			tx.Commit()
		}
	}
}
