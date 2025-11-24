package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"melodee/internal/models"
	"melodee/internal/pagination"
	"melodee/internal/services"
	"melodee/internal/utils"
)

// SharesHandler manages share operations
type SharesHandler struct {
	repo *services.Repository
}

// NewSharesHandler creates a new shares handler
func NewSharesHandler(repo *services.Repository) *SharesHandler {
	return &SharesHandler{
		repo: repo,
	}
}

// Share represents a share
type Share struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	TrackIDs             []string  `json:"track_ids"`
	ExpiresAt            time.Time `json:"expires_at"`
	MaxStreamingMinutes  int       `json:"max_streaming_minutes"`
	AllowDownload        bool      `json:"allow_download"`
	CreatedAt            time.Time `json:"created_at"`
}

// GetSharesRequest represents the request for getting shares
type GetSharesRequest struct {
	Page int `query:"page"`
	Size int `query:"size"`
}

// GetSharesResponse represents the response for getting shares
type GetSharesResponse struct {
	Data       []Share      `json:"data"`
	Pagination Pagination   `json:"pagination"`
}

// Pagination represents pagination information
type Pagination struct {
	Page  int `json:"page"`
	Size  int `json:"size"`
	Total int `json:"total"`
}

// GetShares retrieves all shares
func (h *SharesHandler) GetShares(c *fiber.Ctx) error {
	page, pageSize := pagination.GetPaginationParams(c, 1, 50)
	offset := pagination.CalculateOffset(page, pageSize)

	// Query shares with pagination from the database
	var shares []models.Share
	if err := h.repo.GetDB().Offset(offset).Limit(pageSize).Find(&shares).Error; err != nil {
		return utils.SendInternalServerError(c, "Failed to retrieve shares")
	}

	// Count total shares for pagination metadata
	var total int64
	if err := h.repo.GetDB().Model(&models.Share{}).Count(&total).Error; err != nil {
		return utils.SendInternalServerError(c, "Failed to count shares")
	}

	// Convert to response format
	responseShares := make([]Share, len(shares))
	for i, share := range shares {
		var expiresAt time.Time
		if share.ExpiresAt != nil {
			expiresAt = *share.ExpiresAt
		}

		responseShares[i] = Share{
			ID:                  strconv.Itoa(int(share.ID)),
			Name:                share.Name,
			TrackIDs:            []string{}, // In a real implementation, would fetch track IDs from related tables
			ExpiresAt:           expiresAt,
			MaxStreamingMinutes: int(share.MaxStreamingMinutes),
			AllowDownload:       share.AllowDownload,
			CreatedAt:           share.CreatedAt,
		}
	}

	// Calculate pagination metadata according to OpenAPI spec
	paginationMeta := pagination.Calculate(total, page, pageSize)

	return c.JSON(fiber.Map{
		"data":       responseShares,
		"pagination": paginationMeta,
	})
}

// CreateShare creates a new share
func (h *SharesHandler) CreateShare(c *fiber.Ctx) error {
	var req struct {
		Name                 string   `json:"name"`
		TrackIDs             []string `json:"track_ids"`
		ExpiresAt            string   `json:"expires_at"`
		MaxStreamingMinutes  int      `json:"max_streaming_minutes"`
		AllowDownload        bool     `json:"allow_download"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid request body")
	}

	expiryTime, err := time.Parse(time.RFC3339, req.ExpiresAt)
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid expires_at format, must be RFC3339")
	}

	// In a real implementation, we'd need to get the authenticated user ID
	// For now, we'll use a placeholder user ID (1)
	newShare := models.Share{
		UserID:              1, // Placeholder - in real implementation, get from authenticated user
		Name:                req.Name,
		Description:         "", // Optionally could be added to the request
		ExpiresAt:           &expiryTime,
		MaxStreamingMinutes: int32(req.MaxStreamingMinutes),
		MaxStreamingCount:   0, // Default to unlimited
		AllowStreaming:      true, // Default to true
		AllowDownload:       req.AllowDownload,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	if err := h.repo.GetDB().Create(&newShare).Error; err != nil {
		return utils.SendInternalServerError(c, "Failed to create share")
	}

	// Convert to response format
	var expiresAt time.Time
	if newShare.ExpiresAt != nil {
		expiresAt = *newShare.ExpiresAt
	}

	responseShare := Share{
		ID:                  strconv.Itoa(int(newShare.ID)),
		Name:                newShare.Name,
		TrackIDs:            req.TrackIDs, // Use the provided track IDs
		ExpiresAt:           expiresAt,
		MaxStreamingMinutes: int(newShare.MaxStreamingMinutes),
		AllowDownload:       newShare.AllowDownload,
		CreatedAt:           newShare.CreatedAt,
	}

	return c.JSON(fiber.Map{
		"status": "ok",
		"share": responseShare,
	})
}

// UpdateShare updates an existing share
func (h *SharesHandler) UpdateShare(c *fiber.Ctx) error {
	shareID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid share ID")
	}

	var req struct {
		Name                 string   `json:"name"`
		TrackIDs             []string `json:"track_ids"`
		ExpiresAt            string   `json:"expires_at"`
		MaxStreamingMinutes  int      `json:"max_streaming_minutes"`
		AllowDownload        bool     `json:"allow_download"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid request body")
	}

	expiryTime, err := time.Parse(time.RFC3339, req.ExpiresAt)
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid expires_at format, must be RFC3339")
	}

	// Fetch the existing share to update
	var existingShare models.Share
	if err := h.repo.GetDB().First(&existingShare, shareID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return utils.SendNotFoundError(c, "Share")
		}
		return utils.SendInternalServerError(c, "Failed to find share")
	}

	// Update the existing share
	existingShare.Name = req.Name
	existingShare.ExpiresAt = &expiryTime
	existingShare.MaxStreamingMinutes = int32(req.MaxStreamingMinutes)
	existingShare.AllowDownload = req.AllowDownload
	existingShare.UpdatedAt = time.Now()

	if err := h.repo.GetDB().Save(&existingShare).Error; err != nil {
		return utils.SendInternalServerError(c, "Failed to update share")
	}

	// Convert to response format
	var expiresAt time.Time
	if existingShare.ExpiresAt != nil {
		expiresAt = *existingShare.ExpiresAt
	}

	updatedShare := Share{
		ID:                  strconv.Itoa(int(existingShare.ID)),
		Name:                existingShare.Name,
		TrackIDs:            req.TrackIDs, // Use the provided track IDs
		ExpiresAt:           expiresAt,
		MaxStreamingMinutes: int(existingShare.MaxStreamingMinutes),
		AllowDownload:       existingShare.AllowDownload,
		CreatedAt:           existingShare.CreatedAt, // Preserve the original created time
	}

	return c.JSON(fiber.Map{
		"status": "ok",
		"share": updatedShare,
	})
}

// DeleteShare deletes an existing share
func (h *SharesHandler) DeleteShare(c *fiber.Ctx) error {
	shareID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid share ID")
	}

	// Find and delete the share from the database
	var existingShare models.Share
	if err := h.repo.GetDB().First(&existingShare, shareID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return utils.SendNotFoundError(c, "Share")
		}
		return utils.SendInternalServerError(c, "Failed to find share")
	}

	if err := h.repo.GetDB().Delete(&existingShare).Error; err != nil {
		return utils.SendInternalServerError(c, "Failed to delete share")
	}

	return c.JSON(fiber.Map{
		"status": "deleted",
	})
}