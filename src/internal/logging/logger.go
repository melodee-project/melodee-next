package logging

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

// LogLevel represents the logging level
type LogLevel string

const (
	DebugLevel LogLevel = "debug"
	InfoLevel  LogLevel = "info"
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
	FatalLevel LogLevel = "fatal"
	PanicLevel LogLevel = "panic"
)

// Logger holds the zerolog logger instance
type Logger struct {
	logger zerolog.Logger
}

// LogContext holds contextual information for logging
type LogContext struct {
	RequestID  string      `json:"req_id,omitempty"`
	UserID     int64       `json:"user_id,omitempty"`
	TraceID    string      `json:"trace_id,omitempty"`
	SpanID     string      `json:"span_id,omitempty"`
	IP         string      `json:"ip,omitempty"`
	Route      string      `json:"route,omitempty"`
	Status     int         `json:"status,omitempty"`
	Duration   int64       `json:"duration_ms,omitempty"`
	Queue      string      `json:"queue,omitempty"`
	JobType    string      `json:"job_type,omitempty"`
	Attempt    int         `json:"attempt,omitempty"`
	LibraryID  int32       `json:"library_id,omitempty"`
	FilePath   string      `json:"file_path,omitempty"`
	Module     string      `json:"module,omitempty"`
	Function   string      `json:"function,omitempty"`
}

// NewLogger creates a new logger instance with the specified log level
func NewLogger(logLevel LogLevel, output io.Writer) *Logger {
	if output == nil {
		output = os.Stdout
	}

	// Parse log level
	level, err := zerolog.ParseLevel(string(logLevel))
	if err != nil {
		level = zerolog.InfoLevel // Default to info level
	}

	// Create a new logger
	logger := zerolog.New(output).
		Level(level).
		With().
		Timestamp().
		Logger()

	return &Logger{
		logger: logger,
	}
}

// WithContext adds contextual fields to the logger
func (l *Logger) WithContext(ctx context.Context) *zerolog.Logger {
	logCtx := l.logger.With()

	// Add request ID if available in context
	if reqID := GetRequestID(ctx); reqID != "" {
		logCtx = logCtx.Str("req_id", reqID)
	}

	// Add trace and span IDs if available
	if spanCtx := trace.SpanContextFromContext(ctx); spanCtx.IsValid() {
		logCtx = logCtx.Str("trace_id", spanCtx.TraceID().String())
		logCtx = logCtx.Str("span_id", spanCtx.SpanID().String())
	}

	// Add user ID if available in context
	if userID := GetUserIDFromContext(ctx); userID != 0 {
		logCtx = logCtx.Int64("user_id", userID)
	}

	// Create the contextual logger
	contextualLogger := logCtx.Logger()

	return &contextualLogger
}

// Debug logs a debug message with contextual fields
func (l *Logger) Debug(msg string) *zerolog.Event {
	return l.logger.Debug().Msg(msg)
}

// Info logs an info message with contextual fields
func (l *Logger) Info(msg string) *zerolog.Event {
	return l.logger.Info().Msg(msg)
}

// Warn logs a warning message with contextual fields
func (l *Logger) Warn(msg string) *zerolog.Event {
	return l.logger.Warn().Msg(msg)
}

// Error logs an error message with contextual fields
func (l *Logger) Error(msg string) *zerolog.Event {
	return l.logger.Error().Msg(msg)
}

// Fatal logs a fatal message with contextual fields, then calls os.Exit(1)
func (l *Logger) Fatal(msg string) *zerolog.Event {
	return l.logger.Fatal().Msg(msg)
}

// Panic logs a panic message with contextual fields, then panics
func (l *Logger) Panic(msg string) *zerolog.Event {
	return l.logger.Panic().Msg(msg)
}

// WithField adds a single field to the logger
func (l *Logger) WithField(key string, value interface{}) *zerolog.Logger {
	logger := l.logger.With().Interface(key, value).Logger()
	return &logger
}

// WithFields adds multiple fields to the logger
func (l *Logger) WithFields(fields map[string]interface{}) *zerolog.Logger {
	logCtx := l.logger.With()
	
	for key, value := range fields {
		logCtx = logCtx.Interface(key, value)
	}
	
	logger := logCtx.Logger()
	return &logger
}

