package media

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
)

// QuarantineReason represents the reason for quarantining a file
type QuarantineReason string

const (
	ChecksumMismatch     QuarantineReason = "checksum_mismatch"
	TagParseError        QuarantineReason = "tag_parse_error"
	UnsupportedContainer QuarantineReason = "unsupported_container"
	FFmpegFailure        QuarantineReason = "ffmpeg_failure"
	PathSafety           QuarantineReason = "path_safety"
	ValidationBounds     QuarantineReason = "validation_bounds"
	MetadataConflict     QuarantineReason = "metadata_conflict"
	DiskFull             QuarantineReason = "disk_full"
	CueMissingAudio      QuarantineReason = "cue_missing_audio"
)

// QuarantineRecord represents a quarantined item in the database
type QuarantineRecord struct {
	ID           int64            `gorm:"primaryKey;autoIncrement" json:"id"`
	FilePath     string           `gorm:"not null" json:"file_path"`
	OriginalPath string           `gorm:"not null" json:"original_path"`
	Reason       QuarantineReason `gorm:"not null" json:"reason"`
	Message      string           `json:"message"`
	LibraryID    int32            `json:"library_id"`
	CreatedAt    time.Time        `json:"created_at"`
}

// QuarantineService handles quarantining of problematic files
type QuarantineService struct {
	db            *gorm.DB
	quarantineDir string
}

// NewQuarantineService creates a new quarantine service with custom directory
func NewQuarantineService(db *gorm.DB, quarantineDir string) *QuarantineService {
	return &QuarantineService{
		db:            db,
		quarantineDir: quarantineDir,
	}
}

// NewDefaultQuarantineService creates a new quarantine service with default directory
func NewDefaultQuarantineService(db *gorm.DB) *QuarantineService {
	return &QuarantineService{
		db:            db,
		quarantineDir: "/melodee/quarantine", // Default quarantine directory
	}
}

// QuarantineFile quarantines a problematic file and records the reason
func (qs *QuarantineService) QuarantineFile(filePath string, reason QuarantineReason, message string, libraryID int32) error {
	// Create quarantine directory structure by date and reason
	quarantinePath := filepath.Join(
		qs.quarantineDir,
		string(reason),
		time.Now().Format("2006-01-02"),
	)

	// Ensure the quarantine directory exists
	if err := os.MkdirAll(quarantinePath, 0755); err != nil {
		return fmt.Errorf("failed to create quarantine directory: %w", err)
	}

	// Move the file to quarantine
	quarantinedFilePath := filepath.Join(quarantinePath, filepath.Base(filePath))
	if err := os.Rename(filePath, quarantinedFilePath); err != nil {
		return fmt.Errorf("failed to move file to quarantine: %w", err)
	}

	// Create a quarantine record in the database
	quarantineRecord := &QuarantineRecord{
		FilePath:     quarantinedFilePath,
		OriginalPath: filePath,
		Reason:       reason,
		Message:      message,
		LibraryID:    libraryID,
		CreatedAt:    time.Now(),
	}

	if err := qs.db.Create(quarantineRecord).Error; err != nil {
		return fmt.Errorf("failed to create quarantine record: %w", err)
	}

	return nil
}

// ListQuarantinedFiles returns all quarantined files with optional filtering
func (qs *QuarantineService) ListQuarantinedFiles(limit, offset int, reason *QuarantineReason) ([]QuarantineRecord, int64, error) {
	var records []QuarantineRecord
	var total int64

	query := qs.db.Model(&QuarantineRecord{})

	if reason != nil {
		query = query.Where("reason = ?", *reason)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count quarantined files: %w", err)
	}

	// Get records with pagination
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch quarantined files: %w", err)
	}

	return records, total, nil
}

