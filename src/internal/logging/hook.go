package logging

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rs/zerolog"
)

// DatabaseHook implements zerolog.Hook to store logs in database
type DatabaseHook struct {
	storage  *LogStorage
	minLevel zerolog.Level
}

// NewDatabaseHook creates a new database hook for zerolog
func NewDatabaseHook(storage *LogStorage, minLevel zerolog.Level) *DatabaseHook {
	return &DatabaseHook{
		storage:  storage,
		minLevel: minLevel,
	}
}

// Run implements the zerolog.Hook interface
func (h *DatabaseHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	// Only store logs at or above the minimum level
	if level < h.minLevel {
		return
	}

	// Create log entry from event fields
	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     level.String(),
		Message:   msg,
		Metadata:  "{}", // Initialize with empty JSON object
	}

	// Extract fields from the event's context
	// Note: This is a simplified implementation. In a real implementation,
	// you'd need to extract fields from the event's internal buffer
	// For now, we'll rely on manually setting fields when logging

	// Store in database synchronously for now (to debug)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.storage.Store(ctx, entry); err != nil {
		// Log the error to stderr for debugging
		println("ERROR storing log to database:", err.Error())
	}
}

// ContextualDatabaseLogger wraps a logger with database storage capabilities
type ContextualDatabaseLogger struct {
	logger  *zerolog.Logger
	storage *LogStorage
}

// NewContextualDatabaseLogger creates a logger that stores to database
func NewContextualDatabaseLogger(logger *zerolog.Logger, storage *LogStorage) *ContextualDatabaseLogger {
	return &ContextualDatabaseLogger{
		logger:  logger,
		storage: storage,
	}
}

// LogWithStorage logs a message and stores it in the database
func (l *ContextualDatabaseLogger) LogWithStorage(ctx context.Context, level zerolog.Level, msg string, fields map[string]interface{}) {
	// Create log entry
	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     level.String(),
		Message:   msg,
	}

	// Extract known fields
	if module, ok := fields["module"].(string); ok {
		entry.Module = module
	}
	if function, ok := fields["function"].(string); ok {
		entry.Function = function
	}
	if requestID, ok := fields["request_id"].(string); ok {
		entry.RequestID = requestID
	}
	if userID, ok := fields["user_id"].(int64); ok {
		entry.UserID = userID
	}
	if traceID, ok := fields["trace_id"].(string); ok {
		entry.TraceID = traceID
	}
	if ip, ok := fields["ip"].(string); ok {
		entry.IP = ip
	}
	if route, ok := fields["route"].(string); ok {
		entry.Route = route
	}
	if status, ok := fields["status"].(int); ok {
		entry.Status = status
	}
	if duration, ok := fields["duration_ms"].(int64); ok {
		entry.Duration = duration
	}
	if queue, ok := fields["queue"].(string); ok {
		entry.Queue = queue
	}
	if jobType, ok := fields["job_type"].(string); ok {
		entry.JobType = jobType
	}
	if libraryID, ok := fields["library_id"].(int32); ok {
		entry.LibraryID = libraryID
	}
	if filePath, ok := fields["file_path"].(string); ok {
		entry.FilePath = filePath
	}
	if err, ok := fields["error"].(string); ok {
		entry.Error = err
	}
	if stack, ok := fields["stack"].(string); ok {
		entry.Stack = stack
	}

	// Store remaining fields as metadata JSON
	metadata := make(map[string]interface{})
	knownFields := map[string]bool{
		"module": true, "function": true, "request_id": true, "user_id": true,
		"trace_id": true, "ip": true, "route": true, "status": true,
		"duration_ms": true, "queue": true, "job_type": true, "library_id": true,
		"file_path": true, "error": true, "stack": true,
	}
	for k, v := range fields {
		if !knownFields[k] {
			metadata[k] = v
		}
	}
	if len(metadata) > 0 {
		if metadataJSON, err := json.Marshal(metadata); err == nil {
			entry.Metadata = string(metadataJSON)
		}
	}

	// Store in database
	go func() {
		storeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = l.storage.Store(storeCtx, entry)
	}()

	// Also log to zerolog
	event := l.logger.WithLevel(level)
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}
