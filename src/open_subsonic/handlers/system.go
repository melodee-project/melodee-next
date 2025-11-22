package handlers

import (
	"encoding/xml"
	"time"

	"github.com/gofiber/fiber/v2"

	"melodee/open_subsonic/utils"
)

// SystemHandler handles OpenSubsonic system endpoints
type SystemHandler struct {
	db interface{} // Using interface{} as placeholder until we determine actual needs
}

// NewSystemHandler creates a new system handler
func NewSystemHandler(db interface{}) *SystemHandler {
	return &SystemHandler{
		db: db,
	}
}

// Ping tests API connectivity
func (h *SystemHandler) Ping(c *fiber.Ctx) error {
	// In OpenSubsonic, ping should return an empty response with status="ok"
	response := utils.SuccessResponse()
	
	return utils.SendResponse(c, response)
}

// GetLicense returns licensing information
func (h *SystemHandler) GetLicense(c *fiber.Ctx) error {
	// Create license response
	response := utils.SuccessResponse()
	license := License{
		XMLName: xml.Name{Local: "license"},
		ID:      "melodee-license-001", // Placeholder ID
		Email:   "license@melodee.example.com", // Placeholder email
		License: "AGPL-3.0", // Actual license
		Version: "1.0.0", // Application version
		Created: utils.FormatTime(time.Now()), // Current time
		Expiry:  utils.FormatTime(time.Now().AddDate(100, 0, 0)), // For perpetual licenses, set to far future
		Valid:   true, // Whether the license is valid
	}

	response.License = &license
	return utils.SendResponse(c, response)
}

// License represents license information
type License struct {
	XMLName xml.Name `xml:"license"`
	ID      string   `xml:"id,attr"`
	Email   string   `xml:"email,attr"`
	License string   `xml:"license,attr"` // The type of license
	Version string   `xml:"version,attr"`
	Created string   `xml:"created,attr"`
	Expiry  string   `xml:"expires,attr"` // When the license expires
	Valid   bool     `xml:"valid,attr"`
}