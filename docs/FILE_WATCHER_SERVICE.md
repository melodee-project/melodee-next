# File Watcher Service

## Overview

The File Watcher Service is a standalone service that monitors file system changes in specified directories and triggers appropriate background jobs via Melodee's existing Asynq job queue system. This service provides real-time file monitoring capabilities while maintaining the existing architectural patterns and integration with the admin UI.

## Architecture

### Service Structure
- Located at `src/watcher/`
- Uses the same configuration, database, and Redis connection patterns as other services
- Built with minimal dependencies to maintain stability
- Containerized with Docker for consistent deployment

### Integration Points
- **Redis**: Uses existing Redis connection for Asynq job queue
- **Database**: Connects to existing PostgreSQL database for configuration and logging
- **API Service**: Exposes health check and status endpoints for monitoring
- **Admin UI**: Configurable via admin UI and provides logging/status information

## Database Schema Changes

The file watcher service requires a minor update to the existing `libraries` table:

- Add `is_watched` boolean column with default `false`
- This allows any library to be watched without requiring separate directory configuration
- The staging library can be enabled for watching by setting `is_watched = true`

## Features

### 1. Directory Monitoring
- Monitors library paths from the `libraries` table where `is_watched = true`
- Starts with staging library path monitoring for the initial use case
- Supports different event types: create, modify, delete, rename
- Configurable file patterns to watch (e.g., "*.mp3", "*.flac")
- Debouncing to prevent duplicate events

### 2. Job Triggering
- Maps file system events to Asynq job types
- Primary use case: when new files are added to watched staging libraries, triggers the "Staging Scan Job" (`media.TypeStagingScan`) to process new media
- Leverages existing job types like `TypeLibraryScan`, `TypeLibraryProcess`, and the staging scan job noted above
- Optional file-specific job types (Phase 2):
    - `file:created`
    - `file:modified`
    - `file:removed`
    - `directory:changed`

### 3. Configuration Management
- Database-backed configuration using Melodee's settings system
- Admin UI integration for configuration changes
- Runtime configuration updates without service restart

## Configuration Options

### Configuration Parameters
- `file_watch.enabled`: Enable/disable file watching (boolean)
- `file_watch.directories`: Array of directory paths to monitor
- `file_watch.patterns`: File patterns to watch (extensions, wildcards)
- `file_watch.debounce_time`: Time window to debounce events (duration)
- `file_watch.scan_workers`: Number of concurrent workers for processing events
- `file_watch.buffer_size`: Buffer size for event processing

### API Endpoints
- `GET /api/config/file-watcher`: Retrieve current configuration
- `PUT /api/config/file-watcher`: Update configuration
- `GET /api/config/file-watcher/schema`: Get configuration schema for UI

## Requirements Implementation

### 1. Configuration Options
The file watcher service configuration is fully manageable through the admin UI:

- **Database Integration**: Configuration primarily managed through the `libraries` table with a new `is_watched` boolean field to indicate which library paths should be monitored, with service-level settings stored in the `settings` table
- **Library-Based Watching**: The service automatically watches the `path` of any library where `is_watched = true`, starting with the staging library use case
- **Real-time Updates**: Changes made in the admin UI update the database, which the service monitors for configuration changes
- **UI Components**: Integrate with existing library management page with:
  - "Watch" toggle per library (including a prominent option for staging libraries)
  - File pattern configuration to specify which types of files to monitor in watched libraries
  - Service enable/disable toggle for overall file watching functionality
  - Advanced settings (debounce, workers, etc.)

### 2. Status Monitoring
Multiple status monitoring mechanisms ensure the service health is visible:

- **Health Check Endpoint**: Service exposes `/healthz` endpoint with detailed status
- **Database Heartbeat**: Service periodically updates a status record in the database
- **API Proxy**: Main API service proxies status requests from UI to watcher service
- **Status Endpoints**:
  - `GET /api/status/watcher`: Current service status
  - `GET /api/status/watcher/health`: Detailed health information
- **UI Integration**: Status dashboard shows:
  - Service uptime and availability
  - Number of active watched libraries
  - List of currently watched library paths
  - Last heartbeat timestamp
  - Error status and counts
  - Service metrics

### 3. Activity Logs
Comprehensive logging system that integrates with the existing logging infrastructure:

- **Database Logging**: All file watcher events logged to database using existing logging table
- **Log Levels**: Support for info, warning, error, and debug levels
- **Log Endpoints**:
  - `GET /api/logs/watcher`: Retrieve recent file watcher logs
  - `GET /api/logs/watcher/events`: Retrieve specific file system events
  - `GET /api/logs/watcher/events?library_id=:id`: Retrieve events for specific library
  - `GET /api/logs/watcher/stats`: Get statistics and metrics
