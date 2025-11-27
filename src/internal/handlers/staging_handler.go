package handlers

import (
	"os"
	"strconv"

	"melodee/internal/models"
	"melodee/internal/processor"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// StagingHandler handles staging workflow endpoints
type StagingHandler struct {
	db          *gorm.DB
	repo        *processor.StagingRepository
	stagingRoot string
}

// NewStagingHandler creates a new staging handler
func NewStagingHandler(db *gorm.DB, stagingRoot string) *StagingHandler {
	return &StagingHandler{
		db:          db,
		repo:        processor.NewStagingRepository(db),
		stagingRoot: stagingRoot,
	}
}

// StagingItemResponse is the API response for a staging item
type StagingItemResponse struct {
	ID           int64   `json:"id"`
	ScanID       string  `json:"scan_id"`
	StagingPath  string  `json:"staging_path"`
	MetadataFile string  `json:"metadata_file"`
	ArtistName   string  `json:"artist_name"`
	AlbumName    string  `json:"album_name"`
	TrackCount   int32   `json:"track_count"`
	TotalSize    int64   `json:"total_size"`
	ProcessedAt  string  `json:"processed_at"`
	Status       string  `json:"status"`
	ReviewedBy   *int64  `json:"reviewed_by,omitempty"`
	ReviewedAt   *string `json:"reviewed_at,omitempty"`
	Notes        string  `json:"notes,omitempty"`
	Checksum     string  `json:"checksum"`
	CreatedAt    string  `json:"created_at"`
}

// ListStagingItems returns all staging items with optional filtering
// GET /api/v1/staging
func (h *StagingHandler) ListStagingItems(c *fiber.Ctx) error {
	status := c.Query("status", "")
	scanID := c.Query("scan_id", "")

	var items []models.StagingItem
	var err error

	query := h.db.Model(&models.StagingItem{})

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if scanID != "" {
		query = query.Where("scan_id = ?", scanID)
	}

	err = query.Order("created_at DESC").Find(&items).Error
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch staging items",
		})
	}

	// Convert to response format
	response := make([]StagingItemResponse, len(items))
	for i, item := range items {
		response[i] = toStagingItemResponse(&item)
	}

	return c.JSON(response)
}

// GetStagingItem returns a single staging item with metadata
// GET /api/v1/staging/:id
func (h *StagingHandler) GetStagingItem(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid staging item ID",
		})
	}

	var item models.StagingItem
	if err := h.db.First(&item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Staging item not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch staging item",
		})
	}

	// Read metadata file
	metadata, err := processor.ReadAlbumMetadata(item.MetadataFile)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to read metadata file",
		})
	}

	return c.JSON(fiber.Map{
		"item":     toStagingItemResponse(&item),
		"metadata": metadata,
	})
}

// ApproveStagingItem approves a staging item
// POST /api/v1/staging/:id/approve
func (h *StagingHandler) ApproveStagingItem(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid staging item ID",
		})
	}

	// Get user ID from context (set by auth middleware)
	userID := c.Locals("user_id").(int64)

	// Parse request body for optional notes
	var req struct {
		Notes string `json:"notes"`
	}
	if err := c.BodyParser(&req); err != nil {
		req.Notes = ""
	}

	// Update status
	if err := h.repo.UpdateStagingItemStatus(id, "approved", &userID, req.Notes); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to approve staging item",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Staging item approved",
	})
}

// RejectStagingItem rejects a staging item
// POST /api/v1/staging/:id/reject
func (h *StagingHandler) RejectStagingItem(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid staging item ID",
		})
	}

	// Get user ID from context
	userID := c.Locals("user_id").(int64)

	// Parse request body for required reason
	var req struct {
		Notes string `json:"notes"`
	}
	if err := c.BodyParser(&req); err != nil || req.Notes == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Rejection reason is required",
		})
	}

	// Update status
	if err := h.repo.UpdateStagingItemStatus(id, "rejected", &userID, req.Notes); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to reject staging item",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Staging item rejected",
	})
}

// GetStagingStats returns statistics about staging items
// GET /api/v1/staging/stats
func (h *StagingHandler) GetStagingStats(c *fiber.Ctx) error {
	var stats struct {
		Total          int64 `json:"total"`
		PendingReview  int64 `json:"pending_review"`
		Approved       int64 `json:"approved"`
		Rejected       int64 `json:"rejected"`
		TotalTracks    int64 `json:"total_tracks"`
		TotalSizeBytes int64 `json:"total_size_bytes"`
	}

	// Get counts by status
	h.db.Model(&models.StagingItem{}).Count(&stats.Total)
	h.db.Model(&models.StagingItem{}).Where("status = ?", "pending_review").Count(&stats.PendingReview)
	h.db.Model(&models.StagingItem{}).Where("status = ?", "approved").Count(&stats.Approved)
	h.db.Model(&models.StagingItem{}).Where("status = ?", "rejected").Count(&stats.Rejected)

	// Get total tracks and size
	h.db.Model(&models.StagingItem{}).
		Select("COALESCE(SUM(track_count), 0)").
		Where("status != ?", "rejected").
		Scan(&stats.TotalTracks)

	h.db.Model(&models.StagingItem{}).
		Select("COALESCE(SUM(total_size), 0)").
		Where("status != ?", "rejected").
		Scan(&stats.TotalSizeBytes)

	return c.JSON(stats)
}

// DeleteStagingItem deletes a rejected staging item
// DELETE /api/v1/staging/:id
func (h *StagingHandler) DeleteStagingItem(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid staging item ID",
		})
	}

	// Get the item first to check status and get path
	var item models.StagingItem
	if err := h.db.First(&item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Staging item not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch staging item",
		})
	}

	// Only allow deletion of rejected items
	if item.Status != "rejected" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Can only delete rejected items",
		})
	}

	// Delete from database
	if err := h.repo.DeleteStagingItem(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete staging item",
		})
	}

	// Optionally delete the staging directory
	deleteFiles := c.QueryBool("delete_files", false)
	if deleteFiles {
		if err := os.RemoveAll(item.StagingPath); err != nil {
			// Log error but don't fail the request
			// The database record is already deleted
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Staging item deleted",
	})
}

// toStagingItemResponse converts a model to response format
func toStagingItemResponse(item *models.StagingItem) StagingItemResponse {
	resp := StagingItemResponse{
		ID:           item.ID,
		ScanID:       item.ScanID,
		StagingPath:  item.StagingPath,
		MetadataFile: item.MetadataFile,
		ArtistName:   item.ArtistName,
		AlbumName:    item.AlbumName,
		TrackCount:   item.TrackCount,
		TotalSize:    item.TotalSize,
		ProcessedAt:  item.ProcessedAt.Format("2006-01-02T15:04:05Z07:00"),
		Status:       item.Status,
		ReviewedBy:   item.ReviewedBy,
		Notes:        item.Notes,
		Checksum:     item.Checksum,
		CreatedAt:    item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if item.ReviewedAt != nil {
		reviewedAt := item.ReviewedAt.Format("2006-01-02T15:04:05Z07:00")
		resp.ReviewedAt = &reviewedAt
	}

	return resp
}
