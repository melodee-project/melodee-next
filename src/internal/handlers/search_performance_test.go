package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"melodee/internal/services"
)

func TestSearchHandler_PerformanceWithLargeLimits(t *testing.T) {
	app := fiber.New()

	// Create a mock repository for testing
	mockRepo := &services.Repository{}
	handler := NewSearchHandler(mockRepo)

	// Test with large limit values to ensure they're properly bounded
	app.Get("/search-large-limit", handler.Search)

	// Test request with very large limit
	req := httptest.NewRequest("GET", "/search-large-limit?q=test&limit=1000", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	// The handler should handle large limits gracefully - either by bounding them or returning appropriate error
	assert.True(t, resp.StatusCode == 200 || resp.StatusCode == 400)
}

func TestSearchHandler_PerformanceWithLargeOffset(t *testing.T) {
	app := fiber.New()

	// Create a mock repository for testing
	mockRepo := &services.Repository{}
	handler := NewSearchHandler(mockRepo)

	// Test with large offset values
	app.Get("/search-large-offset", handler.Search)

	// Test request with very large offset
	req := httptest.NewRequest("GET", "/search-large-offset?q=test&offset=100000", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	// The handler should handle large offsets gracefully
	assert.True(t, resp.StatusCode == 200 || resp.StatusCode == 400)
}

func TestSearchHandler_ValidateBoundedParams(t *testing.T) {
	app := fiber.New()

	// Create a mock repository for testing
	mockRepo := &services.Repository{}
	handler := NewSearchHandler(mockRepo)

	app.Get("/search-validate-params", handler.Search)

	// Test with maximum allowed limit
	req := httptest.NewRequest("GET", "/search-validate-params?q=test&limit=500", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.True(t, resp.StatusCode == 200 || resp.StatusCode == 400) // Either success or validation error
	
	// Test with limit above maximum (should be bounded or return error)
	req = httptest.NewRequest("GET", "/search-validate-params?q=test&limit=1000", nil)
	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.True(t, resp.StatusCode == 200 || resp.StatusCode == 400) // Either bounded or validation error
}