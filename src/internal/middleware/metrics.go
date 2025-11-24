package middleware

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Initialize Prometheus metrics
var (
	// Request counter - tracks total requests by method, route, and status
	RequestCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_requests_total",
			Help: "Total number of API requests by method, route, and status",
		},
		[]string{"method", "route", "status"},
	)

	// Request duration histogram - tracks response times by method, route, and status
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "api_request_duration_seconds",
			Help: "Histogram of request durations by method, route, and status",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "route", "status"},
	)

	// Request size histogram - tracks request body sizes
	RequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "api_request_size_bytes",
			Help: "Histogram of request sizes by method and route",
			Buckets: []float64{100, 500, 1000, 5000, 10000, 50000, 100000},
		},
		[]string{"method", "route"},
	)

	// Response size histogram - tracks response body sizes
	ResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "api_response_size_bytes",
			Help: "Histogram of response sizes by method and route",
			Buckets: []float64{100, 500, 1000, 5000, 10000, 50000, 100000, 500000, 1000000},
		},
		[]string{"method", "route"},
	)

	// Error counter - tracks errors by type
	ErrorCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_errors_total",
			Help: "Total number of API errors by method, route, and error code",
		},
		[]string{"method", "route", "error_code"},
	)
)

// MetricsMiddleware creates middleware that records request metrics
func MetricsMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Store the start time in the context for potential use by other handlers
		c.Locals("request_start_time", start)

		// Get route path - normalize it to remove variable parts like IDs
		route := c.Route().Path
		method := c.Method()

		// Continue with the request
		err := c.Next()

		// Calculate duration after the request completes
		duration := time.Since(start).Seconds()

		// Get the status code
		statusCode := c.Response().StatusCode()
		statusCodeStr := strconv.Itoa(statusCode)

		// Record metrics
		RequestCount.WithLabelValues(method, route, statusCodeStr).Inc()
		RequestDuration.WithLabelValues(method, route, statusCodeStr).Observe(duration)

		// Record request and response sizes (if available)
		reqSize := len(c.Request().Body())
		if reqSize > 0 {
			RequestSize.WithLabelValues(method, route).Observe(float64(reqSize))
		}

		respSize := len(c.Response().Body())
		if respSize > 0 {
			ResponseSize.WithLabelValues(method, route).Observe(float64(respSize))
		}

		// If there was an error, record it in the error counter
		if err != nil || statusCode >= 400 {
			errorCode := statusCodeStr
			if err != nil {
				// If we have a specific error, we could use a more specific label
				// For now, just use the status code
			}
			ErrorCount.WithLabelValues(method, route, errorCode).Inc()
		}

		return err
	}
}

// GetRequestStartTime retrieves the request start time from the context
func GetRequestStartTime(c *fiber.Ctx) (time.Time, bool) {
	startTime, ok := c.Locals("request_start_time").(time.Time)
	return startTime, ok
}