- **Real-time Streaming**: Server-Sent Events (SSE) for live log viewing in UI
- **UI Components**:
  - Filterable log viewer (by library, event type, date range)
  - Search functionality
  - Export capabilities
  - Live log streaming option

## Implementation Details

### Event Model and Recursion
- Uses `fsnotify` for filesystem events. Canonical event set: `Create`, `Write`, `Remove`, `Rename`, `Chmod`.
- Event mapping (recommended defaults):
    - `Create|Write` → file upsert handling (debounced)
    - `Remove` → file removed
    - `Rename` → file moved/renamed
    - `Chmod` → ignored by default (configurable)
- Recursive watching: `fsnotify` is non-recursive. The service walks existing subdirectories on startup and adds watches; it dynamically adds watches for newly created subdirectories and removes watches when directories are deleted.
- Debounce scope: debounce per-path within `file_watch.debounce_time` to collapse rapid `Create`/`Write` bursts. Use trailing-edge debounce to ensure final state.

### Service Startup
```go
// Example startup logic
func main() {
    // Load configuration
    cfg, err := config.LoadConfig()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Initialize database connection
    dbManager, err := database.NewDatabaseManager(&cfg.Database, nil)
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }

    // Initialize Redis connection for Asynq
    redisAddr := cfg.Redis.Address
    client := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})

    // Initialize file watcher
    watcher, err := NewFileWatcher(cfg.FileWatch, dbManager.GetGormDB(), client)
    if err != nil {
        log.Fatalf("Failed to create file watcher: %v", err)
    }

    // Start watching
    if err := watcher.Start(); err != nil {
        log.Fatalf("Failed to start file watcher: %v", err)
    }

    // Start health check server
    go startHealthServer(watcher)

    // Wait for shutdown signal
    waitForShutdown(watcher)
}
```

### Library Path Monitoring
The service will periodically (or in response to DB changes) query the libraries table to identify watched paths:

```go
// Example library monitoring logic
func (fw *FileWatcher) updateWatchedLibraries() error {
    var watchedLibraries []struct {
        ID   int32  `gorm:"column:id"`
        Path string `gorm:"column:path"`
    }

    // Query all libraries where is_watched = true
    if err := fw.db.Table("libraries").Select("id, path").Where("is_watched = ?", true).Find(&watchedLibraries).Error; err != nil {
        return fmt.Errorf("failed to query watched libraries: %w", err)
    }

    // Update fsnotify to watch new paths and unwatch removed paths
    for _, lib := range watchedLibraries {
        if err := fw.addWatchPath(lib.Path); err != nil {
            log.Printf("Error adding watch for library path %s: %v", lib.Path, err)
        }
    }

    return nil
}

// isStagingLibraryPath checks if the given file path belongs to a watched staging library
func (fw *FileWatcher) isStagingLibraryPath(filePath string) bool {
    var count int64
    // Safer prefix match: ensure filePath starts with library.path followed by '/' (LIKE path || '/%')
    // Prevents false positives where path is a prefix of a different directory name.
    err := fw.db.Table("libraries").
        Where("is_watched = ? AND type = 'staging' AND ? LIKE (path || '/%')", true, filePath).
        Count(&count).Error

    return err == nil && count > 0
}
```

### Event Handling
```go
// Example event handling
type FileEvent struct {
    Path      string    `json:"path"`
    EventType string    `json:"event_type"`
    Timestamp time.Time `json:"timestamp"`
}

func (fw *FileWatcher) handleEvent(event FileEvent) error {
    // Log the event
    fw.logEvent(event)
    
    // Check if this is a staging library path (where new files should trigger staging scan)
    if fw.isStagingLibraryPath(event.Path) {
        // For staging libraries, trigger the Staging Scan Job directly
        // Enqueue the staging scan job
        payload := map[string]interface{}{
            "source":  "file_watcher",
            "path":    event.Path,
            "type":    event.EventType,
            "trigger": "new_file_added",
        }
        payloadBytes, _ := json.Marshal(payload)
        task := asynq.NewTask(media.TypeStagingScan, payloadBytes)
        return fw.client.Enqueue(task, asynq.Queue("maintenance")) // queue configurable
    } else {
        // For other watched libraries, optionally use file-specific job types (Phase 2)
        var jobType string
        switch event.EventType {
        case "create", "modify":
            jobType = "file:created"
        case "remove":
            jobType = "file:removed"
        default:
            jobType = "file:generic"
        }

        // Enqueue job
        payload, _ := json.Marshal(event)
        task := asynq.NewTask(jobType, payload)
        return fw.client.Enqueue(task)
    }
}
```

## Deployment

### Docker Integration
- New service in `docker-compose.yml` (and compatible with `podman compose` locally)
- Same networking and volume patterns as other services
- Mount host media paths read-only into the watcher container; ensure container user has read/exec on directories
- Proper dependency management (database, Redis)

