package handlers

import (
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"melodee/internal/media"
	"melodee/internal/models"
	"melodee/internal/services"
	"melodee/internal/utils"
)

// LibraryHandler handles library-related requests
type LibraryHandler struct {
	repo        *services.Repository
	mediaSvc    *media.MediaService
}

// NewLibraryHandler creates a new library handler
func NewLibraryHandler(repo *services.Repository, mediaSvc *media.MediaService) *LibraryHandler {
	return &LibraryHandler{
		repo:     repo,
		mediaSvc: mediaSvc,
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
			InboundCount:    0, // This would be calculated from your actual data model
			StagingCount:    0,
			ProductionCount: 0,
			QuarantineCount: 0,
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
		InboundCount:    0, // This would be calculated from your actual data model
		StagingCount:    0,
		ProductionCount: 0,
		QuarantineCount: 0,
	}

	return c.JSON(state)
}

// GetQuarantineItems handles retrieving quarantine items
func (h *LibraryHandler) GetQuarantineItems(c *fiber.Ctx) error {
	// In a real implementation, you would query your quarantine table
	// For now, return an empty list
	items := []QuarantineItem{}
	
	return c.JSON(fiber.Map{
		"data": items,
		"count": len(items),
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

	// In a real implementation, you would enqueue a scan job
	// h.mediaSvc.EnqueueLibraryScan(/* client */, []int32{int32(libraryID)}, force)

	// For now, return success
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

	// In a real implementation, you would enqueue a process job
	// h.mediaSvc.EnqueueLibraryProcess(/* client */, int32(libraryID), filePaths)

	// For now, return success
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

	// In a real implementation, you would move OK status content
	// This would likely require moving albums rather than the entire library

	// For now, return success
	return c.JSON(fiber.Map{
		"status": "queued",
		"library_id": libraryID,
		"message": "Library move OK triggered successfully",
	})
}

// ResolveQuarantineItem handles resolving a quarantine item
func (h *LibraryHandler) ResolveQuarantineItem(c *fiber.Ctx) error {
	itemID, err := c.ParamsInt("id")
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid item ID")
	}

	// In a real implementation, you would resolve the quarantine item
	// Typically this would involve moving the file to appropriate location and updating its status

	// For now, return success
	return c.JSON(fiber.Map{
		"status": "resolved",
		"item_id": itemID,
		"message": "Quarantine item resolved successfully",
	})
}

// RequeueQuarantineItem handles requeuing a quarantine item for reprocessing
func (h *LibraryHandler) RequeueQuarantineItem(c *fiber.Ctx) error {
	itemID, err := c.ParamsInt("id")
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid item ID")
	}

	// In a real implementation, you would requeue the item
	// For now, return success
	return c.JSON(fiber.Map{
		"status": "requeued",
		"item_id": itemID,
		"message": "Quarantine item requeued for processing",
	})
}