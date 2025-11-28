package handlers

import (
	"time"

	"melodee/open_subsonic/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type ChatHandler struct {
	DB *gorm.DB
}

func NewChatHandler(db *gorm.DB) *ChatHandler {
	return &ChatHandler{DB: db}
}

func (h *ChatHandler) GetChatMessages(c *fiber.Ctx) error {
	// Stub implementation
	// since := c.QueryInt("since", 0)

	return utils.SendResponse(c, &utils.OpenSubsonicResponse{
		Status:       "ok",
		Version:      "1.16.1",
		ChatMessages: &utils.ChatMessages{Messages: []utils.ChatMessage{}},
	})
}

func (h *ChatHandler) AddChatMessage(c *fiber.Ctx) error {
	message := c.Query("message")
	if message == "" {
		return utils.SendOpenSubsonicError(c, 10, "Missing message parameter")
	}

	// Stub implementation - just echo back
	user, _ := utils.GetUserFromContext(c)
	username := "anonymous"
	if user != nil {
		username = user.Username
	}

	return utils.SendResponse(c, &utils.OpenSubsonicResponse{
		Status:  "ok",
		Version: "1.16.1",
		ChatMessages: &utils.ChatMessages{
			Messages: []utils.ChatMessage{
				{
					Username: username,
					Time:     time.Now().UnixMilli(),
					Message:  message,
				},
			},
		},
	})
}
