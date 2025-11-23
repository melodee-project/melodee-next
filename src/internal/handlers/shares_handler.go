package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
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

	// In a real implementation, this would fetch shares from the database
	// For now, return a sample response with pagination as specified in documentation

	// This is a simplified implementation that matches the documented contract
	shares := []Share{
		{
			ID:                  "share-1",
			Name:                "Family Mix",
			TrackIDs:            []string{"track-1", "track-2"},
			ExpiresAt:           time.Now().AddDate(0, 0, 30), // 30 days from now
			MaxStreamingMinutes: 600,
			AllowDownload:       true,
			CreatedAt:           time.Now(),
		},
	}

	total := int64(len(shares)) // In real implementation, would query total from DB

	// Calculate pagination metadata according to OpenAPI spec
	paginationMeta := pagination.Calculate(total, page, pageSize)

	return c.JSON(fiber.Map{
		"data":       shares,
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
	
	// In a real implementation, this would create the share in the database
	// For now, return success response as specified in the API documentation
	
	expiryTime, err := time.Parse(time.RFC3339, req.ExpiresAt)
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid expires_at format, must be RFC3339")
	}
	
	share := Share{
		ID:                  uuid.New().String(),
		Name:                req.Name,
		TrackIDs:            req.TrackIDs,
		ExpiresAt:           expiryTime,
		MaxStreamingMinutes: req.MaxStreamingMinutes,
		AllowDownload:       req.AllowDownload,
		CreatedAt:           time.Now(),
	}
	
	return c.JSON(fiber.Map{
		"status": "ok",
		"share": share,
	})
}

// UpdateShare updates an existing share
func (h *SharesHandler) UpdateShare(c *fiber.Ctx) error {
	shareID := c.Params("id")
	
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
	
	// In a real implementation, this would update the share in the database
	// For now, return success response as specified in the API documentation
	
	expiryTime, err := time.Parse(time.RFC3339, req.ExpiresAt)
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid expires_at format, must be RFC3339")
	}
	
	updatedShare := Share{
		ID:                  shareID,
		Name:                req.Name,
		TrackIDs:            req.TrackIDs,
		ExpiresAt:           expiryTime,
		MaxStreamingMinutes: req.MaxStreamingMinutes,
		AllowDownload:       req.AllowDownload,
		CreatedAt:           time.Now(), // In real implementation, would preserve the original created time
	}
	
	return c.JSON(fiber.Map{
		"status": "ok",
		"share": updatedShare,
	})
}

// DeleteShare deletes an existing share
func (h *SharesHandler) DeleteShare(c *fiber.Ctx) error {
	shareID := c.Params("id")
	
	// Validate share ID format
	if _, err := uuid.Parse(shareID); err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid share ID format")
	}
	
	// In a real implementation, this would delete the share from the database
	// For now, return success response as specified in the API documentation
	
	return c.JSON(fiber.Map{
		"status": "deleted",
	})
}