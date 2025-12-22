package db

import (
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	dbQueriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_queries_total",
			Help: "Total number of database queries",
		},
		[]string{"operation", "table"},
	)

	dbQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: []float64{.0001, .0005, .001, .005, .01, .025, .05, .1, .25, .5, 1},
		},
		[]string{"operation", "table"},
	)

	dbErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_errors_total",
			Help: "Total number of database errors",
		},
		[]string{"operation", "error_type"},
	)

	// SQLite-specific metrics
	sqliteBusyErrorsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "sqlite_busy_errors_total",
			Help: "Total number of SQLite database busy/locked errors",
		},
	)
)

// recordQuery records database query metrics
func (db *DB) recordQuery(operation, table string, start time.Time) {
	duration := time.Since(start).Seconds()
	dbQueryDuration.WithLabelValues(operation, table).Observe(duration)
	dbQueriesTotal.WithLabelValues(operation, table).Inc()
}

// recordError records database error metrics
func (db *DB) recordError(operation string, err error) {
	if err == nil {
		return
	}

	errorType := "unknown"
	errStr := err.Error()

	// Categorize common SQLite errors
	if strings.Contains(errStr, "locked") || strings.Contains(errStr, "busy") {
		errorType = "lock_contention"
		// Track SQLite-specific busy errors
		sqliteBusyErrorsTotal.Inc()
	} else if strings.Contains(errStr, "constraint") {
		errorType = "constraint_violation"
	} else if strings.Contains(errStr, "no such table") {
		errorType = "schema_error"
	} else if strings.Contains(errStr, "syntax error") {
		errorType = "syntax_error"
	}

	dbErrorsTotal.WithLabelValues(operation, errorType).Inc()
}
