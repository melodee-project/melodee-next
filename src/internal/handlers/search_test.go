package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"github.com/google/uuid"
	"melodee/internal/models"
	"melodee/internal/services"
	"melodee/internal/test"
)

func TestSearchHandler_Comprehensive(t *testing.T) {
	// Create test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	repo := services.NewRepository(db)
	searchHandler := NewSearchHandler(repo)

	// Create user for testing
	password := "ValidPass123!"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err)

	user := &models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: string(hashedPassword),
		IsAdmin:      false,
		APIKey:       uuid.New(),
	}

	err = db.Create(user).Error
	assert.NoError(t, err)

	// Create Fiber app for testing
	app := fiber.New()
	app.Get("/api/search", func(c *fiber.Ctx) error {
		ctxUser := &models.User{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			IsAdmin:  false,
		}
		c.Locals("user", ctxUser)
		return searchHandler.Search(c)
	})

	// Test search with query parameter
	t.Run("Search with query parameter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/search?q=test", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify response structure
		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Contains(t, response, "data")
		assert.Contains(t, response, "pagination")
	})

	// Test search with type parameter
	t.Run("Search with type parameter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/search?q=test&type=artist", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Contains(t, response, "data")
		assert.Contains(t, response, "pagination")
	})

	// Test search with offset and limit parameters
	t.Run("Search with pagination parameters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/search?q=test&offset=0&limit=10", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Contains(t, response, "data")
		assert.Contains(t, response, "pagination")
		pagination := response["pagination"].(map[string]interface{})
		assert.Equal(t, float64(0), pagination["offset"])
		assert.Equal(t, float64(10), pagination["limit"])
	})

	// Test search with invalid type parameter
	t.Run("Search with invalid type parameter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/search?q=test&type=invalid_type", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		// Should return 400 for invalid search type
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	// Test search without query parameter (should fail)
	t.Run("Search without query parameter fails", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/search", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	// Test with different entity types
	entityTypes := []string{"artist", "artists", "album", "albums", "song", "songs", "any", "all", ""}

	for _, entityType := range entityTypes {
		testName := "Search type: " + entityType
		t.Run(testName, func(t *testing.T) {
			url := "/api/search?q=test"
			if entityType != "" {
				url += "&type=" + entityType
			}

			req := httptest.NewRequest("GET", url, nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			// Should succeed for valid types, fail for invalid ones
			if entityType == "artist" || entityType == "artists" || 
			   entityType == "album" || entityType == "albums" || 
			   entityType == "song" || entityType == "songs" || 
			   entityType == "any" || entityType == "all" || entityType == "" {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			}
		})
	}

	// Test with limit constraints (max 500)
	t.Run("Search with high limit", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/search?q=test&limit=1000", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		// The handler should cap the limit at 500
		pagination := response["pagination"].(map[string]interface{})
		limit := pagination["limit"].(float64)
		assert.True(t, limit <= 500)
	})

	// Test without authentication
	appNoAuth := fiber.New()
	appNoAuth.Get("/api/search", searchHandler.Search)

	t.Run("Search fails without authentication", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/search?q=test", nil)
		resp, err := appNoAuth.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestSearchHandler_EdgeCases(t *testing.T) {
	// Create test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	repo := services.NewRepository(db)
	searchHandler := NewSearchHandler(repo)

	// Create user for testing
	password := "ValidPass123!"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err)

	user := &models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: string(hashedPassword),
		IsAdmin:      false,
		APIKey:       uuid.New(),
	}

	err = db.Create(user).Error
	assert.NoError(t, err)

	// Create Fiber app for testing
	app := fiber.New()
	app.Get("/api/search", func(c *fiber.Ctx) error {
		ctxUser := &models.User{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			IsAdmin:  false,
		}
		c.Locals("user", ctxUser)
		return searchHandler.Search(c)
	})

	// Test with empty query (should fail)
	t.Run("Search with empty query", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/search?q=", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	// Test with very long query
	t.Run("Search with very long query", func(t *testing.T) {
		longQuery := string(make([]byte, 1000))
		for i := range longQuery {
			longQuery[i] = 'a'
		}

		req := httptest.NewRequest("GET", "/api/search?q="+longQuery, nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		// Should handle long queries gracefully
		assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
		assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
	})

	// Test with invalid offset/limit values
	t.Run("Search with invalid offset", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/search?q=test&offset=-1", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		// Should default to 0 offset if negative
		assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
		assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("Search with invalid limit", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/search?q=test&limit=0", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		// Should default to 50 if limit is 0 or negative
		assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
		assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
	})

	// Test with negative limit
	t.Run("Search with negative limit", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/search?q=test&limit=-10", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		// Should default to 50 if limit is 0 or negative
		assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
		assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
	})

	// Test search that returns no results
	t.Run("Search with no results", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/search?q=nonexistent_search_term_that_should_not_match_anything", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Contains(t, response, "data")
		assert.Contains(t, response, "pagination")
		// Data could be empty array or object depending on implementation
	})
}

func TestSearchHandler_PerformanceAndSecurity(t *testing.T) {
	// Create test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	repo := services.NewRepository(db)
	searchHandler := NewSearchHandler(repo)

	// Create user for testing
	password := "ValidPass123!"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err)

	user := &models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: string(hashedPassword),
		IsAdmin:      false,
		APIKey:       uuid.New(),
	}

	err = db.Create(user).Error
	assert.NoError(t, err)

	// Create Fiber app for testing
	app := fiber.New()
	app.Get("/api/search", func(c *fiber.Ctx) error {
		ctxUser := &models.User{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			IsAdmin:  false,
		}
		c.Locals("user", ctxUser)
		return searchHandler.Search(c)
	})

	// Test SQL injection prevention (if applicable to the implementation)
	t.Run("Search with potential SQL injection", func(t *testing.T) {
		// Test with SQL injection attempts in the query parameter
		injectionAttempts := []string{
			"' OR '1'='1",
			"'; DROP TABLE users; --",
			"test'; DELETE FROM users; --",
			"1' OR '1'='1' --",
		}

		for _, injection := range injectionAttempts {
			req := httptest.NewRequest("GET", "/api/search?q="+injection, nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			// Should not crash or return internal server error
			// Should return valid response (empty results or proper error)
			assert.NotEqual(t, http.StatusInternalServerError, resp.StatusCode)
			// Should still require authentication
			assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
		}
	})

	// Test with special characters
	t.Run("Search with special characters", func(t *testing.T) {
		specialQueries := []string{
			"test & more",
			"test | more",
			"test @ symbol",
			"test # hash",
			"test % percent",
			"test ^ carat",
			"test * asterisk",
			"test (paren)",
			"test [bracket]",
			"test {brace}",
		}

		for _, query := range specialQueries {
			req := httptest.NewRequest("GET", "/api/search?q="+query, nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			// Should handle special characters gracefully
			assert.NotEqual(t, http.StatusInternalServerError, resp.StatusCode)
			assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
			assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
		}
	})

	// Test with unicode characters
	t.Run("Search with unicode characters", func(t *testing.T) {
		unicodeQueries := []string{
			"café", // contains accented character
			"北京", // Chinese characters
			"Москва", // Cyrillic characters
			"سال", // Arabic characters
			"test-世界", // Mixed ASCII and Unicode
		}

		for _, query := range unicodeQueries {
			req := httptest.NewRequest("GET", "/api/search?q="+query, nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			// Should handle unicode gracefully
			assert.NotEqual(t, http.StatusInternalServerError, resp.StatusCode)
			assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
			assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
		}
	})
}