package web

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// HTTP Metrics
var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	httpRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Current number of HTTP requests being served",
		},
	)
)

// Business Metrics - Shift Operations
var (
	shiftsCreatedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "shifts_created_total",
			Help: "Total number of shifts created",
		},
		[]string{"plan_token"},
	)

	shiftsDeletedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "shifts_deleted_total",
			Help: "Total number of shifts deleted",
		},
		[]string{"plan_token"},
	)
)

// Business Metrics - Participant Operations
var (
	participantJoinsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "participant_joins_total",
			Help: "Total number of participant join attempts",
		},
		[]string{"plan_token", "result"},
	)

	participantJoinsFailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "participant_joins_failed_total",
			Help: "Total number of failed participant joins by reason",
		},
		[]string{"plan_token", "reason"},
	)

	participantRemovalsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "participant_removals_total",
			Help: "Total number of participant removals",
		},
		[]string{"plan_token"},
	)
)

// Business Metrics - Plan Operations
var (
	shiftPlansCreatedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "shift_plans_created_total",
			Help: "Total number of shift plans created",
		},
	)

	shiftPlansDeletedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "shift_plans_deleted_total",
			Help: "Total number of shift plans deleted",
		},
	)
)

// Performance Metrics - Export Operations
var (
	exportRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "export_requests_total",
			Help: "Total number of export requests",
		},
		[]string{"format", "result"},
	)

	exportDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "export_duration_seconds",
			Help:    "Export generation duration in seconds",
			Buckets: []float64{.1, .5, 1, 2.5, 5, 10, 30, 60, 90, 120},
		},
		[]string{"format"},
	)

	exportErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "export_errors_total",
			Help: "Total number of export errors",
		},
		[]string{"format", "error_type"},
	)
)

// Performance Metrics - Authentication
var (
	adminAuthAttemptsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "admin_auth_attempts_total",
			Help: "Total number of admin authentication attempts",
		},
		[]string{"result"},
	)

	adminAuthDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "admin_auth_duration_seconds",
			Help:    "Admin authentication check duration in seconds",
			Buckets: []float64{.0001, .0005, .001, .005, .01, .025, .05, .1},
		},
	)
)

// Realtime (SSE) metrics
var (
	realtimeConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "shifts_realtime_connections",
			Help: "Current number of open realtime (SSE) connections",
		},
	)

	realtimeEventsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "shifts_realtime_events_total",
			Help: "Total number of realtime events sent to clients",
		},
	)
)

// responseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// newResponseWriter creates a new responseWriter
func newResponseWriter(w http.ResponseWriter) *responseWriter {
	// Default status code is 200 if WriteHeader is never called
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

// WriteHeader captures the status code and calls the underlying WriteHeader
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Unwrap lets http.ResponseController reach the underlying writer (deadlines, flushing)
func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

// Flush forwards to the underlying writer if it supports flushing
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// metricsMiddleware instruments HTTP requests with Prometheus metrics
func (m *middlewareManager) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip metrics endpoint itself to avoid recursion
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		httpRequestsInFlight.Inc()
		defer httpRequestsInFlight.Dec()

		// Wrap response writer to capture status code
		wrapped := newResponseWriter(w)

		// Process request
		next.ServeHTTP(wrapped, r)

		// Record metrics
		duration := time.Since(start).Seconds()
		path := r.URL.Path
		method := r.Method
		status := strconv.Itoa(wrapped.statusCode)

		httpRequestDuration.WithLabelValues(method, path).Observe(duration)
		httpRequestsTotal.WithLabelValues(method, path, status).Inc()
	})
}
