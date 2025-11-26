# Melodee Logging System

## Overview

The Melodee logging system provides comprehensive structured logging with database persistence, real-time viewing, and powerful filtering capabilities.

## Architecture

### Backend Components

1. **Structured Logging (Zerolog)**
   - Fast, zero-allocation JSON logging
   - Contextual fields (request_id, user_id, trace_id, etc.)
   - Multiple log levels: debug, info, warn, error, fatal, panic
   - Configured via `config.yaml` under `logging.level` and `logging.format`

2. **Database Storage**
   - Table: `melodee_log_entries`
   - Stores logs for querying, auditing, and analysis
   - Automatic indexing for fast searches
   - Full-text search on messages and errors
   - Configurable retention periods

3. **Log Handlers**
   - `LogsHandler`: REST API endpoints for log access
   - Filtering by level, module, time range, user, library, job type
   - Pagination support
   - Export/download functionality
   - Statistics and metrics

### Frontend Components

1. **LogViewer Component**
   - Real-time log viewing with auto-refresh
   - Advanced filtering (level, module, search, time range)
   - Tail mode for continuous monitoring
   - Pagination for large log sets
   - Export logs as JSON
   - Color-coded log levels
   - Statistics dashboard

## Usage

### Backend Logging

#### Basic Logging

```go
import "melodee/internal/logging"

// Create logger
logger := logging.NewLogger(logging.InfoLevel, os.Stdout)

// Simple logging
logger.Info("User logged in")
logger.Warn("High memory usage detected")
logger.Error("Failed to process file")
```

#### Contextual Logging

```go
// Log with context fields
logCtx := logging.LogContext{
    Module:    "media",
    Function:  "ScanLibrary",
    LibraryID: 1,
    UserID:    42,
}

contextLogger := logger.WithContextFields(logCtx)
contextLogger.Info().Msg("Starting library scan")
```

#### HTTP Request Logging

```go
// Fiber middleware automatically logs requests
logger.LogHTTPRequest(c, duration)
```

#### Job Processing Logging

```go
logger.LogJobProcessing(
    "default",        // queue
    "library.scan",   // job type
    1,                // attempt
    duration,         // duration
    true,             // success
    "",               // error message if failed
)
```

#### Database Persistence

```go
// Create storage
logStorage := logging.NewLogStorage(db)

// Create logger with database hook
dbLogger := logging.NewContextualDatabaseLogger(&logger, logStorage)

// Log with automatic database storage
dbLogger.LogWithStorage(ctx, zerolog.InfoLevel, "Operation completed", map[string]interface{}{
    "module": "media",
    "library_id": 1,
    "duration_ms": 1500,
})
```

### API Endpoints

#### Get Logs
```
GET /api/admin/logs
Query Parameters:
  - level: debug|info|warn|error|fatal
  - module: string
  - job_type: string
  - search: string (searches message and error fields)
  - library_id: integer
  - user_id: integer
  - request_id: string
  - start_time: RFC3339 timestamp
  - end_time: RFC3339 timestamp
  - page: integer (default: 1)
  - page_size: integer (default: 100, max: 1000)
```

#### Get Log Statistics
```
GET /api/admin/logs/stats
Query Parameters:
  - since: RFC3339 timestamp (default: last 24 hours)

Response:
{
  "by_level": [
    {"level": "info", "count": 1250},
    {"level": "error", "count": 15}
  ],
  "error_count": 15,
  "warn_count": 42,
  "since": "2025-11-25T00:00:00Z",
  "generated_at": "2025-11-26T12:00:00Z"
}
```

#### Download Logs
```
GET /api/admin/logs/download
Query Parameters: (same as Get Logs)
Returns: JSON file download
```

#### Cleanup Old Logs
```
POST /api/admin/logs/cleanup
Query Parameters:
  - older_than_days: integer (default: 30)

Response:
{
  "status": "ok",
  "deleted": 5432,
  "message": "Old logs cleaned up successfully"
}
```

### Frontend Usage

#### Accessing Log Viewer

Navigate to: `http://localhost:5173/admin/logs`

#### Features

1. **Real-time Monitoring**
   - Toggle "Auto Refresh" to poll every 5 seconds
   - Enable "Tail Mode" to auto-scroll to newest logs

2. **Filtering**
   - By Level: All, Debug, Info, Warning, Error, Fatal
   - By Module: Enter module name (e.g., "media", "auth")
   - Search: Full-text search in messages and errors
   - Time Range: Filter by start/end timestamps

3. **Statistics Dashboard**
   - Errors (24h): Count of error and fatal logs
   - Warnings (24h): Count of warning logs
   - Total Logs: Total matching current filters
   - Auto Refresh: Toggle for real-time updates

4. **Export**
   - Click "Download" to export filtered logs as JSON
   - Includes all fields and metadata
   - Useful for offline analysis or archiving

5. **Pagination**
   - Navigate through large log sets
   - Configurable page size
   - Shows current position and total count

## Log Levels

| Level | Use Case | Example |
|-------|----------|---------|
| **DEBUG** | Development details, verbose output | Variable values, state changes |
| **INFO** | Normal operations | Job completed, user logged in |
| **WARN** | Recoverable issues | High disk usage, slow query |
| **ERROR** | Failed operations | File not found, API error |
| **FATAL** | System-critical failures | Database connection lost |
| **PANIC** | Immediate crash with stack trace | Unrecoverable error |

## Log Retention

Configure automatic cleanup via cron job or manual API call:

```bash
# Delete logs older than 30 days
curl -X POST "http://localhost:8080/api/admin/logs/cleanup?older_than_days=30" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## Best Practices

1. **Use Appropriate Log Levels**
   - Don't log errors as info
   - Use debug sparingly in production
   - Reserve fatal for unrecoverable errors

2. **Add Context**
   - Always include module and function names
   - Add IDs (user_id, library_id, job_id) for traceability
   - Include duration for performance monitoring

3. **Sensitive Data**
   - Never log passwords or tokens
   - Redact PII when logging user data
   - Be careful with file paths

4. **Performance**
   - Database writes are async (non-blocking)
   - Logs are indexed for fast queries
   - Set appropriate log level in production

5. **Monitoring**
   - Check error/warning counts daily
   - Set up alerts for fatal errors
   - Review slow operations (high duration_ms)

## Database Schema

```sql
CREATE TABLE melodee_log_entries (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    level VARCHAR(10) NOT NULL,
    message TEXT NOT NULL,
    module VARCHAR(100),
    function VARCHAR(100),
    request_id VARCHAR(50),
    user_id BIGINT,
    trace_id VARCHAR(50),
    ip VARCHAR(50),
    route VARCHAR(255),
    status INTEGER,
    duration BIGINT,
    queue VARCHAR(50),
    job_type VARCHAR(100),
    library_id INTEGER,
    file_path VARCHAR(1024),
    error TEXT,
    stack TEXT,
    metadata JSONB
);
```

## Configuration

Edit `config.yaml`:

```yaml
logging:
  level: "info"      # debug|info|warn|error|fatal
  format: "json"     # json|text (json recommended for production)
```

## Troubleshooting

### Logs Not Appearing in Database
- Check database connection
- Verify migration 007 has run
- Check application has write permissions

### High Database Size
- Run cleanup API regularly
- Adjust retention period
- Consider log rotation

### Performance Issues
- Reduce log level in production
- Increase cleanup frequency
- Check database indexes exist

## Future Enhancements

Potential additions:
- WebSocket streaming for real-time logs
- Loki integration for centralized logging
- Grafana dashboard integration
- Log aggregation across worker nodes
- Alert rules based on log patterns
- Machine learning for anomaly detection
