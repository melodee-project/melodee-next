package handlers

import (
	"melodee/open_subsonic/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type JukeboxHandler struct {
	DB *gorm.DB
}

func NewJukeboxHandler(db *gorm.DB) *JukeboxHandler {
	return &JukeboxHandler{DB: db}
}

func (h *JukeboxHandler) JukeboxControl(c *fiber.Ctx) error {
	action := c.Query("action")
	if action == "" {
		return utils.SendOpenSubsonicError(c, 10, "Missing action parameter")
	}

	// Stub implementation
	// Actions: get, status, set, start, stop, skip, add, clear, remove, shuffle, setGain

	return utils.SendResponse(c, &utils.OpenSubsonicResponse{
		Status:  "ok",
		Version: "1.16.1",
		JukeboxStatus: &utils.JukeboxStatus{
			CurrentIndex: 0,
			Playing:      false,
			Gain:         0.5,
			Position:     0,
		},
	})
}
