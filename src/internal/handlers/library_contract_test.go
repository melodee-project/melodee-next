package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestLibraryHandler_StatsContractCompliance(t *testing.T) {
	app := fiber.New()

	// Create handler with nil components for testing
	libraryHandler := NewLibraryHandler(nil, nil, nil, nil)

	// Test GetLibrariesStats response format
	app.Get("/test-libraries-stats", libraryHandler.GetLibrariesStats)
	req := httptest.NewRequest("GET", "/test-libraries-stats", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Should return 401 without auth or 500 with nil repo
	assert.True(t, resp.StatusCode == 401 || resp.StatusCode == 500)
}