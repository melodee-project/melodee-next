package admin

import (
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"melodee/internal/middleware"
	"melodee/internal/models"
	"melodee/internal/services"
)

// SharesHandler manages sharing functionality
type SharesHandler struct {
	db          *gorm.DB
	repo        *services.Repository
	authService *services.AuthService
}

// NewSharesHandler creates a new shares handler
func NewSharesHandler(
	db *gorm.DB,
	repo *services.Repository,
	authService *services.AuthService,
) *SharesHandler {
	return &SharesHandler{
		db:          db,
		repo:        repo,
		authService: authService,
	}
}

// Share represents a shared entity
type Share struct {
	ID                   int32     `json:"id"`
	APIKey               string    `json:"api_key"`
	UserID               int64     `json:"user_id"`
	Name                 string    `json:"name"`
	Description          *string   `json:"description,omitempty"`
	ExpiresAt            *time.Time `json:"expires_at,omitempty"`
	MaxStreamingMinutes  *int32    `json:"max_streaming_minutes,omitempty"`
	MaxStreamingCount    *int32    `json:"max_streaming_count,omitempty"`
	AllowStreaming       bool      `json:"allow_streaming"`
	AllowDownload        bool      `json:"allow_download"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// GetSharesRequest is the request structure for getting shares
type GetSharesRequest struct {
	Page   int    `query:"page"`
	Size   int    `query:"size"`
	Filter string `query:"filter"`
}

// GetSharesResponse is the response structure for getting shares
type GetSharesResponse struct {
	Data       []Share    `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// CreateShareRequest is the request structure for creating a share
type CreateShareRequest struct {
	Name                 string     `json:"name"`
	Description          *string    `json:"description,omitempty"`
	IDs                  []int64    `json:"ids"`
	ExpiresAt            *time.Time `json:"expires_at,omitempty"`
	MaxStreamingMinutes  *int32     `json:"max_streaming_minutes,omitempty"`
	MaxStreamingCount    *int32     `json:"max_streaming_count,omitempty"`
	AllowStreaming       bool       `json:"allow_streaming"`
	AllowDownload        bool       `json:"allow_download"`
}

// UpdateShareRequest is the request structure for updating a share
type UpdateShareRequest struct {
	Name                 *string    `json:"name,omitempty"`
	Description          *string    `json:"description,omitempty"`
	IDs                  *[]int64   `json:"ids,omitempty"`
	ExpiresAt            *time.Time `json:"expires_at,omitempty"`
	MaxStreamingMinutes  *int32     `json:"max_streaming_minutes,omitempty"`
	MaxStreamingCount    *int32     `json:"max_streaming_count,omitempty"`
	AllowStreaming       *bool      `json:"allow_streaming,omitempty"`
	AllowDownload        *bool      `json:"allow_download,omitempty"`
}

// GetShares retrieves all shares
func (h *SharesHandler) GetShares(c *fiber.Ctx) error {
	// Check if user is admin
	user, ok := middleware.GetUserFromContext(c)
	if !ok || !user.IsAdmin {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	// Parse query parameters
	page := c.QueryInt("page", 1)
	size := c.QueryInt("size", 50)
	filter := c.Query("filter", "")

	// Set limits
	if size > 100 {
		size = 100 // Max page size
	}

	// Build query
	query := h.db.Model(&models.Share{})
	
	if filter != "" {
		query = query.Where("name ILIKE ?", "%"+filter+"%")
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to count shares",
		})
	}

	// Get shares with pagination
	var dbShares []models.Share
	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Find(&dbShares).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve shares",
		})
	}

	// Convert to response format
	shares := make([]Share, len(dbShares))
	for i, s := range dbShares {
		shares[i] = Share{
			ID:                   s.ID,
			APIKey:               s.APIKey.String(),
			UserID:               s.UserID,
			Name:                 s.Name,
			Description:          s.Description,
			ExpiresAt:            s.ExpiresAt,
			MaxStreamingMinutes:  s.MaxStreamingMinutes,
			MaxStreamingCount:    s.MaxStreamingCount,
			AllowStreaming:       s.AllowStreaming,
			AllowDownload:        s.AllowDownload,
			CreatedAt:            s.CreatedAt,
			UpdatedAt:            s.UpdatedAt,
		}
	}

	response := GetSharesResponse{
		Data: shares,
		Pagination: Pagination{
			Page:  page,
			Size:  len(shares),
			Total: int(total),
		},
	}

	return c.JSON(response)
}

