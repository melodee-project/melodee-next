package handlers

import (
	"melodee/open_subsonic/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type ScanHandler struct {
	DB *gorm.DB
}

func NewScanHandler(db *gorm.DB) *ScanHandler {
	return &ScanHandler{DB: db}
}

func (h *ScanHandler) GetScanStatus(c *fiber.Ctx) error {
	// Stub implementation
	// In a real implementation, we would check the status of the scanner service
	return utils.SendResponse(c, &utils.OpenSubsonicResponse{
		Status:  "ok",
		Version: "1.16.1",
		ScanStatus: &utils.ScanStatus{
			Scanning: false,
			Count:    0, // Total scanned count
		},
	})
}

func (h *ScanHandler) StartScan(c *fiber.Ctx) error {
	// Stub implementation
	return utils.SendResponse(c, &utils.OpenSubsonicResponse{
		Status:  "ok",
		Version: "1.16.1",
		ScanStatus: &utils.ScanStatus{
			Scanning: true,
			Count:    0,
		},
	})
}
