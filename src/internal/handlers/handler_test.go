package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"melodee/internal/services"
)

func TestAuthHandler_Login(t *testing.T) {
	// Create a new Fiber app for testing
	app := fiber.New()

	// Create a mock auth service
	authService := services.NewAuthService(nil, "test-secret-key")
	authHandler := NewAuthHandler(authService)

	// Define the route
	app.Post("/api/auth/login", authHandler.Login)

	// Test payload
	payload := map[string]string{
		"username": "testuser",
		"password": "testpassword123!",
	}
	jsonPayload, _ := json.Marshal(payload)

	// Create request
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	// Perform request
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// Check status code
	// This will be 500 because the DB is nil, which is expected in this test
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestAuthHandler_Refresh(t *testing.T) {
	// Create a new Fiber app for testing
	app := fiber.New()

	// Create a mock auth service
	authService := services.NewAuthService(nil, "test-secret-key")
	authHandler := NewAuthHandler(authService)

	// Define the route
	app.Post("/api/auth/refresh", authHandler.Refresh)

	// Test payload
	payload := map[string]string{
		"refresh_token": "some-refresh-token",
	}
	jsonPayload, _ := json.Marshal(payload)

	// Create request
	req := httptest.NewRequest("POST", "/api/auth/refresh", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	// Perform request
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// Check status code - this should return 401 for invalid token
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestUserHandler_GetUsers(t *testing.T) {
	// Create a new Fiber app for testing
	app := fiber.New()

	// Create mock services
	repo := services.NewRepository(nil)
	authService := services.NewAuthService(nil, "test-secret-key")
	userHandler := NewUserHandler(repo, authService)

	// Define the route
	app.Get("/api/users", userHandler.GetUsers)

	// Create request
	req := httptest.NewRequest("GET", "/api/users", nil)

	// Perform request
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// Without auth middleware, this should return 401
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestUserHandler_CreateUser(t *testing.T) {
	// Create a new Fiber app for testing
	app := fiber.New()

	// Create mock services
	repo := services.NewRepository(nil)
	authService := services.NewAuthService(nil, "test-secret-key")
	userHandler := NewUserHandler(repo, authService)

	// Define the route
	app.Post("/api/users", userHandler.CreateUser)

	// Test payload
	payload := map[string]interface{}{
		"username": "newuser",
		"email":    "newuser@example.com",
		"password": "NewPass123!",
		"is_admin": false,
	}
	jsonPayload, _ := json.Marshal(payload)

	// Create request
	req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	// Perform request
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// Without auth middleware, this should return 401
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestPlaylistHandler_GetPlaylists(t *testing.T) {
	// Create a new Fiber app for testing
	app := fiber.New()

	// Create mock services
	repo := services.NewRepository(nil)
	playlistHandler := NewPlaylistHandler(repo)

	// Define the route
	app.Get("/api/playlists", playlistHandler.GetPlaylists)

	// Create request
	req := httptest.NewRequest("GET", "/api/playlists", nil)

	// Perform request
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// Without auth middleware, this should return 401
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestPlaylistHandler_CreatePlaylist(t *testing.T) {
	// Create a new Fiber app for testing
	app := fiber.New()

	// Create mock services
	repo := services.NewRepository(nil)
	playlistHandler := NewPlaylistHandler(repo)

	// Define the route
	app.Post("/api/playlists", playlistHandler.CreatePlaylist)

	// Test payload
	payload := map[string]interface{}{
		"name":    "Test Playlist",
		"comment": "A test playlist",
		"public":  false,
		"song_ids": []int64{1, 2, 3},
	}
	jsonPayload, _ := json.Marshal(payload)

	// Create request
	req := httptest.NewRequest("POST", "/api/playlists", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	// Perform request
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// Without auth middleware, this should return 401
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}