// DeleteFromQuarantine permanently removes a quarantined file
func (qs *QuarantineService) DeleteFromQuarantine(quarantineID int64) error {
	var record QuarantineRecord
	if err := qs.db.First(&record, quarantineID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("quarantine record not found: %d", quarantineID)
		}
		return fmt.Errorf("failed to find quarantine record: %w", err)
	}

	// Delete the file from the filesystem
	if err := os.Remove(record.FilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove quarantined file: %w", err)
	}

	// Delete the record from database
	if err := qs.db.Delete(&QuarantineRecord{}, quarantineID).Error; err != nil {
		return fmt.Errorf("failed to delete quarantine record: %w", err)
	}

	return nil
}

// RestoreFromQuarantine attempts to restore a quarantined file to its original location
func (qs *QuarantineService) RestoreFromQuarantine(quarantineID int64, targetDir string) error {
	var record QuarantineRecord
	if err := qs.db.First(&record, quarantineID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("quarantine record not found: %d", quarantineID)
		}
		return fmt.Errorf("failed to find quarantine record: %w", err)
	}

	// Determine the target path (original or a specified directory)
	targetPath := record.OriginalPath
	if targetDir != "" {
		targetPath = filepath.Join(targetDir, filepath.Base(record.OriginalPath))
	}

	// Move the file back from quarantine to its target location
	if err := os.Rename(record.FilePath, targetPath); err != nil {
		return fmt.Errorf("failed to restore quarantined file: %w", err)
	}

	// Delete the quarantine record from database
	if err := qs.db.Delete(&QuarantineRecord{}, quarantineID).Error; err != nil {
		return fmt.Errorf("failed to delete quarantine record: %w", err)
	}

	return nil
}

// ProcessQuarantineCleanup cleans up old quarantined files based on retention policy
func (qs *QuarantineService) ProcessQuarantineCleanup(maxAgeDays int) error {
	cutoffDate := time.Now().AddDate(0, 0, -maxAgeDays)

	var records []QuarantineRecord
	if err := qs.db.Where("created_at < ?", cutoffDate).Find(&records).Error; err != nil {
		return fmt.Errorf("failed to find old quarantine records: %w", err)
	}

	for _, record := range records {
		// Delete the file
		if err := os.Remove(record.FilePath); err != nil && !os.IsNotExist(err) {
			// Log the error but continue with other files
			fmt.Printf("Failed to remove old quarantine file %s: %v\n", record.FilePath, err)
			continue
		}

		// Delete the record from database
		if err := qs.db.Delete(&QuarantineRecord{}, record.ID).Error; err != nil {
			// Log the error but continue with other records
			fmt.Printf("Failed to remove old quarantine record %d: %v\n", record.ID, err)
			continue
		}
	}

	return nil
}

// GetQuarantineStats returns statistics about quarantined files
func (qs *QuarantineService) GetQuarantineStats() (map[QuarantineReason]int64, error) {
	var results []struct {
		Reason QuarantineReason `json:"reason"`
		Count  int64            `json:"count"`
	}

	// Count files by reason
	if err := qs.db.Model(&QuarantineRecord{}).
		Select("reason, COUNT(*) as count").
		Group("reason").
		Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get quarantine stats: %w", err)
	}

	// Convert to map
	stats := make(map[QuarantineReason]int64)
	for _, result := range results {
		stats[result.Reason] = result.Count
	}

	return stats, nil
}

// ValidateFilePath validates that a file path is safe and doesn't contain traversal attempts
func (qs *QuarantineService) ValidateFilePath(path string) error {
	// Check for path traversal attempts
	if containsPathTraversal(path) {
		return fmt.Errorf("path traversal detected in path: %s", path)
	}

	// Additional validation can be added here

	return nil
}

// containsPathTraversal checks if a path contains potential traversal attempts
func containsPathTraversal(path string) bool {
	// Check for common path traversal patterns
	return (filepath.Clean(path) != path) ||
		(strings.Contains(path, "../")) ||
		(strings.Contains(path, "..\\")) ||
		(strings.Contains(path, "/..")) ||
		(strings.Contains(path, "\\.."))
}
