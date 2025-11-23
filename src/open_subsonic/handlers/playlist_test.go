package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"melodee/internal/models"
	"melodee/internal/services"
)

func TestPlaylistHandler_GetPlaylists(t *testing.T) {
	// Create a minimal setup for testing
	db := getTestDB()
	repo := services.NewRepository(db)
	playlistHandler := NewPlaylistHandler(repo)

	// Create Fiber app for testing
	app := fiber.New()
	app.Get("/rest/getPlaylists", func(c *fiber.Ctx) error {
		// Simulate authenticated user context
		testUser := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
			IsAdmin:  false,
		}
		c.Locals("user", testUser)
		return playlistHandler.GetPlaylists(c)
	})

	// Test basic request
	req := httptest.NewRequest("GET", "/rest/getPlaylists", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Should return 200 OK or similar (not authentication error)
	assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
	assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
}

func TestPlaylistHandler_GetPlaylist(t *testing.T) {
	// Create a minimal setup for testing
	db := getTestDB()
	repo := services.NewRepository(db)
	playlistHandler := NewPlaylistHandler(repo)

	// Create Fiber app for testing
	app := fiber.New()
	app.Get("/rest/getPlaylist", func(c *fiber.Ctx) error {
		// Simulate authenticated user context
		testUser := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
			IsAdmin:  false,
		}
		c.Locals("user", testUser)
		return playlistHandler.GetPlaylist(c)
	})

	// Test basic request with playlist id
	req := httptest.NewRequest("GET", "/rest/getPlaylist?id=1", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Should return 200 OK or 404 (not found) or similar (not authentication error)
	assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
	assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
}

func TestPlaylistHandler_CreatePlaylist(t *testing.T) {
	// Create a minimal setup for testing
	db := getTestDB()
	repo := services.NewRepository(db)
	playlistHandler := NewPlaylistHandler(repo)

	// Create Fiber app for testing
	app := fiber.New()
	app.Get("/rest/createPlaylist", func(c *fiber.Ctx) error {
		// Simulate authenticated user context
		testUser := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
			IsAdmin:  false,
		}
		c.Locals("user", testUser)
		return playlistHandler.CreatePlaylist(c)
	})

	// Test basic request
	req := httptest.NewRequest("GET", "/rest/createPlaylist", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Should return appropriate status (not authentication error)
	assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
	assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
}

func TestPlaylistHandler_UpdatePlaylist(t *testing.T) {
	// Create a minimal setup for testing
	db := getTestDB()
	repo := services.NewRepository(db)
	playlistHandler := NewPlaylistHandler(repo)

	// Create Fiber app for testing
	app := fiber.New()
	app.Get("/rest/updatePlaylist", func(c *fiber.Ctx) error {
		// Simulate authenticated user context
		testUser := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
			IsAdmin:  false,
		}
		c.Locals("user", testUser)
		return playlistHandler.UpdatePlaylist(c)
	})

	// Test basic request
	req := httptest.NewRequest("GET", "/rest/updatePlaylist", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Should return appropriate status (not authentication error)
	assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
	assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
}

func TestPlaylistHandler_DeletePlaylist(t *testing.T) {
	// Create a minimal setup for testing
	db := getTestDB()
	repo := services.NewRepository(db)
	playlistHandler := NewPlaylistHandler(repo)

	// Create Fiber app for testing
	app := fiber.New()
	app.Get("/rest/deletePlaylist", func(c *fiber.Ctx) error {
		// Simulate authenticated user context
		testUser := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
			IsAdmin:  false,
		}
		c.Locals("user", testUser)
		return playlistHandler.DeletePlaylist(c)
	})

	// Test basic request with playlist id
	req := httptest.NewRequest("GET", "/rest/deletePlaylist?id=1", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Should return appropriate status (not authentication error)
	assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
	assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
}

func getTestDB() *gorm.DB {
	// This would create a test database
	// For now, return nil since we're just testing the auth/endpoint structure
	return nil
}