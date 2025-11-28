package handlers

import (
	"melodee/open_subsonic/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type VideoHandler struct {
	DB *gorm.DB
}

func NewVideoHandler(db *gorm.DB) *VideoHandler {
	return &VideoHandler{DB: db}
}

func (h *VideoHandler) GetVideos(c *fiber.Ctx) error {
	// Stub implementation
	return utils.SendResponse(c, &utils.OpenSubsonicResponse{
		Status:  "ok",
		Version: "1.16.1",
		Videos:  &utils.Videos{Videos: []utils.Child{}},
	})
}

func (h *VideoHandler) GetVideoInfo(c *fiber.Ctx) error {
	id := c.Query("id")
	if id == "" {
		return utils.SendOpenSubsonicError(c, 10, "Missing id parameter")
	}

	// Stub implementation
	return utils.SendResponse(c, &utils.OpenSubsonicResponse{
		Status:  "ok",
		Version: "1.16.1",
		VideoInfo: &utils.VideoInfo{
			ID: id,
		},
	})
}

func (h *VideoHandler) HLS(c *fiber.Ctx) error {
	// Stub implementation
	// In a real implementation, this would return an m3u8 playlist or ts segment
	return c.SendStatus(404)
}

func (h *VideoHandler) GetCaptions(c *fiber.Ctx) error {
	id := c.Query("id")
	if id == "" {
		return utils.SendOpenSubsonicError(c, 10, "Missing id parameter")
	}

	// Stub implementation
	// In a real implementation, this would return the VTT or SRT content
	return c.SendStatus(404)
}
