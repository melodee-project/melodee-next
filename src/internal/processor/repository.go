package processor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"melodee/internal/models"
	"gorm.io/gorm"
)

// StagingRepository handles database operations for staging items
type StagingRepository struct {
	db *gorm.DB
}

// NewStagingRepository creates a new staging repository
func NewStagingRepository(db *gorm.DB) *StagingRepository {
	return &StagingRepository{db: db}
}

// CreateStagingItem creates a staging item record in PostgreSQL
func (r *StagingRepository) CreateStagingItem(item *models.StagingItem) error {
	return r.db.Create(item).Error
}

// CreateStagingItemFromResult creates a staging item from a process result
func (r *StagingRepository) CreateStagingItemFromResult(result ProcessResult, metadata *AlbumMetadata) error {
	// Calculate checksum
	checksum, err := calculateJSONChecksum(metadata)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	item := &models.StagingItem{
		ScanID:       metadata.ScanID,
		StagingPath:  result.StagingPath,
		MetadataFile: result.MetadataFile,
		ArtistName:   metadata.Artist.Name,
		AlbumName:    metadata.Album.Name,
		TrackCount:   int32(len(metadata.Tracks)),
		TotalSize:    result.TotalSize,
		ProcessedAt:  metadata.ProcessedAt,
		Status:       metadata.Status,
		Checksum:     checksum,
		CreatedAt:    time.Now(),
	}

	return r.CreateStagingItem(item)
}

// GetStagingItemsByStatus returns staging items with a specific status
func (r *StagingRepository) GetStagingItemsByStatus(status string) ([]models.StagingItem, error) {
	var items []models.StagingItem
	err := r.db.Where("status = ?", status).Find(&items).Error
	return items, err
}

// GetStagingItemsByScanID returns all staging items for a scan
func (r *StagingRepository) GetStagingItemsByScanID(scanID string) ([]models.StagingItem, error) {
	var items []models.StagingItem
	err := r.db.Where("scan_id = ?", scanID).Find(&items).Error
	return items, err
}

// UpdateStagingItemStatus updates the status of a staging item
func (r *StagingRepository) UpdateStagingItemStatus(id int64, status string, reviewedBy *int64, notes string) error {
	updates := map[string]interface{}{
		"status":      status,
		"reviewed_at": time.Now(),
	}
	
	if reviewedBy != nil {
		updates["reviewed_by"] = *reviewedBy
	}
	
	if notes != "" {
		updates["notes"] = notes
	}

	return r.db.Model(&models.StagingItem{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteStagingItem deletes a staging item
func (r *StagingRepository) DeleteStagingItem(id int64) error {
	return r.db.Delete(&models.StagingItem{}, id).Error
}

// GetPendingStagingItems returns items pending review
func (r *StagingRepository) GetPendingStagingItems() ([]models.StagingItem, error) {
	return r.GetStagingItemsByStatus("pending_review")
}

// GetApprovedStagingItems returns approved items ready for promotion
func (r *StagingRepository) GetApprovedStagingItems() ([]models.StagingItem, error) {
	return r.GetStagingItemsByStatus("approved")
}

// calculateJSONChecksum calculates SHA256 checksum of JSON data
func calculateJSONChecksum(data interface{}) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:]), nil
}
