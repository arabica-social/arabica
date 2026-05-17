package backup

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	lastSuccessTimestamp = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "arabica_backup_last_success_timestamp_seconds",
		Help: "Unix timestamp of the last successful backup run, per source.",
	}, []string{"source"})

	lastFailureTimestamp = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "arabica_backup_last_failure_timestamp_seconds",
		Help: "Unix timestamp of the last failed backup run, per source.",
	}, []string{"source"})

	lastDurationSeconds = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "arabica_backup_last_duration_seconds",
		Help: "Wall-clock duration of the most recent backup run, per source.",
	}, []string{"source"})

	lastSizeBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "arabica_backup_last_size_bytes",
		Help: "Size in bytes of the most recent successful backup file, per source.",
	}, []string{"source"})

	retainedCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "arabica_backup_retained_count",
		Help: "Number of backup files currently retained on the destination, per source.",
	}, []string{"source"})

	runsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "arabica_backup_runs_total",
		Help: "Total number of backup runs, partitioned by source and status.",
	}, []string{"source", "status"})
)
