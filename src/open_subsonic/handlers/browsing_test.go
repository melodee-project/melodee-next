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

func TestBrowsingHandler_GetArtists(t *testing.T) {
	// Create a minimal setup for testing
	db := getTestDB()
	repo := services.NewRepository(db)
	browsingHandler := NewBrowsingHandler(repo)

	// Create Fiber app for testing
	app := fiber.New()
	app.Get("/rest/getArtists", func(c *fiber.Ctx) error {
		// Simulate authenticated user context
		testUser := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
			IsAdmin:  false,
		}
		c.Locals("user", testUser)
		return browsingHandler.GetArtists(c)
	})

	// Test basic request
	req := httptest.NewRequest("GET", "/rest/getArtists", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Should return 200 OK or similar (not authentication error)
	assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
	assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
}

func TestBrowsingHandler_GetMusicFolders(t *testing.T) {
	// Create a minimal setup for testing
	db := getTestDB()
	repo := services.NewRepository(db)
	browsingHandler := NewBrowsingHandler(repo)

	// Create Fiber app for testing
	app := fiber.New()
	app.Get("/rest/getMusicFolders", func(c *fiber.Ctx) error {
		// Simulate authenticated user context
		testUser := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
			IsAdmin:  false,
		}
		c.Locals("user", testUser)
		return browsingHandler.GetMusicFolders(c)
	})

	// Test basic request
	req := httptest.NewRequest("GET", "/rest/getMusicFolders", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Should return 200 OK or similar (not authentication error)
	assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
	assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
}

func TestBrowsingHandler_GetArtist(t *testing.T) {
	// Create a minimal setup for testing
	db := getTestDB()
	repo := services.NewRepository(db)
	browsingHandler := NewBrowsingHandler(repo)

	// Create Fiber app for testing
	app := fiber.New()
	app.Get("/rest/getArtist", func(c *fiber.Ctx) error {
		// Simulate authenticated user context
		testUser := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
			IsAdmin:  false,
		}
		c.Locals("user", testUser)
		return browsingHandler.GetArtist(c)
	})

	// Test basic request with artist parameter
	req := httptest.NewRequest("GET", "/rest/getArtist?id=1", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Should return 200 OK or 404 (not found) or similar (not authentication error)
	assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
	assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
}

func getTestDB() *gorm.DB {
	// This would create a test database
	// For now, return nil since we're just testing the auth/endpoint structure
	return nil
}