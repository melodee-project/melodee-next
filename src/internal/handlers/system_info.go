package handlers

import (
	"fmt"

	"melodee/internal/config"

	"github.com/gofiber/fiber/v2"
)

// SystemInfoHandler exposes system/runtime info for the admin UI
type SystemInfoHandler struct {
	cfg *config.AppConfig
}

func NewSystemInfoHandler(cfg *config.AppConfig) *SystemInfoHandler {
	return &SystemInfoHandler{cfg: cfg}
}

// OpenSubsonicInfo returns the canonical OpenSubsonic base URL as seen by the client
// It prefers the Host header (proxy-aware) and request scheme, with a fallback to server config.
func (h *SystemInfoHandler) OpenSubsonicInfo(c *fiber.Ctx) error {
	scheme := c.Protocol() // "http" or "https"
	hostHeader := c.Get("Host")
	note := "derived from request Host header"

	if hostHeader == "" {
		// Fallback to configured host:port
		hostHeader = fmt.Sprintf("%s:%d", h.cfg.Server.Host, h.cfg.Server.Port)
		note = "derived from server config host:port"
	}

	base := fmt.Sprintf("%s://%s/rest", scheme, hostHeader)

	return c.JSON(fiber.Map{
		"base":           base,
		"ping":           base + "/ping.view",
		"example_stream": base + "/stream.view?id=<trackId>&u=<user>&t=<token>&s=<salt>&v=1.16.1&c=melodee",
		"enabled":        true, // always exposed; auth handled by OpenSubsonic middleware where applicable
		"source":         note,
	})
}
