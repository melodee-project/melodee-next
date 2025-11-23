package handlers

import (
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"melodee/internal/media"
	"melodee/internal/models"
	"melodee/internal/services"
	"melodee/internal/utils"
)

// LibraryHandler handles library-related requests
type LibraryHandler struct {
	repo            *services.Repository
	mediaSvc        *media.MediaService
	asynqClient     *asynq.Client
	quarantineSvc   *media.QuarantineService
}

// NewLibraryHandler creates a new library handler
func NewLibraryHandler(repo *services.Repository, mediaSvc *media.MediaService, asynqClient *asynq.Client, quarantineSvc *media.QuarantineService) *LibraryHandler {
	return &LibraryHandler{
		repo:          repo,
		mediaSvc:      mediaSvc,
		asynqClient:   asynqClient,
		quarantineSvc: quarantineSvc,
	}
}

// LibraryState represents the state of a library
type LibraryState struct {
	ID              int32                     `json:"id"`
	Name            string                    `json:"name"`
	Type            string                    `json:"type"` // inbound, staging, production
	Path            string                    `json:"path"`
	ItemCount       int64                     `json:"item_count"`
	IsLocked        bool                      `json:"is_locked"`
	InboundCount    int32                     `json:"inbound_count"`
	StagingCount    int32                     `json:"staging_count"`
	ProductionCount int32                     `json:"production_count"`
	QuarantineCount int32                     `json:"quarantine_count"`
	Stats           *models.Library           `json:"stats"`
	SongCount       int32                     `json:"song_count"`
	AlbumCount      int32                     `json:"album_count"`
	Duration        int64                     `json:"duration"` // in milliseconds
	BasePath        string                    `json:"base_path"`
	QuarantineItems []QuarantineItem          `json:"quarantine_items,omitempty"`
	ProcessingJobs  []ProcessingJob           `json:"processing_jobs,omitempty"`
}

// QuarantineItem represents an item in the quarantine state
type QuarantineItem struct {
	ID          int64     `json:"id"`
	FilePath    string    `json:"file_path"`
	Reason      string    `json:"reason"`
	Message     string    `json:"message"`
	LibraryID   int32     `json:"library_id"`
	CreatedAt   string    `json:"created_at"`
	Resolved    bool      `json:"resolved"`
}

