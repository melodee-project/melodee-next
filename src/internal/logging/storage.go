package logging

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// LogEntry represents a stored log entry in the database
type LogEntry struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Timestamp time.Time `gorm:"index;not null" json:"timestamp"`
	Level     string    `gorm:"size:10;index;not null" json:"level"`
	Message   string    `gorm:"type:text;not null" json:"message"`
	Module    string    `gorm:"size:100;index" json:"module,omitempty"`
	Function  string    `gorm:"size:100" json:"function,omitempty"`
	RequestID string    `gorm:"size:50;index" json:"request_id,omitempty"`
	UserID    int64     `gorm:"index" json:"user_id,omitempty"`
	TraceID   string    `gorm:"size:50;index" json:"trace_id,omitempty"`
	IP        string    `gorm:"size:50" json:"ip,omitempty"`
	Route     string    `gorm:"size:255" json:"route,omitempty"`
	Status    int       `json:"status,omitempty"`
	Duration  int64     `json:"duration_ms,omitempty"` // Duration in milliseconds
	Queue     string    `gorm:"size:50" json:"queue,omitempty"`
	JobType   string    `gorm:"size:100" json:"job_type,omitempty"`
	LibraryID int32     `json:"library_id,omitempty"`
	FilePath  string    `gorm:"size:1024" json:"file_path,omitempty"`
	Error     string    `gorm:"type:text" json:"error,omitempty"`
	Stack     string    `gorm:"type:text" json:"stack,omitempty"`
	Metadata  string    `gorm:"type:jsonb" json:"metadata,omitempty"` // Additional JSON metadata
}

// TableName specifies the table name for LogEntry
func (LogEntry) TableName() string {
	return "log_entries"
}

// LogStorage handles persistent storage of logs
type LogStorage struct {
	db *gorm.DB
}

// NewLogStorage creates a new log storage instance
func NewLogStorage(db *gorm.DB) *LogStorage {
	return &LogStorage{db: db}
}

// Store saves a log entry to the database
func (s *LogStorage) Store(ctx context.Context, entry *LogEntry) error {
	return s.db.WithContext(ctx).Create(entry).Error
}

// Query retrieves log entries based on filters
func (s *LogStorage) Query(ctx context.Context, filters LogFilters) ([]LogEntry, int64, error) {
	query := s.db.WithContext(ctx).Model(&LogEntry{})

	// Apply filters
	if filters.Level != "" {
		query = query.Where("level = ?", filters.Level)
	}
	if filters.Module != "" {
		query = query.Where("module = ?", filters.Module)
	}
	if filters.UserID != 0 {
		query = query.Where("user_id = ?", filters.UserID)
	}
	if filters.RequestID != "" {
		query = query.Where("request_id = ?", filters.RequestID)
	}
	if filters.LibraryID != 0 {
		query = query.Where("library_id = ?", filters.LibraryID)
	}
	if filters.JobType != "" {
		query = query.Where("job_type = ?", filters.JobType)
	}
	if !filters.StartTime.IsZero() {
		query = query.Where("timestamp >= ?", filters.StartTime)
	}
	if !filters.EndTime.IsZero() {
		query = query.Where("timestamp <= ?", filters.EndTime)
	}
	if filters.Search != "" {
		searchPattern := fmt.Sprintf("%%%s%%", filters.Search)
		query = query.Where("message ILIKE ? OR error ILIKE ?", searchPattern, searchPattern)
	}

	// Count total before pagination
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply ordering and pagination
	var entries []LogEntry
	err := query.
		Order("timestamp DESC").
		Offset(filters.Offset).
		Limit(filters.Limit).
		Find(&entries).Error

	return entries, total, err
}

// GetRecent retrieves the most recent log entries
func (s *LogStorage) GetRecent(ctx context.Context, limit int) ([]LogEntry, error) {
	var entries []LogEntry
	err := s.db.WithContext(ctx).
		Model(&LogEntry{}).
		Order("timestamp DESC").
		Limit(limit).
		Find(&entries).Error
	return entries, err
}

// DeleteOldLogs removes log entries older than the specified duration
func (s *LogStorage) DeleteOldLogs(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result := s.db.WithContext(ctx).
		Where("timestamp < ?", cutoff).
		Delete(&LogEntry{})
	return result.RowsAffected, result.Error
}

// GetLogStats returns statistics about stored logs
func (s *LogStorage) GetLogStats(ctx context.Context, since time.Time) (*LogStats, error) {
	var stats LogStats

	// Count by level
	err := s.db.WithContext(ctx).
		Model(&LogEntry{}).
		Select("level, COUNT(*) as count").
		Where("timestamp >= ?", since).
		Group("level").
		Scan(&stats.ByLevel).Error
	if err != nil {
		return nil, err
	}

	// Count errors
	err = s.db.WithContext(ctx).
		Model(&LogEntry{}).
		Where("level IN ('error', 'fatal') AND timestamp >= ?", since).
		Count(&stats.ErrorCount).Error
	if err != nil {
		return nil, err
	}

	// Count warnings
	err = s.db.WithContext(ctx).
		Model(&LogEntry{}).
		Where("level = 'warn' AND timestamp >= ?", since).
		Count(&stats.WarnCount).Error
	if err != nil {
		return nil, err
	}

	stats.Since = since
	stats.GeneratedAt = time.Now()

	return &stats, nil
}

// LogFilters defines filters for querying logs
type LogFilters struct {
	Level     string
	Module    string
	UserID    int64
	RequestID string
	LibraryID int32
	JobType   string
	StartTime time.Time
	EndTime   time.Time
	Search    string
	Offset    int
	Limit     int
}

// LogStats represents statistics about logs
type LogStats struct {
	ByLevel     []LevelCount `json:"by_level"`
	ErrorCount  int64        `json:"error_count"`
	WarnCount   int64        `json:"warn_count"`
	Since       time.Time    `json:"since"`
	GeneratedAt time.Time    `json:"generated_at"`
}

// LevelCount represents count per log level
type LevelCount struct {
	Level string `json:"level"`
	Count int64  `json:"count"`
}
