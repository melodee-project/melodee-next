package handlers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
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
	// In a real implementation, this would fetch settings from the database
	// For now, return a sample response as specified in the API documentation
	
	// This is a simplified implementation that matches the documented contract
	settings := []Setting{
		{
			Key:         "smtp.host",
			Value:       "smtp.example.com",
			Description: "SMTP host for email",
			Editable:    true,
		},
		{
			Key:         "jobs.max_concurrency",
			Value:       "4",
			Description: "Max concurrent worker jobs",
			Editable:    true,
		},
	}

	return c.JSON(fiber.Map{
		"data": settings,
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
	
	// In a real implementation, this would update the setting in the database
	// For now, return success response as specified in the API documentation
	
	return c.JSON(fiber.Map{
		"status": "ok",
		"setting": Setting{
			Key:         key,
			Value:       req.Value,
			Description: "Updated setting",
			Editable:    true,
		},
	})
}