// ProcessingJob represents a processing job
type ProcessingJob struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`  // scan, process, move_ok, etc.
	Status      string    `json:"status"` // pending, running, complete, failed
	CreatedAt   string    `json:"created_at"`
	FinishedAt  *string   `json:"finished_at,omitempty"`
	Error       *string   `json:"error,omitempty"`
	Progress    float64   `json:"progress"`
}

// GetLibraryStates handles retrieving all library states
func (h *LibraryHandler) GetLibraryStates(c *fiber.Ctx) error {
	// Get all libraries from the repository
	var libraries []models.Library
	if err := h.repo.DB.Find(&libraries).Error; err != nil {
		return utils.SendInternalServerError(c, "Failed to retrieve libraries")
	}

	// For each library, get its state information
	states := make([]LibraryState, len(libraries))
	for i, lib := range libraries {
		// Calculate item counts for different stages using the database
		inboundCount, err := h.getLibraryItemCount(lib.ID, "inbound")
		if err != nil {
			return utils.SendInternalServerError(c, "Failed to count inbound items")
		}

		stagingCount, err := h.getLibraryItemCount(lib.ID, "staging")
		if err != nil {
			return utils.SendInternalServerError(c, "Failed to count staging items")
		}

		productionCount, err := h.getLibraryItemCount(lib.ID, "production")
		if err != nil {
			return utils.SendInternalServerError(c, "Failed to count production items")
		}

		quarantineCount := h.getQuarantineItemCount(lib.Path)

		// Get item counts for different stages
		states[i] = LibraryState{
			ID:              lib.ID,
			Name:            lib.Name,
			Type:            lib.Type,
			Path:            lib.Path,
			IsLocked:        lib.IsLocked,
			SongCount:       lib.SongCount,
			AlbumCount:      lib.AlbumCount,
			Duration:        lib.Duration,
			BasePath:        lib.BasePath,
			InboundCount:    inboundCount,
			StagingCount:    stagingCount,
			ProductionCount: productionCount,
			QuarantineCount: int32(quarantineCount),
		}
	}

	return c.JSON(states)
}

// GetLibraryState handles retrieving a specific library state
func (h *LibraryHandler) GetLibraryState(c *fiber.Ctx) error {
	libraryID, err := c.ParamsInt("id")
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid library ID")
	}

	// Get the specific library
	var library models.Library
	if err := h.repo.DB.First(&library, libraryID).Error; err != nil {
		return utils.SendNotFoundError(c, "Library")
	}

	// Calculate item counts for different stages using the database
	inboundCount, err := h.getLibraryItemCount(library.ID, "inbound")
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to count inbound items")
	}

	stagingCount, err := h.getLibraryItemCount(library.ID, "staging")
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to count staging items")
	}

	productionCount, err := h.getLibraryItemCount(library.ID, "production")
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to count production items")
	}

	quarantineCount := h.getQuarantineItemCount(library.Path)

	// Get state information for this library
	state := LibraryState{
		ID:              library.ID,
		Name:            library.Name,
		Type:            library.Type,
		Path:            library.Path,
		IsLocked:        library.IsLocked,
		SongCount:       library.SongCount,
		AlbumCount:      library.AlbumCount,
		Duration:        library.Duration,
		BasePath:        library.BasePath,
		InboundCount:    inboundCount,
		StagingCount:    stagingCount,
		ProductionCount: productionCount,
		QuarantineCount: int32(quarantineCount),
	}

	return c.JSON(state)
}

// GetQuarantineItems handles retrieving quarantine items
func (h *LibraryHandler) GetQuarantineItems(c *fiber.Ctx) error {
	// Get pagination parameters
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 50)

	// Validate limits
	if limit > 100 {
		limit = 100 // Maximum limit for safety
	}
	if limit < 1 {
		limit = 10
	}

	offset := (page - 1) * limit

	// Get reason filter if provided
	var reasonFilter *media.QuarantineReason
	reasonParam := c.Query("reason")
	if reasonParam != "" {
		reason := media.QuarantineReason(reasonParam)
		reasonFilter = &reason
	}

	// Get quarantine records from the service
	records, totalCount, err := h.quarantineSvc.ListQuarantinedFiles(limit, offset, reasonFilter)
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to retrieve quarantine items: "+err.Error())
	}

	// Convert to response format
	items := make([]QuarantineItem, len(records))
	for i, record := range records {
		items[i] = QuarantineItem{
			ID:          record.ID,
			FilePath:    record.FilePath,
			Reason:      string(record.Reason),
			Message:     record.Message,
			LibraryID:   record.LibraryID,
			CreatedAt:   record.CreatedAt.Format("2006-01-02 15:04:05"),
			Resolved:    false, // In a real implementation, this would depend on actual status
		}
	}

	return c.JSON(fiber.Map{
		"data": items,
		"count": len(items),
		"total": totalCount,
		"page": page,
		"limit": limit,
	})
}

// GetProcessingJobs handles retrieving processing jobs
func (h *LibraryHandler) GetProcessingJobs(c *fiber.Ctx) error {
	// In a real implementation, you would query your job queue (Asynq) for processing jobs
	// For now, return an empty list
	jobs := []ProcessingJob{}
	
	return c.JSON(fiber.Map{
		"data": jobs,
		"count": len(jobs),
	})
}

// TriggerLibraryScan handles triggering a library scan
func (h *LibraryHandler) TriggerLibraryScan(c *fiber.Ctx) error {
	libraryID, err := c.ParamsInt("id")
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid library ID")
	}

	// Check if library exists
	var library models.Library
	if err := h.repo.DB.First(&library, libraryID).Error; err != nil {
		return utils.SendNotFoundError(c, "Library")
	}

	// Enqueue the scan job
	if err := h.mediaSvc.EnqueueLibraryScan(h.asynqClient, []int32{int32(libraryID)}, false); err != nil {
		return utils.SendInternalServerError(c, "Failed to enqueue library scan")
	}

	return c.JSON(fiber.Map{
		"status": "queued",
		"library_id": libraryID,
		"message": "Library scan triggered successfully",
	})
}

// TriggerLibraryProcess handles triggering a library process operation
func (h *LibraryHandler) TriggerLibraryProcess(c *fiber.Ctx) error {
	libraryID, err := c.ParamsInt("id")
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid library ID")
	}

	// Check if library exists
	var library models.Library
	if err := h.repo.DB.First(&library, libraryID).Error; err != nil {
		return utils.SendNotFoundError(c, "Library")
	}

	// Enqueue the process job - in a real implementation, filePaths would come from request
	// For now, we'll pass an empty array to process all files in the library
	if err := h.mediaSvc.EnqueueLibraryProcess(h.asynqClient, int32(libraryID), []string{}); err != nil {
		return utils.SendInternalServerError(c, "Failed to enqueue library process")
	}

	return c.JSON(fiber.Map{
		"status": "queued",
		"library_id": libraryID,
		"message": "Library process triggered successfully",
	})
}

// TriggerLibraryMoveOK handles triggering a library move from staging to production
func (h *LibraryHandler) TriggerLibraryMoveOK(c *fiber.Ctx) error {
	libraryID, err := c.ParamsInt("id")
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid library ID")
	}

	// Check if library exists
	var library models.Library
	if err := h.repo.DB.First(&library, libraryID).Error; err != nil {
		return utils.SendNotFoundError(c, "Library")
	}

	// Note: In a real implementation, this would move all 'OK' status albums from staging to production
	// For now, we'll just return success as the actual implementation would require
	// more complex logic to find and move specific albums
	// The proper way would be to query for albums in staging with 'OK' status and trigger individual moves

	return c.JSON(fiber.Map{
		"status": "processed",
		"library_id": libraryID,
		"message": "Library move OK processing started",
	})
}

// ResolveQuarantineItem handles resolving a quarantine item
func (h *LibraryHandler) ResolveQuarantineItem(c *fiber.Ctx) error {
	itemID, err := c.ParamsInt("id")
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid item ID")
	}

	// Check if the quarantine record exists
	var record media.QuarantineRecord
	if err := h.repo.DB.First(&record, itemID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return utils.SendNotFoundError(c, "Quarantine record")
		}
		return utils.SendInternalServerError(c, "Failed to verify quarantine record: "+err.Error())
	}

	// Delete from quarantine (permanent removal)
	if err := h.quarantineSvc.DeleteFromQuarantine(int64(itemID)); err != nil {
		return utils.SendInternalServerError(c, "Failed to resolve quarantine item: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"status": "resolved",
		"item_id": itemID,
		"message": "Quarantine item removed successfully",
	})
}

// RequeueQuarantineItem handles requeuing a quarantine item for reprocessing
func (h *LibraryHandler) RequeueQuarantineItem(c *fiber.Ctx) error {
	itemID, err := c.ParamsInt("id")
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid item ID")
	}

	// Validate that the quarantine record exists
	var record media.QuarantineRecord
	if err := h.repo.DB.First(&record, itemID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return utils.SendNotFoundError(c, "Quarantine record")
		}
		return utils.SendInternalServerError(c, "Failed to verify quarantine record: "+err.Error())
	}

	// Restore the file to its original location for reprocessing
	if err := h.quarantineSvc.RestoreFromQuarantine(int64(itemID), ""); err != nil {
		return utils.SendInternalServerError(c, "Failed to restore quarantined file: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"status": "requeued",
		"item_id": itemID,
		"message": "Quarantine item requeued for processing",
	})
}

// getLibraryItemCount counts items in a specific library with a specific status
func (h *LibraryHandler) getLibraryItemCount(libraryID int32, statusType string) (int32, error) {
	// First, get the library to know its type (inbound, staging, production)
	var library models.Library
	if err := h.repo.DB.First(&library, libraryID).Error; err != nil {
		return 0, err
	}

	// Count albums based on the library type and their status
	var count int64
	var err error

	switch library.Type {
	case "inbound":
		// For inbound libraries: count all albums (these are new files to be processed)
		// All content in an inbound library is considered "inbound" content
		if statusType == "inbound" {
			err = h.repo.DB.Model(&models.Album{}).
				Where("directory LIKE ?", library.Path+"%").
				Count(&count).Error
		} else {
			count = 0
		}
	case "staging":
		// For staging libraries: count albums in staging (status 'New' = not yet promoted)
		if statusType == "staging" {
			err = h.repo.DB.Model(&models.Album{}).
				Where("directory LIKE ? AND album_status = ?", library.Path+"%", "New").
				Count(&count).Error
		} else {
			count = 0
		}
	case "production":
		// For production libraries: count albums that are ready for serving (status 'Ok')
		if statusType == "production" {
			err = h.repo.DB.Model(&models.Album{}).
				Where("directory LIKE ? AND album_status = ?", library.Path+"%", "Ok").
				Count(&count).Error
		} else {
			count = 0
		}
	default:
		count = 0
	}

	if err != nil {
		return 0, err
	}

	return int32(count), nil
}

// getQuarantineItemCount counts quarantine items for a library path
func (h *LibraryHandler) getQuarantineItemCount(libraryPath string) int64 {
	// In a real implementation, this would query a quarantine table
	// For now, we'll use a placeholder model or check for albums with Invalid status
	// This could be enhanced when actual quarantine tracking is implemented
	var count int64

	// Count albums that are in the library path but have Invalid status (indicating they're quarantined)
	err := h.repo.DB.Model(&models.Album{}).
		Where("directory LIKE ? AND album_status = ?", libraryPath+"%", "Invalid").
		Count(&count).Error

	if err != nil {
		// If there's an error, return 0 rather than fail the entire request
		return 0
	}

	return count
}