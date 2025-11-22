package tracing

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	ServiceName    = "melodee"
	ServiceVersion = "1.0.0"
)

// Tracer holds the tracer instance
type Tracer struct {
	tracer trace.Tracer
	tp     *sdktrace.TracerProvider
}

// NewTracer creates a new tracer instance
func NewTracer(serviceName, collectorEndpoint string, useOTLP bool) (*Tracer, error) {
	// Create resource with service information
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(serviceName),
		semconv.ServiceVersionKey.String(ServiceVersion),
	)

	var exp sdktrace.SpanExporter
	var err error

	if useOTLP {
		// Use OTLP exporter (for sending traces to collector like Jaeger, Tempo, etc.)
		client := otlptracegrpc.NewClient(
			otlptracegrpc.WithEndpoint(collectorEndpoint),
			otlptracegrpc.WithInsecure(), // In production, use TLS
		)
		exp, err = otlptrace.New(context.Background(), client)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
		}
	} else {
		// Use stdout exporter for development/debugging
		exp, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, fmt.Errorf("failed to create stdout exporter: %w", err)
		}
	}

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)

	// Set global trace provider
	otel.SetTracerProvider(tp)

	// Set the propagator to support distributed tracing
	otel.SetTextMapPropagator(propagation.TraceContext{})

	tracer := &Tracer{
		tracer: tp.Tracer(serviceName),
		tp:     tp,
	}

	return tracer, nil
}

// StartSpan starts a new span with the provided name
func (t *Tracer) StartSpan(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, spanName, opts...)
}

// StartSpanWithAttributes starts a new span with the provided attributes
func (t *Tracer) StartSpanWithAttributes(ctx context.Context, spanName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	opts := []trace.SpanStartOption{
		trace.WithAttributes(attrs...),
	}
	return t.tracer.Start(ctx, spanName, opts...)
}

// AddAttributes adds attributes to the current span
func AddAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		span.SetAttributes(attrs...)
	}
}

// AddEvent adds an event to the current span
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

// SetSpanError marks the current span as having an error
func SetSpanError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		span.RecordError(err)
	}
}

// Shutdown gracefully shuts down the trace provider
func (t *Tracer) Shutdown(ctx context.Context) error {
	return t.tp.Shutdown(ctx)
}

// StreamTracingAttrs returns common attributes for stream operations
func StreamTracingAttrs(user_id, song_id, format string) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(ServiceName),
		attribute.String("component", "stream"),
	}

	if user_id != "" {
		attrs = append(attrs, attribute.String("user_id", user_id))
	}
	if song_id != "" {
		attrs = append(attrs, attribute.String("song_id", song_id))
	}
	if format != "" {
		attrs = append(attrs, attribute.String("format", format))
	}

	return attrs
}

// MetadataWritebackTracingAttrs returns common attributes for metadata writeback operations
func MetadataWritebackTracingAttrs(user_id string, song_ids []int64) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(ServiceName),
		attribute.String("component", "metadata.writeback"),
	}

	if user_id != "" {
		attrs = append(attrs, attribute.String("user_id", user_id))
	}

	if len(song_ids) > 0 {
		ids := make([]string, len(song_ids))
		for i, id := range song_ids {
			ids[i] = fmt.Sprintf("%d", id)
		}
		attrs = append(attrs, attribute.StringSlice("song_ids", ids))
	}

	return attrs
}

// LibraryScanTracingAttrs returns common attributes for library scan operations
func LibraryScanTracingAttrs(user_id string, library_ids []int32) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(ServiceName),
		attribute.String("component", "library.scan"),
	}

	if user_id != "" {
		attrs = append(attrs, attribute.String("user_id", user_id))
	}

	if len(library_ids) > 0 {
		ids := make([]string, len(library_ids))
		for i, id := range library_ids {
			ids[i] = fmt.Sprintf("%d", id)
		}
		attrs = append(attrs, attribute.StringSlice("library_ids", ids))
	}

	return attrs
}

// JobProcessingTracingAttrs returns common attributes for job processing operations
func JobProcessingTracingAttrs(job_id, queue, job_type string, attempt int) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(ServiceName),
		attribute.String("component", "job.processing"),
	}

	if job_id != "" {
		attrs = append(attrs, attribute.String("job.id", job_id))
	}
	if queue != "" {
		attrs = append(attrs, attribute.String("job.queue", queue))
	}
	if job_type != "" {
		attrs = append(attrs, attribute.String("job.type", job_type))
	}
	attrs = append(attrs, attribute.Int("job.attempt", attempt))

	return attrs
}

// HTTPTracingAttrs returns common attributes for HTTP requests
func HTTPTracingAttrs(method, route, status_code string, duration time.Duration) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(ServiceName),
		attribute.String("component", "http"),
		semconv.HTTPRequestMethodKey.String(method),
		semconv.HTTPRouteKey.String(route),
		semconv.HTTPStatusCodeKey.String(status_code),
		attribute.Float64("http.duration_ms", float64(duration.Milliseconds())),
	}

	return attrs
}

// WithTracingContext adds tracing context to an existing context
func WithTracingContext(ctx context.Context, spanName string, attrs ...attribute.KeyValue) (context.Context, trace.Span, func()) {
	tracer := otel.Tracer(ServiceName)
	ctx, span := tracer.Start(
		ctx,
		spanName,
		trace.WithAttributes(attrs...),
	)

	cleanup := func() {
		span.End()
	}

	return ctx, span, cleanup
}

// TracingMiddleware creates a middleware that automatically adds tracing to incoming requests
func (t *Tracer) TracingMiddleware(serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract tracing context from headers
			propagator := otel.GetTextMapPropagator()
			ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			// Start a new span for the request
			ctx, span := t.StartSpanWithAttributes(ctx, r.URL.Path,
				semconv.HTTPMethodKey.String(r.Method),
				semconv.HTTPURLKey.String(r.URL.String()),
				semconv.HTTPUserAgentKey.String(r.UserAgent()),
				semconv.NetPeerIPKey.String(r.RemoteAddr),
			)
			
			defer span.End()

			// Add the tracing context to the request context
			r = r.WithContext(ctx)

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}
}