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
- Primary use case: when new files are added to watched staging libraries, triggers the "Staging Scan Job" (`TypeStagingCron`) to process new media
- Leverages existing job types like `TypeLibraryScan`, `TypeLibraryProcess`, `TypeStagingCron`
- Creates new job types for file-specific operations:
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
    // Check if the file path starts with any watched library path that is of type 'staging'
    err := fw.db.Table("libraries").
        Where("is_watched = ? AND type = ? AND ?", true, "staging", gorm.Expr("starts_with(?, path)", filePath)).
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
        // This reuses the existing TypeStagingCron job to process new media
        payload := map[string]interface{}{
            "source":  "file_watcher",
            "path":    event.Path,
            "type":    event.EventType,
            "trigger": "new_file_added",
        }
        payloadBytes, _ := json.Marshal(payload)
        task := asynq.NewTask(TypeStagingCron, payloadBytes)
        return fw.client.Enqueue(task, asynq.Queue("maintenance"))
    } else {
        // For other watched libraries, use file-specific job types
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
- New service in `docker-compose.yml`
- Same networking and volume patterns as other services
- Proper dependency management (database, Redis)

### Health and Monitoring
- Integrated with existing Prometheus metrics
- Jaeger tracing support
- Health check endpoints conform to existing patterns

## Security Considerations

- **File System Permissions**: Service requires appropriate permissions to monitor directories
- **Configuration Validation**: Input validation for directory paths to prevent path traversal
- **Resource Limits**: Limits on number of watched directories to prevent resource exhaustion
- **Rate Limiting**: Rate limiting on event processing to prevent system overload

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

## Error Handling and Resilience

- **Connection Recovery**: Automatic reconnection to database and Redis
- **Event Buffering**: Buffer events during temporary service outages
- **Circuit Breakers**: Prevent cascading failures
- **Graceful Degradation**: Continue operation with reduced functionality if parts fail

## Future Enhancements

- **File Type Validation**: Enhanced file type detection for media files
- **Smart Processing**: Intelligent processing based on file types and library locations
- **Library-Specific Rules**: Different processing rules based on library type (staging vs. production vs. inbound)
- **Performance Metrics**: Detailed performance and usage metrics per library
- **Custom Hooks**: Support for custom actions on specific file events per library
- **Batch Processing**: Batch event processing for high-volume scenarios