// GetShare retrieves a specific share
func (h *SharesHandler) GetShare(c *fiber.Ctx) error {
	// Check if user is admin
	user, ok := middleware.GetUserFromContext(c)
	if !ok || !user.IsAdmin {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	shareID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid share ID",
		})
	}

	var dbShare models.Share
	if err := h.db.Preload("User").First(&dbShare, shareID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{
				"error": "Share not found",
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve share",
		})
	}

	share := Share{
		ID:                   dbShare.ID,
		APIKey:               dbShare.APIKey.String(),
		UserID:               dbShare.UserID,
		Name:                 dbShare.Name,
		Description:          dbShare.Description,
		ExpiresAt:            dbShare.ExpiresAt,
		MaxStreamingMinutes:  dbShare.MaxStreamingMinutes,
		MaxStreamingCount:    dbShare.MaxStreamingCount,
		AllowStreaming:       dbShare.AllowStreaming,
		AllowDownload:        dbShare.AllowDownload,
		CreatedAt:            dbShare.CreatedAt,
		UpdatedAt:            dbShare.UpdatedAt,
	}

	return c.JSON(share)
}

// CreateShare creates a new share
func (h *SharesHandler) CreateShare(c *fiber.Ctx) error {
	// Check if user is admin
	user, ok := middleware.GetUserFromContext(c)
	if !ok || !user.IsAdmin {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	var req CreateShareRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Get current user to associate with the share
	currentUser, ok := middleware.GetUserFromContext(c)
	if !ok {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	// Create new share in database
	dbShare := models.Share{
		UserID:               currentUser.ID,
		Name:                 req.Name,
		Description:          req.Description,
		ExpiresAt:            req.ExpiresAt,
		MaxStreamingMinutes:  req.MaxStreamingMinutes,
		MaxStreamingCount:    req.MaxStreamingCount,
		AllowStreaming:       req.AllowStreaming,
		AllowDownload:        req.AllowDownload,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	if err := h.db.Create(&dbShare).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create share",
		})
	}

	// Create share activities for each ID if needed (would depend on specific implementation)
	// This would create entries in the ShareActivity table

	// Return created share
	share := Share{
		ID:                   dbShare.ID,
		APIKey:               dbShare.APIKey.String(),
		UserID:               dbShare.UserID,
		Name:                 dbShare.Name,
		Description:          dbShare.Description,
		ExpiresAt:            dbShare.ExpiresAt,
		MaxStreamingMinutes:  dbShare.MaxStreamingMinutes,
		MaxStreamingCount:    dbShare.MaxStreamingCount,
		AllowStreaming:       dbShare.AllowStreaming,
		AllowDownload:        dbShare.AllowDownload,
		CreatedAt:            dbShare.CreatedAt,
		UpdatedAt:            dbShare.UpdatedAt,
	}

	return c.Status(http.StatusCreated).JSON(share)
}

// UpdateShare updates an existing share
func (h *SharesHandler) UpdateShare(c *fiber.Ctx) error {
	// Check if user is admin
	user, ok := middleware.GetUserFromContext(c)
	if !ok || !user.IsAdmin {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	shareID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid share ID",
		})
	}

	// Check if share exists and belongs to the current user or if user is admin
	var dbShare models.Share
	if err := h.db.First(&dbShare, shareID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{
				"error": "Share not found",
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve share",
		})
	}

	var req UpdateShareRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Update fields if provided
	if req.Name != nil {
		dbShare.Name = *req.Name
	}
	if req.Description != nil {
		dbShare.Description = req.Description
	}
	if req.ExpiresAt != nil {
		dbShare.ExpiresAt = req.ExpiresAt
	}
	if req.MaxStreamingMinutes != nil {
		dbShare.MaxStreamingMinutes = req.MaxStreamingMinutes
	}
	if req.MaxStreamingCount != nil {
		dbShare.MaxStreamingCount = req.MaxStreamingCount
	}
	if req.AllowStreaming != nil {
		dbShare.AllowStreaming = *req.AllowStreaming
	}
	if req.AllowDownload != nil {
		dbShare.AllowDownload = *req.AllowDownload
	}

	dbShare.UpdatedAt = time.Now()

	if err := h.db.Save(&dbShare).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update share",
		})
	}

	// Return updated share
	share := Share{
		ID:                   dbShare.ID,
		APIKey:               dbShare.APIKey.String(),
		UserID:               dbShare.UserID,
		Name:                 dbShare.Name,
		Description:          dbShare.Description,
		ExpiresAt:            dbShare.ExpiresAt,
		MaxStreamingMinutes:  dbShare.MaxStreamingMinutes,
		MaxStreamingCount:    dbShare.MaxStreamingCount,
		AllowStreaming:       dbShare.AllowStreaming,
		AllowDownload:        dbShare.AllowDownload,
		CreatedAt:            dbShare.CreatedAt,
		UpdatedAt:            dbShare.UpdatedAt,
	}

	return c.JSON(share)
}

// DeleteShare deletes a share
func (h *SharesHandler) DeleteShare(c *fiber.Ctx) error {
	// Check if user is admin
	user, ok := middleware.GetUserFromContext(c)
	if !ok || !user.IsAdmin {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	shareID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid share ID",
		})
	}

	var existingShare models.Share
	if err := h.db.First(&existingShare, shareID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{
				"error": "Share not found",
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to verify share exists",
		})
	}

	// Delete the share
	if err := h.db.Delete(&existingShare).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete share",
		})
	}

	return c.JSON(fiber.Map{
		"status": "deleted",
	})
}