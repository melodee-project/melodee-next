package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestLibraryHandler_StatsRouterWiring(t *testing.T) {
	app := fiber.New()

	// Create a simple library handler for testing
	libraryHandler := NewLibraryHandler(nil, nil, nil, nil) // Use nil components for smoke test

	// Test GetLibrariesStats endpoint
	app.Get("/test-libraries-stats", libraryHandler.GetLibrariesStats)
	req := httptest.NewRequest("GET", "/test-libraries-stats", nil)
	resp, _ := app.Test(req)
	// We expect this to return at least a 401 (unauthorized) or 500 (due to nil repo), but not panic
	assert.True(t, resp.StatusCode == 401 || resp.StatusCode == 500 || resp.StatusCode == 200)
}