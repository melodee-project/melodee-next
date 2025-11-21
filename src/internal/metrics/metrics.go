package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all application metrics
type Metrics struct {
	// Stream metrics
	StreamRequestsTotal *prometheus.CounterVec
	StreamBytesTotal    prometheus.Counter

	// Job metrics
	JobDurationSeconds *prometheus.HistogramVec

	// Metadata metrics
	MetadataDriftTotal prometheus.Counter

	// Quarantine metrics
	QuarantineTotal *prometheus.CounterVec

	// Health metrics
	HealthStatus *prometheus.GaugeVec
}

// NewMetrics creates a new Metrics instance with all required metrics registered
func NewMetrics() *Metrics {
	return &Metrics{
		// Stream metrics
		StreamRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "melodee_stream_requests_total",
				Help: "Total number of stream requests",
			},
			[]string{"status", "format"},
		),
		StreamBytesTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "melodee_stream_bytes_total",
				Help: "Total number of bytes streamed",
			},
		),

		// Job metrics
		JobDurationSeconds: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "melodee_job_duration_seconds",
				Help: "Duration of jobs in seconds",
			},
			[]string{"queue", "type", "status"},
		),

		// Metadata metrics
		MetadataDriftTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "melodee_metadata_drift_total",
				Help: "Total number of metadata drift events",
			},
		),

		// Quarantine metrics
		QuarantineTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "melodee_quarantine_total",
				Help: "Total quarantine events",
			},
			[]string{"reason"},
		),

		// Health metrics
		HealthStatus: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "melodee_health_status",
				Help: "Health status of dependencies (1=ok, 0=down)",
			},
			[]string{"dependency"},
		),
	}
}

// InitializeMetrics sets up default values for metrics
func InitializeMetrics() *Metrics {
	metrics := NewMetrics()
	
	// Initialize health metrics with default values
	metrics.HealthStatus.WithLabelValues("db").Set(0)
	metrics.HealthStatus.WithLabelValues("redis").Set(0)
	
	return metrics
}