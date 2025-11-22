package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

const (
	ServiceName = "melodee"
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

	// Capacity metrics
	CapacityPercent *prometheus.GaugeVec

	// Capacity probe metrics
	CapacityProbeFailuresTotal prometheus.Counter

	// Server metrics - for basic HTTP request metrics
	RequestDurationSeconds *prometheus.HistogramVec
	RequestTotal           *prometheus.CounterVec
	ResponseSizeBytes      *prometheus.HistogramVec
}

// NewMetrics creates a new Metrics instance with all required metrics registered
func NewMetrics() *Metrics {
	return &Metrics{
		// Stream metrics (from TECHNICAL_SPEC.md)
		StreamRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "melodee_stream_requests_total",
				Help: "Total number of stream requests",
			},
			[]string{"status", "format"}, // status could be "success", "error"; format could be "mp3", "flac", etc.
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

		// Capacity metrics
		CapacityPercent: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "melodee_capacity_percent",
				Help: "Capacity percentage by library",
			},
			[]string{"library"},
		),

		// Capacity probe metrics
		CapacityProbeFailuresTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "melodee_capacity_probe_failures_total",
				Help: "Total number of capacity probe failures",
			},
		),

		// Server metrics - basic HTTP metrics
		RequestDurationSeconds: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "http_request_duration_seconds",
				Help: "Duration of HTTP requests in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "route", "status_code"},
		),

		RequestTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "route", "status_code"},
		),

		ResponseSizeBytes: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "http_response_size_bytes",
				Help: "Size of HTTP responses in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "route", "status_code"},
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

// RecordStreamRequest records a stream request
func (m *Metrics) RecordStreamRequest(status, format string) {
	m.StreamRequestsTotal.WithLabelValues(status, format).Inc()
}

// RecordStreamBytes records bytes streamed
func (m *Metrics) RecordStreamBytes(bytes int64) {
	m.StreamBytesTotal.Add(float64(bytes))
}

// RecordJobDuration records the duration of a job
func (m *Metrics) RecordJobDuration(queue, jobType, status string, duration time.Duration) {
	m.JobDurationSeconds.WithLabelValues(queue, jobType, status).Observe(duration.Seconds())
}

// RecordMetadataDrift increments the metadata drift counter
func (m *Metrics) RecordMetadataDrift() {
	m.MetadataDriftTotal.Inc()
}

// RecordQuarantineEvent records a quarantine event with reason
func (m *Metrics) RecordQuarantineEvent(reason string) {
	m.QuarantineTotal.WithLabelValues(reason).Inc()
}

// SetHealthStatus sets health status for a dependency
func (m *Metrics) SetHealthStatus(dependency string, isHealthy bool) {
	if isHealthy {
		m.HealthStatus.WithLabelValues(dependency).Set(1)
	} else {
		m.HealthStatus.WithLabelValues(dependency).Set(0)
	}
}

// SetCapacityPercent sets capacity percentage for a library
func (m *Metrics) SetCapacityPercent(library string, percent float64) {
	m.CapacityPercent.WithLabelValues(library).Set(percent)
}

// RecordCapacityProbeFailure increments the capacity probe failure counter
func (m *Metrics) RecordCapacityProbeFailure() {
	m.CapacityProbeFailuresTotal.Inc()
}

// StartOpenTelemetry initializes OpenTelemetry with Prometheus exporter
func StartOpenTelemetry(serviceName, serviceVersion string) error {
	// Create a Prometheus exporter
	exporter, err := prometheus.New()
	if err != nil {
		return fmt.Errorf("failed to create Prometheus exporter: %w", err)
	}

	// Create a meter provider with the Prometheus exporter
	provider := metric.NewMeterProvider(
		metric.WithReader(exporter),
	)

	// Set the global meter provider
	otel.SetMeterProvider(provider)

	// Set service info as resource attributes
	otel.SetTracerProvider(metric.NewMeterProvider(
		metric.WithReader(exporter),
	))

	return nil
}

// SetupMetricsEndpoint configures an HTTP handler for the metrics endpoint
func SetupMetricsEndpoint() http.HandlerFunc {
	return promhttp.Handler().ServeHTTP
}

// HTTPMiddleware creates a middleware that records HTTP request metrics
func (m *Metrics) HTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := &responseWriter{ResponseWriter: w, statusCode: 200}
			
			next.ServeHTTP(ww, r)
			
			duration := time.Since(start)
			
			// Record metrics
			m.RequestDurationSeconds.WithLabelValues(
				r.Method,
				r.URL.Path,
				fmt.Sprintf("%d", ww.statusCode),
			).Observe(duration.Seconds())
			
			m.RequestTotal.WithLabelValues(
				r.Method,
				r.URL.Path,
				fmt.Sprintf("%d", ww.statusCode),
			).Inc()
			
			m.ResponseSizeBytes.WithLabelValues(
				r.Method,
				r.URL.Path,
				fmt.Sprintf("%d", ww.statusCode),
			).Observe(float64(ww.size))
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code and response size
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// InstrumentFunctionTime measures the execution time of a function and records it in a histogram
func (m *Metrics) InstrumentFunctionTime(histogram *prometheus.HistogramVec, labels prometheus.Labels, fn func()) {
	start := time.Now()
	fn()
	duration := time.Since(start)
	
	if histogram != nil {
		histogram.With(labels).Observe(duration.Seconds())
	}
}