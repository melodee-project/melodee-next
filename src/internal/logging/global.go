package logging

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog"
)

// Global logger instance
var globalLogger *Logger

// InitGlobalLogger initializes the global logger instance
func InitGlobalLogger(level LogLevel, format string, storage *LogStorage) *Logger {
	var output = zerolog.ConsoleWriter{Out: os.Stdout}

	if format == "json" {
		globalLogger = NewLogger(level, os.Stdout)
	} else {
		globalLogger = NewLogger(level, &output)
	}

	// Add database hook if storage is provided
	if storage != nil {
		zerologLevel, _ := zerolog.ParseLevel(string(level))
		hook := NewDatabaseHook(storage, zerologLevel)
		globalLogger.logger = globalLogger.logger.Hook(hook)
	}

	return globalLogger
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *Logger {
	if globalLogger == nil {
		// Initialize with default settings if not already initialized
		globalLogger = NewLogger(InfoLevel, os.Stdout)
	}
	return globalLogger
}

// Quick logging functions using the global logger

// Debug logs a debug message
func Debug(msg string) {
	GetGlobalLogger().logger.Debug().Msg(msg)
}

// Debugf logs a formatted debug message
func Debugf(format string, args ...interface{}) {
	GetGlobalLogger().logger.Debug().Msg(fmt.Sprintf(format, args...))
}

// Info logs an info message
func Info(msg string) {
	GetGlobalLogger().logger.Info().Msg(msg)
}

// Infof logs a formatted info message
func Infof(format string, args ...interface{}) {
	GetGlobalLogger().logger.Info().Msg(fmt.Sprintf(format, args...))
}

// Warn logs a warning message
func Warn(msg string) {
	GetGlobalLogger().logger.Warn().Msg(msg)
}

// Warnf logs a formatted warning message
func Warnf(format string, args ...interface{}) {
	GetGlobalLogger().logger.Warn().Msg(fmt.Sprintf(format, args...))
}

// Error logs an error message
func Error(msg string) {
	GetGlobalLogger().logger.Error().Msg(msg)
}

// Errorf logs a formatted error message
func Errorf(format string, args ...interface{}) {
	GetGlobalLogger().logger.Error().Msg(fmt.Sprintf(format, args...))
}

// Fatal logs a fatal message and exits
func Fatal(msg string) {
	GetGlobalLogger().logger.Fatal().Msg(msg)
}

// Fatalf logs a formatted fatal message and exits
func Fatalf(format string, args ...interface{}) {
	GetGlobalLogger().logger.Fatal().Msg(fmt.Sprintf(format, args...))
}

// WithFields creates a logger with additional fields
func WithFields(fields map[string]interface{}) *zerolog.Logger {
	return GetGlobalLogger().WithFields(fields)
}

// WithContext creates a logger with context
func WithContext(ctx context.Context) *zerolog.Logger {
	return GetGlobalLogger().WithContext(ctx)
}

// WithModule creates a logger with module field
func WithModule(module string) *zerolog.Logger {
	logger := GetGlobalLogger().logger.With().Str("module", module).Logger()
	return &logger
}

// WithLibrary creates a logger with library_id field
func WithLibrary(libraryID int32) *zerolog.Logger {
	logger := GetGlobalLogger().logger.With().Int32("library_id", libraryID).Logger()
	return &logger
}

// WithUser creates a logger with user_id field
func WithUser(userID int64) *zerolog.Logger {
	logger := GetGlobalLogger().logger.With().Int64("user_id", userID).Logger()
	return &logger
}

// WithJob creates a logger with job-related fields
func WithJob(queue, jobType string) *zerolog.Logger {
	logger := GetGlobalLogger().logger.With().
		Str("queue", queue).
		Str("job_type", jobType).
		Logger()
	return &logger
}

// WithError creates a logger with error field
func WithError(err error) *zerolog.Logger {
	logger := GetGlobalLogger().logger.With().Err(err).Logger()
	return &logger
}
