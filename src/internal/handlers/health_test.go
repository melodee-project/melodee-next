package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"melodee/internal/database"
)

func TestHealthHandler_HealthCheck(t *testing.T) {
	// Create a new Fiber app for testing
	app := fiber.New()

	// Create a mock database manager
	dbManager := database.NewDatabaseManagerFromExisting(nil, nil)
	healthHandler := NewHealthHandler(dbManager)

	// Define the route
	app.Get("/healthz", healthHandler.HealthCheck)

	// Create request
	req := httptest.NewRequest("GET", "/healthz", nil)

	// Perform request
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// Check status code - this will likely be 500 because dbManager is empty
	// But it should at least execute the function without panicking
	assert.Condition(t, func() bool {
		return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusServiceUnavailable
	})

	// Check headers
	assert.Equal(t, "no-store", resp.Header.Get("Cache-Control"))
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	// Check response body is valid JSON
	var healthResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&healthResponse)
	assert.NoError(t, err)
}