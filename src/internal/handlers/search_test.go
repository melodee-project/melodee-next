package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"melodee/internal/services"
	"melodee/internal/test"
)

func TestSearchHandler_Search(t *testing.T) {
	// Create a test database instance
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Initialize repository with test db
	repo := services.NewRepository(db)

	// Create search handler
	handler := NewSearchHandler(repo)

	app := fiber.New()
	app.Get("/search", handler.Search)

	// Test search with missing query parameter
	req := httptest.NewRequest("GET", "/search", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	// Test search with valid parameters
	req = httptest.NewRequest("GET", "/search?q=test&type=artist", nil)
	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Test search with invalid type
	req = httptest.NewRequest("GET", "/search?q=test&type=invalid", nil)
	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	// Test search with default 'any' type
	req = httptest.NewRequest("GET", "/search?q=test", nil)
	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestSearchHandler_SearchPagination(t *testing.T) {
	// Create a test database instance
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Initialize repository with test db
	repo := services.NewRepository(db)

	// Create search handler
	handler := NewSearchHandler(repo)

	app := fiber.New()
	app.Get("/search", handler.Search)

	// Test with custom offset and limit
	req := httptest.NewRequest("GET", "/search?q=test&offset=10&limit=25", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	
	// Test with large limit (should be capped)
	req = httptest.NewRequest("GET", "/search?q=test&limit=600", nil)
	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestSearchHandler_SearchSong(t *testing.T) {
	// Create a test database instance
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Initialize repository with test db
	repo := services.NewRepository(db)

	// Create search handler
	handler := NewSearchHandler(repo)

	app := fiber.New()
	app.Get("/search", handler.Search)

	// Test song search
	req := httptest.NewRequest("GET", "/search?q=test&type=song", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Verify response structure
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	
	// Check expected fields are present
	assert.Contains(t, response, "data")
	assert.Contains(t, response, "pagination")
}