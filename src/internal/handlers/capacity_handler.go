package handlers

import (
	"melodee/internal/models"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// CapacityHandler handles capacity-related API requests
type CapacityHandler struct {
	db *gorm.DB
}

// NewCapacityHandler creates a new capacity handler
func NewCapacityHandler(db *gorm.DB) *CapacityHandler {
	return &CapacityHandler{
		db: db,
	}
}

// GetAllCapacityStatuses returns capacity status for all libraries
// GET /api/admin/capacity
func (h *CapacityHandler) GetAllCapacityStatuses(c *fiber.Ctx) error {
	var statuses []models.CapacityStatus

	if err := h.db.Find(&statuses).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch capacity statuses",
		})
	}

	// Convert to response format
	var results []fiber.Map
	for _, status := range statuses {
		results = append(results, fiber.Map{
			"library_id":     status.LibraryID,
			"path":           status.Path,
			"used_percent":   status.UsedPercent,
			"status":         status.Status,
			"latest_read_at": status.LatestReadAt,
			"error_count":    status.ErrorCount,
			"last_error":     status.LastError,
			"next_check_at":  status.NextCheckAt,
		})
	}

	return c.JSON(fiber.Map{
		"data": results,
	})
}

// GetCapacityForLibrary returns capacity status for a specific library
// GET /api/admin/capacity/:id
func (h *CapacityHandler) GetCapacityForLibrary(c *fiber.Ctx) error {
	libraryID, err := strconv.ParseInt(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid library ID",
		})
	}

	var status models.CapacityStatus
	if err := h.db.Where("library_id = ?", libraryID).First(&status).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Capacity status not found for this library",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch capacity status",
		})
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"library_id":     status.LibraryID,
			"path":           status.Path,
			"used_percent":   status.UsedPercent,
			"status":         status.Status,
			"latest_read_at": status.LatestReadAt,
			"error_count":    status.ErrorCount,
			"last_error":     status.LastError,
			"next_check_at":  status.NextCheckAt,
		},
	})
}