// WithContextFields adds context-specific fields to the logger
func (l *Logger) WithContextFields(ctx LogContext) *zerolog.Logger {
	logCtx := l.logger.With()
	
	if ctx.RequestID != "" {
		logCtx = logCtx.Str("req_id", ctx.RequestID)
	}
	if ctx.UserID != 0 {
		logCtx = logCtx.Int64("user_id", ctx.UserID)
	}
	if ctx.TraceID != "" {
		logCtx = logCtx.Str("trace_id", ctx.TraceID)
	}
	if ctx.SpanID != "" {
		logCtx = logCtx.Str("span_id", ctx.SpanID)
	}
	if ctx.IP != "" {
		logCtx = logCtx.Str("ip", ctx.IP)
	}
	if ctx.Route != "" {
		logCtx = logCtx.Str("route", ctx.Route)
	}
	if ctx.Status != 0 {
		logCtx = logCtx.Int("status", ctx.Status)
	}
	if ctx.Duration != 0 {
		logCtx = logCtx.Int64("duration_ms", ctx.Duration)
	}
	if ctx.Queue != "" {
		logCtx = logCtx.Str("queue", ctx.Queue)
	}
	if ctx.JobType != "" {
		logCtx = logCtx.Str("job_type", ctx.JobType)
	}
	if ctx.Attempt != 0 {
		logCtx = logCtx.Int("attempt", ctx.Attempt)
	}
	if ctx.LibraryID != 0 {
		logCtx = logCtx.Int32("library_id", ctx.LibraryID)
	}
	if ctx.FilePath != "" {
		logCtx = logCtx.Str("file_path", ctx.FilePath)
	}
	if ctx.Module != "" {
		logCtx = logCtx.Str("module", ctx.Module)
	}
	if ctx.Function != "" {
		logCtx = logCtx.Str("function", ctx.Function)
	}

	logger := logCtx.Logger()
	return &logger
}

// LogHTTPRequest logs HTTP request information
func (l *Logger) LogHTTPRequest(c *fiber.Ctx, duration time.Duration) {
	user, ok := c.Locals("user").(map[string]interface{})
	var userID int64
	if ok && user != nil {
		if id, exists := user["id"]; exists {
			if uid, ok := id.(int64); ok {
				userID = uid
			} else if sid, ok := id.(float64); ok {
				userID = int64(sid)
			}
		}
	}

	logCtx := l.logger.With().
		Str("req_id", GetRequestID(c.Context())).
		Int64("user_id", userID).
		Str("ip", c.IP()).
		Str("method", c.Method()).
		Str("url", c.OriginalURL()).
		Int("status", c.Response().StatusCode()).
		Int64("duration_ms", duration.Milliseconds()).
		Str("user_agent", c.Get("User-Agent")).
		Logger()

	logCtx.Info().Msg("HTTP request processed")
}

// LogJobProcessing logs job processing information
func (l *Logger) LogJobProcessing(queue, jobType string, attempt int, duration time.Duration, success bool, errorMsg string) {
	event := l.logger.With().
		Str("queue", queue).
		Str("job_type", jobType).
		Int("attempt", attempt).
		Int64("duration_ms", duration.Milliseconds()).
		Bool("success", success).
		Logger()

	if success {
		event.Info().Msg("Job processed successfully")
	} else {
		event.Error().Str("error", errorMsg).Msg("Job processing failed")
	}
}

// LogQuarantineEvent logs quarantine events
func (l *Logger) LogQuarantineEvent(filePath, reason, userID string) {
	l.logger.Info().
		Str("file_path", filePath).
		Str("reason", reason).
		Str("user_id", userID).
		Msg("File quarantined")
}

// LogCapacityCheck logs capacity check results
func (l *Logger) LogCapacityCheck(libraryID int32, path, filesystem string, usedPercent float64, threshold float64, status string) {
	l.logger.Info().
		Int32("library_id", libraryID).
		Str("path", path).
		Str("filesystem", filesystem).
		Float64("used_percent", usedPercent).
		Float64("threshold", threshold).
		Str("status", status).
		Msg("Capacity check performed")
}

// LogMetadataDrift logs metadata drift events
func (l *Logger) LogMetadataDrift(entityType, entityID, fieldName string, oldValue, newValue interface{}) {
	l.logger.Info().
		Str("entity_type", entityType).
		Str("entity_id", entityID).
		Str("field", fieldName).
		Interface("old_value", oldValue).
		Interface("new_value", newValue).
		Msg("Metadata drift detected")
}

// FiberLoggerMiddleware creates a Fiber-compatible logging middleware
func (l *Logger) FiberLoggerMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Process the request
		err := c.Next()

		duration := time.Since(start)

		// Log the request
		l.LogHTTPRequest(c, duration)

		return err
	}
}

// GetRequestID extracts request ID from context
func GetRequestID(ctx context.Context) string {
	// In Fiber, request ID could be stored differently depending on implementation
	// For now, this is a placeholder - in practice, you'd implement this based on your req ID generation
	return ""
}

// GetUserIDFromContext extracts user ID from context (placeholder implementation)
func GetUserIDFromContext(ctx context.Context) int64 {
	// This would extract the user ID from your authentication context
	// Implementation depends on how you store user info in context
	return 0
}

// SetLogLevel dynamically changes the logging level
func (l *Logger) SetLogLevel(logLevel LogLevel) error {
	level, err := zerolog.ParseLevel(string(logLevel))
	if err != nil {
		return fmt.Errorf("invalid log level: %s", logLevel)
	}

	l.logger = l.logger.Level(level)
	return nil
}