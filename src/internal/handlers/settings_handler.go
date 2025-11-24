package handlers

import (
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"melodee/internal/models"
	"melodee/internal/pagination"
	"melodee/internal/services"
	"melodee/internal/utils"
)

// SettingsHandler manages application settings
type SettingsHandler struct {
	repo *services.Repository
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(repo *services.Repository) *SettingsHandler {
	return &SettingsHandler{
		repo: repo,
	}
}

// Setting represents a single setting
type Setting struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
	Editable    bool   `json:"editable"`
}

// GetSettings retrieves all application settings
func (h *SettingsHandler) GetSettings(c *fiber.Ctx) error {
	// Get pagination parameters
	page, pageSize := pagination.GetPaginationParams(c, 1, 50)
	offset := pagination.CalculateOffset(page, pageSize)

	// Query all settings from the database
	var settings []models.Setting
	var total int64

	// Count total settings
	err := h.repo.GetDB().Model(&models.Setting{}).Count(&total).Error
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to count settings")
	}

	// Fetch settings with pagination
	err = h.repo.GetDB().
		Offset(offset).
		Limit(pageSize).
		Order("key ASC").
		Find(&settings).Error
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to retrieve settings")
	}

	// Convert to response format
	responseSettings := make([]Setting, len(settings))
	for i, setting := range settings {
		responseSettings[i] = Setting{
			Key:         setting.Key,
			Value:       setting.Value,
			Description: setting.Comment, // Use comment field as description
			Editable:    true, // In a real implementation, this could be based on setting type
		}
	}

	// Calculate pagination metadata according to OpenAPI spec
	paginationMeta := pagination.Calculate(total, page, pageSize)

	return c.JSON(fiber.Map{
		"data": responseSettings,
		"pagination": paginationMeta,
	})
}

// UpdateSetting updates a single application setting
func (h *SettingsHandler) UpdateSetting(c *fiber.Ctx) error {
	key := c.Params("key")

	var req struct {
		Value string `json:"value"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid request body")
	}

	// Find the existing setting or create if it doesn't exist
	var existingSetting models.Setting
	result := h.repo.GetDB().Where("key = ?", key).First(&existingSetting)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// Create new setting
			newSetting := models.Setting{
				Key:       key,
				Value:     req.Value,
				Category:  nil,    // Default to no category
				Comment:   "",     // Default to no comment
			}

			if err := h.repo.GetDB().Create(&newSetting).Error; err != nil {
				return utils.SendInternalServerError(c, "Failed to create setting")
			}

			// Return the created setting in the specified contract format
			settingResponse := Setting{
				Key:         newSetting.Key,
				Value:       newSetting.Value,
				Description: newSetting.Comment,
				Editable:    true,
			}

			return c.JSON(fiber.Map{
				"status": "ok",
				"setting": settingResponse,
			})
		} else {
			return utils.SendInternalServerError(c, "Failed to find setting")
		}
	} else {
		// Update existing setting
		existingSetting.Value = req.Value
		existingSetting.UpdatedAt = time.Now()

		if err := h.repo.GetDB().Save(&existingSetting).Error; err != nil {
			return utils.SendInternalServerError(c, "Failed to update setting")
		}

		// Return the updated setting in the specified contract format
		settingResponse := Setting{
			Key:         existingSetting.Key,
			Value:       existingSetting.Value,
			Description: existingSetting.Comment,
			Editable:    true,
		}

		return c.JSON(fiber.Map{
			"status": "ok",
			"setting": settingResponse,
		})
	}
}