### Health and Monitoring
- Integrated with existing Prometheus metrics
- Jaeger tracing support
- Health check endpoints conform to existing patterns
- Tune Linux inotify limits as needed (host and container):
    - `fs.inotify.max_user_watches`
    - `fs.inotify.max_user_instances`
    - Provide guidance in ops runbook for expected values at scale

## Security Considerations

- **File System Permissions**: Service requires appropriate permissions to monitor directories
- **Configuration Validation**: Input validation for directory paths to prevent path traversal
- **Resource Limits**: Limits on number of watched directories to prevent resource exhaustion
- **Rate Limiting**: Rate limiting on event processing to prevent system overload

## API Boundaries and Auth

- Watcher service exposes only internal endpoints: `/healthz` and `/metrics`.
- Admin-facing endpoints live in the main API and either proxy to the watcher or read from DB/metrics directly:
    - `GET /api/status/watcher` (proxy or aggregate)
    - `GET /api/status/watcher/health`
    - `GET /api/logs/watcher*`
- Inter-service communication: API → Watcher via stable service address (`WATCHER_HTTP_ADDR`) with short timeouts.
- All admin endpoints require JWT auth with appropriate admin scope/role; SSE log streams use the same auth and timeouts.

## Admin UI Integration

### New UI Components
- `/admin/watcher/config`: Configuration management page
- `/admin/watcher/status`: Service status and health
- `/admin/watcher/logs`: Activity logs viewer
- `/admin/watcher/stats`: Statistics and metrics

### API Integration
- All endpoints follow existing API patterns
- Authentication and authorization via existing JWT system
- Request/response schemas consistent with other admin endpoints
- API proxies to watcher for health/metrics when needed; otherwise reads DB-backed config and logs directly

## Error Handling and Resilience

- **Connection Recovery**: Jittered exponential backoff with caps when reconnecting to DB/Redis; emit readiness only after all deps healthy
- **Event Buffering**: In-memory ring buffer for transient outages with max size configurable; drop-oldest policy with metrics and warnings
- **Circuit Breakers**: Trip on repeated Redis enqueue or DB read errors; auto-reset on sustained success window
- **Graceful Degradation**: Continue operating with reduced functionality if parts fail (e.g., log-only when Redis unavailable)
- **Startup Gap**: Perform a lightweight scan of watched libraries on startup/config change to catch files created before watchers attached

## Ops and Scaling

- **Backpressure**: Use dedicated Asynq queue for watcher-originated tasks (e.g., `maintenance`), set rate limits and retries per library/type
- **Dedupe Keys**: Optionally set deduplication keys per file path + event window to avoid flooding
- **Batching**: (Optional) group multiple file events for the same directory into a single enqueue under heavy load

## Future Enhancements

- **File Type Validation**: Enhanced file type detection for media files
- **Smart Processing**: Intelligent processing based on file types and library locations
- **Library-Specific Rules**: Different processing rules based on library type (staging vs. production vs. inbound)
- **Performance Metrics**: Detailed performance and usage metrics per library
- **Custom Hooks**: Support for custom actions on specific file events per library
- **Batch Processing**: Batch event processing for high-volume scenarios

## Configuration

### Recommended Defaults
- `file_watch.enabled`: `false`
- `file_watch.patterns`: `*.mp3,*.flac,*.m4a,*.ogg,*.wav`
- `file_watch.ignore_patterns`: `*.part,~$*,.*` (temp/hidden files)
- `file_watch.debounce_time`: `1s` (tune based on ingest characteristics)
- `file_watch.scan_workers`: `4`
- `file_watch.buffer_size`: `1024`

### Runtime Updates
- Prefer Postgres `LISTEN/NOTIFY` on config tables to reduce polling; fall back to periodic polling if unavailable.

## Metrics

Expose Prometheus metrics:
- `watcher_events_total{event,type,library}`: Count of fs events observed
- `watcher_enqueue_total{queue,type}`: Count of tasks enqueued
- `watcher_errors_total{stage}`: Errors by stage (watch, debounce, enqueue, db)
- `watcher_watches{state}`: Current number of active watches (added/removed)
- `watcher_debounce_suppressed_total`: Events dropped by debounce

## Database Schema Changes

Add migration (e.g., `init-scripts/002_file_watcher.sql`):

```sql
ALTER TABLE libraries
    ADD COLUMN IF NOT EXISTS is_watched boolean NOT NULL DEFAULT false;
```

## Testing

- Unit: event mapping, debounce behavior, path prefix matching SQL generation
- Integration: temp directory tree with fsnotify events; verify enqueue calls and metrics updates
- Resilience: simulate Redis/DB outages; confirm backoff, buffering, and graceful degradation
- E2E: mass copy into staging library; ensure staging scan job enqueued and processed