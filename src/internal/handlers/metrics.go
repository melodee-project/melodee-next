package handlers

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsHandler handles metrics-related requests
type MetricsHandler struct{}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler() *MetricsHandler {
	return &MetricsHandler{}
}

// Metrics returns a Fiber handler for Prometheus metrics
func (h *MetricsHandler) Metrics() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Create prometheus handler
		handler := promhttp.Handler()
		
		// Create a basic http.ResponseWriter implementation
		writer := &fiberResponseWriter{c: c}
		
		// Create a basic http.Request
		uri := c.Request().URI()
		urlStr := string(uri.RequestURI())
		httpReqURL, err := url.ParseRequestURI(urlStr)
		if err != nil {
			return fmt.Errorf("failed to parse request URI: %w", err)
		}

		req := &http.Request{
			Method: c.Method(),
			URL:    httpReqURL,
			Header: make(http.Header),
		}
		
		// Copy headers from Fiber request
		c.Request().Header.VisitAll(func(key, value []byte) {
			req.Header.Set(string(key), string(value))
		})
		
		// Serve metrics
		handler.ServeHTTP(writer, req)
		
		return nil
	}
}

// fiberResponseWriter implements http.ResponseWriter for Fiber context
type fiberResponseWriter struct {
	c *fiber.Ctx
}

func (w *fiberResponseWriter) Header() http.Header {
	return make(http.Header)
}

func (w *fiberResponseWriter) Write(data []byte) (int, error) {
	return w.c.Write(data)
}

func (w *fiberResponseWriter) WriteHeader(statusCode int) {
	w.c.Status(statusCode)
}

// RegisterCustomMetrics registers custom application metrics
func RegisterCustomMetrics() {
	// This function is kept for compatibility but metrics are now registered
	// automatically through the promauto package in the middleware
	// No need to register metrics here since they're already registered in the middleware package
}