package handlers

import (
	"fmt"
	"net"
	"strings"

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
// It prefers the Host header (proxy-aware) and request scheme, with a fallback to LAN IP.
func (h *SystemInfoHandler) OpenSubsonicInfo(c *fiber.Ctx) error {
	scheme := c.Protocol() // "http" or "https"
	hostHeader := c.Get("Host")
	note := "derived from request Host header"

	if hostHeader == "" || strings.HasPrefix(hostHeader, "localhost") || strings.HasPrefix(hostHeader, "127.0.0.1") {
		// If using localhost, prefer LAN IP for better client compatibility
		// (Chrome blocks HTTPS -> localhost HTTP for security)
		lanIP := getPreferredLANIP()
		if lanIP != "" {
			hostHeader = fmt.Sprintf("%s:%d", lanIP, h.cfg.Server.Port)
			note = "using LAN IP (recommended for web clients)"
		} else {
			// Fallback to configured host:port
			hostHeader = fmt.Sprintf("%s:%d", h.cfg.Server.Host, h.cfg.Server.Port)
			note = "derived from server config host:port"
		}
	}

	base := fmt.Sprintf("%s://%s/rest", scheme, hostHeader)
	baseForClients := fmt.Sprintf("%s://%s", scheme, hostHeader) // Base URL without /rest for client configuration

	return c.JSON(fiber.Map{
		"base":            base,           // Full URL with /rest for display
		"base_for_client": baseForClients, // Base URL without /rest for client configuration
		"ping":            base + "/ping.view",
		"example_stream":  base + "/stream.view?id=<trackId>&u=<user>&t=<token>&s=<salt>&v=1.16.1&c=melodee",
		"enabled":         true, // always exposed; auth handled by OpenSubsonic middleware where applicable
		"source":          note,
	})
}

// getPreferredLANIP returns the first non-loopback IPv4 address
func getPreferredLANIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip := ipnet.IP.String()
				// Prefer 192.168.x.x addresses over others (common LAN)
				if strings.HasPrefix(ip, "192.168.") {
					return ip
				}
			}
		}
	}

	// If no 192.168.x.x found, return any non-loopback IPv4
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return ""
}
