package handlers

import (
	"fmt"
	"melodee/internal/models"
	"melodee/open_subsonic/utils"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type SharesHandler struct {
	DB *gorm.DB
}

func NewSharesHandler(db *gorm.DB) *SharesHandler {
	return &SharesHandler{DB: db}
}

func (h *SharesHandler) GetShares(c *fiber.Ctx) error {
	var shares []models.Share
	if err := h.DB.Preload("User").Find(&shares).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 70, "Could not fetch shares")
	}

	responseShares := make([]utils.Share, len(shares))
	for i, s := range shares {
		var expires string
		if s.ExpiresAt != nil {
			expires = utils.FormatTime(*s.ExpiresAt)
		}

		username := "admin"
		if s.User != nil {
			username = s.User.Username
		}

		responseShares[i] = utils.Share{
			ID:          fmt.Sprintf("%d", s.ID),
			Url:         fmt.Sprintf("http://example.com/share/%d", s.ID), // Placeholder URL
			Description: s.Description,
			Username:    username,
			Created:     utils.FormatTime(s.CreatedAt),
			Expires:     expires,
			VisitCount:  0, // Not tracked in current model?
		}
	}

	return utils.SendResponse(c, &utils.OpenSubsonicResponse{
		Status:  "ok",
		Version: "1.16.1",
		Shares:  &utils.Shares{Shares: responseShares},
	})
}

func (h *SharesHandler) CreateShare(c *fiber.Ctx) error {
	// id := c.Query("id") // Can be multiple

	description := c.Query("description")
	expiresStr := c.Query("expires")

	var expiresAt *time.Time
	if expiresStr != "" {
		ms, err := strconv.ParseInt(expiresStr, 10, 64)
		if err == nil {
			t := time.UnixMilli(ms)
			expiresAt = &t
		}
	}

	user, ok := utils.GetUserFromContext(c)
	if !ok {
		return utils.SendOpenSubsonicError(c, 50, "Unauthorized")
	}

	share := models.Share{
		UserID:      user.ID,
		Description: description,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.DB.Create(&share).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Could not create share")
	}

	// Return the created share
	responseShare := utils.Share{
		ID:          fmt.Sprintf("%d", share.ID),
		Url:         fmt.Sprintf("http://example.com/share/%d", share.ID),
		Description: share.Description,
		Username:    user.Username,
		Created:     utils.FormatTime(share.CreatedAt),
		VisitCount:  0,
	}
	if share.ExpiresAt != nil {
		responseShare.Expires = utils.FormatTime(*share.ExpiresAt)
	}

	return utils.SendResponse(c, &utils.OpenSubsonicResponse{
		Status:  "ok",
		Version: "1.16.1",
		Shares:  &utils.Shares{Shares: []utils.Share{responseShare}},
	})
}

func (h *SharesHandler) UpdateShare(c *fiber.Ctx) error {
	idStr := c.Query("id")
	if idStr == "" {
		return utils.SendOpenSubsonicError(c, 10, "Missing id parameter")
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return utils.SendOpenSubsonicError(c, 10, "Invalid id parameter")
	}

	var share models.Share
	if err := h.DB.First(&share, id).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 70, "Share not found")
	}

	description := c.Query("description")
	if description != "" {
		share.Description = description
	}

	expiresStr := c.Query("expires")
	if expiresStr != "" {
		ms, err := strconv.ParseInt(expiresStr, 10, 64)
		if err == nil {
			t := time.UnixMilli(ms)
			share.ExpiresAt = &t
		}
	}

	if err := h.DB.Save(&share).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Could not update share")
	}

	return utils.SendResponse(c, utils.SuccessResponse())
}

func (h *SharesHandler) DeleteShare(c *fiber.Ctx) error {
	idStr := c.Query("id")
	if idStr == "" {
		return utils.SendOpenSubsonicError(c, 10, "Missing id parameter")
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return utils.SendOpenSubsonicError(c, 10, "Invalid id parameter")
	}

	if err := h.DB.Delete(&models.Share{}, id).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 70, "Share not found")
	}

	return utils.SendResponse(c, utils.SuccessResponse())
}
