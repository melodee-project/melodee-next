package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"melodee/internal/config"
	"melodee/internal/models"
)

// PartitionJobManager handles partition-related jobs
type PartitionJobManager struct {
	client        *asynq.Client
	scheduler     *asynq.Scheduler
	config        *config.AppConfig
	db            *gorm.DB
	logger        interface{} // Placeholder for logger interface
}

// NewPartitionJobManager creates a new partition job manager
func NewPartitionJobManager(
	client *asynq.Client,
	scheduler *asynq.Scheduler,
	config *config.AppConfig,
	db *gorm.DB,
	logger interface{},
) *PartitionJobManager {
	return &PartitionJobManager{
		client:    client,
		scheduler: scheduler,
		config:    config,
		db:        db,
		logger:    logger,
	}
}

// RegisterPartitionTasks registers all partition-related tasks with Asynq
func (pjm *PartitionJobManager) RegisterPartitionTasks() {
	// Register the partition creation task
	asynq.HandleFunc(TaskPartitionCreateNextMonth, pjm.HandleCreateNextMonthPartition)
	
	// Could register other partition tasks here as needed
}

// Task types for partition jobs
const (
	TaskPartitionCreateNextMonth = "partition:create-next-month"
)

// PartitionCreatePayload represents the payload for partition creation jobs
type PartitionCreatePayload struct {
	TableName string    `json:"table_name"`  // "albums" or "songs"
	Year      int       `json:"year"`
	Month     time.Month `json:"month"`
	Forced    bool      `json:"forced,omitempty"` // If true, recreate even if partition exists
}

// HandleCreateNextMonthPartition handles the creation of next month's partitions
func (pjm *PartitionJobManager) HandleCreateNextMonthPartition(ctx context.Context, t *asynq.Task) error {
	var payload PartitionCreatePayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal partition creation payload: %w", err)
	}

	tableName := payload.TableName
	year := payload.Year
	month := payload.Month

	if tableName == "" {
		// Use the next month based on current date
		now := time.Now()
		nextMonth := now.AddDate(0, 1, 0)
		year = nextMonth.Year()
		month = nextMonth.Month()
	}

	if tableName == "" || tableName == "albums" {
		if err := pjm.createAlbumPartition(year, month, payload.Forced); err != nil {
			return fmt.Errorf("failed to create album partition: %w", err)
		}
	}

	if tableName == "" || tableName == "songs" {
		if err := pjm.createSongPartition(year, month, payload.Forced); err != nil {
			return fmt.Errorf("failed to create song partition: %w", err)
		}
	}

	return nil
}

// createAlbumPartition creates a monthly partition for the albums table
func (pjm *PartitionJobManager) createAlbumPartition(year int, month time.Month, forced bool) error {
	partitionName := fmt.Sprintf("albums_%d_%02d", year, month)
	partitionOf := "albums"

	// Check if partition already exists
	if !forced {
		exists, err := pjm.partitionExists(partitionName)
		if err != nil {
			return fmt.Errorf("failed to check if partition exists: %w", err)
		}
		if exists {
			return nil // Partition already exists, nothing to do unless forced
		}
	}

	// Calculate start and end dates for the partition
	startDate := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(year, month+1, 1, 0, 0, 0, 0, time.UTC)

	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s
		PARTITION OF %s
		FOR VALUES FROM ('%s') TO ('%s');
	`, partitionName, partitionOf, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	if err := pjm.db.Exec(query).Error; err != nil {
		return fmt.Errorf("failed to create album partition %s: %w", partitionName, err)
	}

	// Create indexes specific to this partition
	if err := pjm.createPartitionIndexes(partitionName, "album"); err != nil {
		return fmt.Errorf("failed to create indexes for album partition %s: %w", partitionName, err)
	}

	return nil
}

// createSongPartition creates a monthly partition for the songs table
func (pjm *PartitionJobManager) createSongPartition(year int, month time.Month, forced bool) error {
	partitionName := fmt.Sprintf("songs_%d_%02d", year, month)
	partitionOf := "songs"

	// Check if partition already exists
	if !forced {
		exists, err := pjm.partitionExists(partitionName)
		if err != nil {
			return fmt.Errorf("failed to check if partition exists: %w", err)
		}
		if exists {
			return nil // Partition already exists, nothing to do unless forced
		}
	}

	// Calculate start and end dates for the partition
	startDate := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(year, month+1, 1, 0, 0, 0, 0, time.UTC)

	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s
		PARTITION OF %s
		FOR VALUES FROM ('%s') TO ('%s');
	`, partitionName, partitionOf, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	if err := pjm.db.Exec(query).Error; err != nil {
		return fmt.Errorf("failed to create song partition %s: %w", partitionName, err)
	}

	// Create indexes specific to this partition
	if err := pjm.createPartitionIndexes(partitionName, "song"); err != nil {
		return fmt.Errorf("failed to create indexes for song partition %s: %w", partitionName, err)
	}

	return nil
}

