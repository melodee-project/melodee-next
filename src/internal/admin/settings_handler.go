package admin

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"melodee/internal/middleware"
	"melodee/internal/models"
	"melodee/internal/pagination"
	"melodee/internal/services"
)

// SettingsHandler manages application settings
type SettingsHandler struct {
	db          *gorm.DB
	repo        *services.Repository
	authService *services.AuthService
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(
	db *gorm.DB,
	repo *services.Repository,
	authService *services.AuthService,
) *SettingsHandler {
	return &SettingsHandler{
		db:          db,
		repo:        repo,
		authService: authService,
	}
}

// Setting represents a single setting
type Setting struct {
	ID        int32  `json:"id"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	Category  *int32 `json:"category,omitempty"`
	Comment   string `json:"comment,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// GetSettingsRequest is the request structure for getting settings
type GetSettingsRequest struct {
	Page    int    `query:"page"`
	Size    int    `query:"size"`
	Filter  string `query:"filter"`
	Category *int32 `query:"category"`
}

// GetSettingsResponse is the response structure for getting settings
type GetSettingsResponse struct {
	Data       []Setting `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// Pagination represents pagination information
type Pagination struct {
	Page  int `json:"page"`
	Size  int `json:"size"`
	Total int `json:"total"`
}

// UpdateSettingRequest is the request structure for updating a setting
type UpdateSettingRequest struct {
	Value string `json:"value"`
}

// GetSettings retrieves all application settings
func (h *SettingsHandler) GetSettings(c *fiber.Ctx) error {
	// Check if user is admin
	user, ok := middleware.GetUserFromContext(c)
	if !ok || !user.IsAdmin {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	// Parse query parameters
	page, pageSize := pagination.GetPaginationParams(c, 1, 50)
	filter := c.Query("filter", "")
	category := c.Query("category", "")

	// Build query
	query := h.db.Model(&models.Setting{})

	if filter != "" {
		query = query.Where("key ILIKE ?", "%"+filter+"%")
	}

	if category != "" {
		catValue := -1
		// In a real app, category would be validated against an enum/lookup table
		query = query.Where("category = ?", catValue)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to count settings",
		})
	}

	// Get settings with pagination
	var dbSettings []models.Setting
	offset := pagination.CalculateOffset(page, pageSize)
	if err := query.Offset(offset).Limit(pageSize).Find(&dbSettings).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve settings",
		})
	}

	// Convert to response format
	settings := make([]Setting, len(dbSettings))
	for i, s := range dbSettings {
		settings[i] = Setting{
			ID:        s.ID,
			Key:       s.Key,
			Value:     s.Value,
			Category:  s.Category,
			Comment:   s.Comment,
			CreatedAt: s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"), // RFC3339 format
			UpdatedAt: s.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	// Calculate pagination metadata according to OpenAPI spec
	paginationMeta := pagination.Calculate(total, page, pageSize)

	return c.JSON(fiber.Map{
		"data":       settings,
		"pagination": paginationMeta,
	})
}

// GetSetting retrieves a specific application setting
func (h *SettingsHandler) GetSetting(c *fiber.Ctx) error {
	// Check if user is admin
	user, ok := middleware.GetUserFromContext(c)
	if !ok || !user.IsAdmin {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	settingID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid setting ID",
		})
	}

	var dbSetting models.Setting
	if err := h.db.First(&dbSetting, settingID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{
				"error": "Setting not found",
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve setting",
		})
	}

	setting := Setting{
		ID:        dbSetting.ID,
		Key:       dbSetting.Key,
		Value:     dbSetting.Value,
		Category:  dbSetting.Category,
		Comment:   dbSetting.Comment,
		CreatedAt: dbSetting.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: dbSetting.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	return c.JSON(setting)
}

// UpdateSetting updates an application setting
func (h *SettingsHandler) UpdateSetting(c *fiber.Ctx) error {
	// Check if user is admin
	user, ok := middleware.GetUserFromContext(c)
	if !ok || !user.IsAdmin {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	settingID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid setting ID",
		})
	}

	var req UpdateSettingRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate setting key exists
	var existingSetting models.Setting
	if err := h.db.First(&existingSetting, settingID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{
				"error": "Setting not found",
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to verify setting exists",
		})
	}

	// Update the setting
	existingSetting.Value = req.Value
	if err := h.db.Save(&existingSetting).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update setting",
		})
	}

	// Return updated setting
	setting := Setting{
		ID:        existingSetting.ID,
		Key:       existingSetting.Key,
		Value:     existingSetting.Value,
		Category:  existingSetting.Category,
		Comment:   existingSetting.Comment,
		CreatedAt: existingSetting.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: existingSetting.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	return c.JSON(setting)
}

// CreateSetting creates a new application setting
func (h *SettingsHandler) CreateSetting(c *fiber.Ctx) error {
	// Check if user is admin
	user, ok := middleware.GetUserFromContext(c)
	if !ok || !user.IsAdmin {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	var setting Setting
	if err := c.BodyParser(&setting); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Create new setting in database
	dbSetting := models.Setting{
		Key:      setting.Key,
		Value:    setting.Value,
		Category: setting.Category,
		Comment:  setting.Comment,
	}

	if err := h.db.Create(&dbSetting).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create setting",
		})
	}

	// Return created setting
	createdSetting := Setting{
		ID:        dbSetting.ID,
		Key:       dbSetting.Key,
		Value:     dbSetting.Value,
		Category:  dbSetting.Category,
		Comment:   dbSetting.Comment,
		CreatedAt: dbSetting.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: dbSetting.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	return c.Status(http.StatusCreated).JSON(createdSetting)
}

// DeleteSetting deletes an application setting
func (h *SettingsHandler) DeleteSetting(c *fiber.Ctx) error {
	// Check if user is admin
	user, ok := middleware.GetUserFromContext(c)
	if !ok || !user.IsAdmin {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	settingID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid setting ID",
		})
	}

	var existingSetting models.Setting
	if err := h.db.First(&existingSetting, settingID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{
				"error": "Setting not found",
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to verify setting exists",
		})
	}

	// Delete the setting
	if err := h.db.Delete(&existingSetting).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete setting",
		})
	}

	return c.JSON(fiber.Map{
		"status": "deleted",
	})
}