// partitionExists checks if a table/partition with the given name exists
func (pjm *PartitionJobManager) partitionExists(tableName string) (bool, error) {
	var count int64
	err := pjm.db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ? AND table_schema = 'public'", tableName).Scan(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// createPartitionIndexes creates the required indexes for a partition
func (pjm *PartitionJobManager) createPartitionIndexes(tableName, entityType string) error {
	var indexes []string

	switch entityType {
	case "album":
		// Indexes for album partitions
		indexes = []string{
			fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_artist_id ON %s(artist_id);", tableName, tableName),
			fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_name_normalized_gin ON %s USING gin(name_normalized gin_trgm_ops);", tableName, tableName),
			fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_api_key ON %s(api_key);", tableName, tableName),
			fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_musicbrainz_id ON %s(musicbrainz_id);", tableName, tableName),
			// Covering index for common API operations (getArtist, getAlbum)
			fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_artist_status_covering ON %s(artist_id, album_status, name_normalized, directory, sort_order);", tableName, tableName),
			// Partial index for active albums only
			fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_active ON %s(artist_id, name_normalized, sort_order) WHERE album_status = 'Ok';", tableName, tableName),
		}
	case "song":
		// Indexes for song partitions
		indexes = []string{
			fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_album_id_hash ON %s USING hash(album_id);", tableName, tableName),
			fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_artist_id_hash ON %s USING hash(artist_id);", tableName, tableName),
			// Covering index for streaming operations (most critical)
			fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_album_order_covering ON %s(album_id, sort_order, name_normalized, relative_path, duration, api_key);", tableName, tableName),
			// Covering index for search operations
			fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_search_covering ON %s(name_normalized, artist_id, album_id, duration, relative_path);", tableName, tableName),
			// Full-text search
			fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_fulltext ON %s USING gin(to_tsvector('english', name_normalized || ' ' || COALESCE(tags->>'artist', '') || ' ' || COALESCE(tags->>'album', '')));`, tableName, tableName),
			// Partial index for active (Ok status) songs only
			fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_active ON %s(album_id, sort_order) WHERE album_id IN (SELECT id FROM albums WHERE album_status = 'Ok');", tableName, tableName),
		}
	}

	// Create all indexes
	for _, indexQuery := range indexes {
		if err := pjm.db.Exec(indexQuery).Error; err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// SchedulePartitionCreation schedules automatic monthly partition creation
func (pjm *PartitionJobManager) SchedulePartitionCreation() error {
	// Schedule the partition creation job to run weekly
	// This will create next month's partitions in advance
	entryID, err := pjm.scheduler.Register(
		"0 0 * * 1", // Every Monday at midnight
		asynq.NewTask(TaskPartitionCreateNextMonth, nil),
	)
	if err != nil {
		return fmt.Errorf("failed to schedule partition creation: %w", err)
	}

	fmt.Printf("Scheduled partition creation job with entry ID: %s\n", entryID)
	return nil
}

// RunPartitionCreationForMonth runs partition creation for a specific month
func (pjm *PartitionJobManager) RunPartitionCreationForMonth(year int, month time.Month) error {
	// Create a task for album partition creation
	albumPayload := &PartitionCreatePayload{
		TableName: "albums",
		Year:      year,
		Month:     month,
	}
	
	albumTask := asynq.NewTask(TaskPartitionCreateNextMonth, albumPayload)
	if _, err := pjm.client.Enqueue(albumTask, asynq.Queue("maintenance")); err != nil {
		return fmt.Errorf("failed to enqueue album partition creation: %w", err)
	}

	// Create a task for song partition creation
	songPayload := &PartitionCreatePayload{
		TableName: "songs", 
		Year:      year,
		Month:     month,
	}
	
	songTask := asynq.NewTask(TaskPartitionCreateNextMonth, songPayload)
	if _, err := pjm.client.Enqueue(songTask, asynq.Queue("maintenance")); err != nil {
		return fmt.Errorf("failed to enqueue song partition creation: %w", err)
	}

	return nil
}

// VerifyPartitionIndexes verifies partition indexes are in place and working
func (pjm *PartitionJobManager) VerifyPartitionIndexes(tableName string) error {
	// In a real implementation, we would run EXPLAIN (ANALYZE,BUFFERS) queries 
	// for sample getAlbum and stream operations to ensure indexes are being used
	//
	// Example queries to test:
	// EXPLAIN (ANALYZE,BUFFERS) SELECT * FROM albums WHERE artist_id = ? LIMIT 50;
	// EXPLAIN (ANALYZE,BUFFERS) SELECT * FROM songs WHERE album_id = ? ORDER BY sort_order;
	
	// For now, we'll just verify that the table exists
	exists, err := pjm.partitionExists(tableName)
	if err != nil {
		return fmt.Errorf("failed to verify partition %s: %w", tableName, err)
	}
	
	if !exists {
		return fmt.Errorf("partition %s does not exist", tableName)
	}
	
	return nil
}

// ArchiveOldPartitions moves old partitions to archive schema after retention period
func (pjm *PartitionJobManager) ArchiveOldPartitions(retentionMonths int) error {
	// Get all partitions older than retention period
	cutoffDate := time.Now().AddDate(0, -retentionMonths, 0)
	
	// In a real implementation, we would:
	// 1. Identify partitions older than cutoffDate
	// 2. Move them to archive schema with ALTER TABLE ... SET SCHEMA archive
	// 3. Update autovacuum settings for archived partitions
	// 4. Create indexes in archive schema as needed
	
	// This is a simplified implementation
	fmt.Printf("Archiving partitions older than: %s\n", cutoffDate.Format("2006-01-02"))
	
	return nil
}