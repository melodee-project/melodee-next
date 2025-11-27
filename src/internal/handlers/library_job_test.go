package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"melodee/internal/models"
	"melodee/internal/services"
	"melodee/internal/test"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestLibraryHandler_GetLibrariesStats(t *testing.T) {
	// Create test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	repo := services.NewRepository(db)
	libraryHandler := NewLibraryHandler(repo, nil, nil, nil)

	// Create user for testing
	password := "ValidPass123!"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err)

	user := &models.User{
		Username:     "admin",
		Email:        "admin@example.com",
		PasswordHash: string(hashedPassword),
		IsAdmin:      true,
		APIKey:       uuid.New(),
	}

	err = db.Create(user).Error
	assert.NoError(t, err)

	// Create test libraries
	library1 := &models.Library{
		Name:       "Test Library 1",
		Type:       "production",
		Path:       "/music/test1",
		IsLocked:   false,
		SongCount:  100,
		AlbumCount: 50,
		Duration:   3600000, // 1 hour in milliseconds
		CreatedAt:  time.Now(),
	}

	err = db.Create(library1).Error
	assert.NoError(t, err)

	// Create Fiber app for testing
	app := fiber.New()
	app.Get("/api/libraries/stats", func(c *fiber.Ctx) error {
		// Set admin user context for testing (simulating middleware)
		ctxUser := &models.User{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			IsAdmin:  true,
		}
		c.Locals("user", ctxUser)
		return libraryHandler.GetLibrariesStats(c)
	})

	// Test successful retrieval of library stats
	t.Run("Get library stats successfully", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/libraries/stats", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify response structure
		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Contains(t, response, "total_libraries")
		assert.Contains(t, response, "total_artists")
		assert.Contains(t, response, "total_albums")
		assert.Contains(t, response, "total_tracks")
		assert.Contains(t, response, "total_size_bytes")
		assert.Contains(t, response, "last_full_scan_at")
	})

	// Test with non-admin user (should fail)
	appNonAdmin := fiber.New()
	appNonAdmin.Get("/api/libraries/stats", func(c *fiber.Ctx) error {
		// Set non-admin user context
		nonAdminUser := &models.User{
			ID:       user.ID + 1,
			Username: "regular",
			Email:    "regular@example.com",
			IsAdmin:  false,
		}
		c.Locals("user", nonAdminUser)
		return libraryHandler.GetLibrariesStats(c)
	})

	t.Run("Non-admin cannot access library stats", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/libraries/stats", nil)
		resp, err := appNonAdmin.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}

func TestLibraryHandler_LibraryOperations(t *testing.T) {
	// Create test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	repo := services.NewRepository(db)
	libraryHandler := NewLibraryHandler(repo, nil, nil, nil)

	// Create admin user for testing
	password := "ValidPass123!"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err)

	adminUser := &models.User{
		Username:     "admin",
		Email:        "admin@example.com",
		PasswordHash: string(hashedPassword),
		IsAdmin:      true,
		APIKey:       uuid.New(),
	}

	err = db.Create(adminUser).Error
	assert.NoError(t, err)

	// Create test library
	library := &models.Library{
		Name:       "Test Library",
		Type:       "production",
		Path:       "/music/test",
		IsLocked:   false,
		SongCount:  50,
		AlbumCount: 25,
		Duration:   1800000, // 30 minutes in milliseconds
		CreatedAt:  time.Now(),
	}

	err = db.Create(library).Error
	assert.NoError(t, err)

	// Test GetLibraryStates
	app := fiber.New()
	app.Get("/api/libraries", func(c *fiber.Ctx) error {
		ctxUser := &models.User{
			ID:       adminUser.ID,
			Username: adminUser.Username,
			Email:    adminUser.Email,
			IsAdmin:  true,
		}
		c.Locals("user", ctxUser)
		return libraryHandler.GetLibraryStates(c)
	})

	t.Run("Get library states successfully", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/libraries", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Contains(t, response, "data") // Should be an array of library states
	})

	// Test GetLibraryState
	appGetState := fiber.New()
	appGetState.Get("/api/libraries/:id", func(c *fiber.Ctx) error {
		ctxUser := &models.User{
			ID:       adminUser.ID,
			Username: adminUser.Username,
			Email:    adminUser.Email,
			IsAdmin:  true,
		}
		c.Locals("user", ctxUser)
		return libraryHandler.GetLibraryState(c)
	})

	t.Run("Get specific library state successfully", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/libraries/1", nil)
		resp, err := appGetState.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Contains(t, response, "id")
		assert.Contains(t, response, "name")
		assert.Contains(t, response, "path")
	})

	// Test library scan operation
	appScan := fiber.New()
	appScan.Post("/api/libraries/scan", func(c *fiber.Ctx) error {
		ctxUser := &models.User{
			ID:       adminUser.ID,
			Username: adminUser.Username,
			Email:    adminUser.Email,
			IsAdmin:  true,
		}
		c.Locals("user", ctxUser)
		return libraryHandler.TriggerLibraryScan(c)
	})

	t.Run("Library scan operation", func(t *testing.T) {
		// The scan operation requires additional parameters and mock services
		// This test focuses on the authentication/authorization part
		req := httptest.NewRequest("POST", "/api/libraries/scan", nil)
		resp, err := appScan.Test(req)
		// Could return 400 (bad request) if parameters are missing or 500 if services are nil
		assert.NoError(t, err)
		// It should not return 401/403 for auth, so check for those specifically to ensure auth works
		assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
		assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
	})

	// Test non-admin access
	appNonAdmin := fiber.New()
	appNonAdmin.Get("/api/libraries", func(c *fiber.Ctx) error {
		ctxUser := &models.User{
			ID:       adminUser.ID + 1, // Different user ID
			Username: "regular",
			Email:    "regular@example.com",
			IsAdmin:  false,
		}
		c.Locals("user", ctxUser)
		return libraryHandler.GetLibraryStates(c)
	})

	t.Run("Non-admin cannot access library operations", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/libraries", nil)
		resp, err := appNonAdmin.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}

func TestDLQHandler_Comprehensive(t *testing.T) {
	// Create test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Mock asynq inspector
	mockInspector := &MockAsynqInspector{}
	dlqHandler := NewDLQHandler(mockInspector)

	// Create admin user for testing
	password := "ValidPass123!"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err)

	adminUser := &models.User{
		Username:     "admin",
		Email:        "admin@example.com",
		PasswordHash: string(hashedPassword),
		IsAdmin:      true,
		APIKey:       uuid.New(),
	}

	err = db.Create(adminUser).Error
	assert.NoError(t, err)

	// Test GetDLQItems
	app := fiber.New()
	app.Get("/api/admin/jobs/dlq", func(c *fiber.Ctx) error {
		ctxUser := &models.User{
			ID:       adminUser.ID,
			Username: adminUser.Username,
			Email:    adminUser.Email,
			IsAdmin:  true,
		}
		c.Locals("user", ctxUser)
		return dlqHandler.GetDLQItems(c)
	})

	t.Run("Get DLQ items successfully", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/jobs/dlq", nil)
		resp, err := app.Test(req)
		// This should fail with 500 since we're using a mock inspector
		// But it should not fail with auth issues (401/403)
		assert.NoError(t, err)
		assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
		assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
	})

	// Test RequeueDLQItems
	appRequeue := fiber.New()
	appRequeue.Post("/api/admin/jobs/requeue", func(c *fiber.Ctx) error {
		ctxUser := &models.User{
			ID:       adminUser.ID,
			Username: adminUser.Username,
			Email:    adminUser.Email,
			IsAdmin:  true,
		}
		c.Locals("user", ctxUser)
		return dlqHandler.RequeueDLQItems(c)
	})

	t.Run("Requeue DLQ items", func(t *testing.T) {
		reqBody := map[string][]string{
			"job_ids": {"job-1", "job-2"},
		}

		reqBodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/admin/jobs/requeue", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := appRequeue.Test(req)
		assert.NoError(t, err)
		// Should not fail with auth issues
		assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
		assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
	})

	// Test PurgeDLQItems
	appPurge := fiber.New()
	appPurge.Post("/api/admin/jobs/purge", func(c *fiber.Ctx) error {
		ctxUser := &models.User{
			ID:       adminUser.ID,
			Username: adminUser.Username,
			Email:    adminUser.Email,
			IsAdmin:  true,
		}
		c.Locals("user", ctxUser)
		return dlqHandler.PurgeDLQItems(c)
	})

	t.Run("Purge DLQ items", func(t *testing.T) {
		reqBody := map[string][]string{
			"job_ids": {"job-1", "job-2"},
		}

		reqBodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/admin/jobs/purge", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := appPurge.Test(req)
		assert.NoError(t, err)
		// Should not fail with auth issues
		assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
		assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
	})

	// Test non-admin access
	appNonAdmin := fiber.New()
	appNonAdmin.Get("/api/admin/jobs/dlq", func(c *fiber.Ctx) error {
		ctxUser := &models.User{
			ID:       adminUser.ID + 1,
			Username: "regular",
			Email:    "regular@example.com",
			IsAdmin:  false,
		}
		c.Locals("user", ctxUser)
		return dlqHandler.GetDLQItems(c)
	})

	t.Run("Non-admin cannot access DLQ operations", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/jobs/dlq", nil)
		resp, err := appNonAdmin.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}

func TestSettingsHandler_Comprehensive(t *testing.T) {
	// Create test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	repo := services.NewRepository(db)
	settingsHandler := NewSettingsHandler(repo)

	// Create admin user for testing
	password := "ValidPass123!"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err)

	adminUser := &models.User{
		Username:     "admin",
		Email:        "admin@example.com",
		PasswordHash: string(hashedPassword),
		IsAdmin:      true,
		APIKey:       uuid.New(),
	}

	err = db.Create(adminUser).Error
	assert.NoError(t, err)

	// Test GetSettings
	app := fiber.New()
	app.Get("/api/settings", func(c *fiber.Ctx) error {
		ctxUser := &models.User{
			ID:       adminUser.ID,
			Username: adminUser.Username,
			Email:    adminUser.Email,
			IsAdmin:  true,
		}
		c.Locals("user", ctxUser)
		return settingsHandler.GetSettings(c)
	})

	t.Run("Get settings successfully", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/settings", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Contains(t, response, "data")
	})

	// Test UpdateSetting
	appUpdate := fiber.New()
	appUpdate.Put("/api/settings/:key", func(c *fiber.Ctx) error {
		ctxUser := &models.User{
			ID:       adminUser.ID,
			Username: adminUser.Username,
			Email:    adminUser.Email,
			IsAdmin:  true,
		}
		c.Locals("user", ctxUser)
		return settingsHandler.UpdateSetting(c)
	})

	t.Run("Update setting", func(t *testing.T) {
		reqBody := map[string]string{
			"value": "new_value",
		}

		reqBodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/settings/test-key", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := appUpdate.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Contains(t, response, "status")
		assert.Contains(t, response, "setting")
	})

	// Test non-admin access
	appNonAdmin := fiber.New()
	appNonAdmin.Get("/api/settings", func(c *fiber.Ctx) error {
		ctxUser := &models.User{
			ID:       adminUser.ID + 1,
			Username: "regular",
			Email:    "regular@example.com",
			IsAdmin:  false,
		}
		c.Locals("user", ctxUser)
		return settingsHandler.GetSettings(c)
	})

	t.Run("Non-admin cannot access settings", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/settings", nil)
		resp, err := appNonAdmin.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}

func TestSharesHandler_Comprehensive(t *testing.T) {
	// Create test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	repo := services.NewRepository(db)
	sharesHandler := NewSharesHandler(repo)

	// Create admin user for testing
	password := "ValidPass123!"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err)

	adminUser := &models.User{
		Username:     "admin",
		Email:        "admin@example.com",
		PasswordHash: string(hashedPassword),
		IsAdmin:      true,
		APIKey:       uuid.New(),
	}

	err = db.Create(adminUser).Error
	assert.NoError(t, err)

	// Test GetShares
	app := fiber.New()
	app.Get("/api/shares", func(c *fiber.Ctx) error {
		ctxUser := &models.User{
			ID:       adminUser.ID,
			Username: adminUser.Username,
			Email:    adminUser.Email,
			IsAdmin:  true,
		}
		c.Locals("user", ctxUser)
		return sharesHandler.GetShares(c)
	})

	t.Run("Get shares successfully", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/shares", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Contains(t, response, "data")
		assert.Contains(t, response, "pagination")
	})

	// Test CreateShare
	appCreate := fiber.New()
	appCreate.Post("/api/shares", func(c *fiber.Ctx) error {
		ctxUser := &models.User{
			ID:       adminUser.ID,
			Username: adminUser.Username,
			Email:    adminUser.Email,
			IsAdmin:  true,
		}
		c.Locals("user", ctxUser)
		return sharesHandler.CreateShare(c)
	})

	t.Run("Create share", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":                  "Test Share",
			"track_ids":             []string{"track-1", "track-2"},
			"expires_at":            time.Now().AddDate(0, 0, 30).Format(time.RFC3339),
			"max_streaming_minutes": 60,
			"allow_download":        true,
		}

		reqBodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/shares", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := appCreate.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Contains(t, response, "status")
		assert.Contains(t, response, "share")
	})

	// Test UpdateShare
	appUpdate := fiber.New()
	appUpdate.Put("/api/shares/:id", func(c *fiber.Ctx) error {
		ctxUser := &models.User{
			ID:       adminUser.ID,
			Username: adminUser.Username,
			Email:    adminUser.Email,
			IsAdmin:  true,
		}
		c.Locals("user", ctxUser)
		return sharesHandler.UpdateShare(c)
	})

	t.Run("Update share", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":                  "Updated Share",
			"track_ids":             []string{"track-1"},
			"expires_at":            time.Now().AddDate(0, 0, 60).Format(time.RFC3339),
			"max_streaming_minutes": 120,
			"allow_download":        false,
		}

		reqBodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/shares/share-1", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := appUpdate.Test(req)
		assert.NoError(t, err)
		// This will likely return 404 since we're using a mock ID
		// But it should not fail with auth issues
		assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
		assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
	})

	// Test DeleteShare
	appDelete := fiber.New()
	appDelete.Delete("/api/shares/:id", func(c *fiber.Ctx) error {
		ctxUser := &models.User{
			ID:       adminUser.ID,
			Username: adminUser.Username,
			Email:    adminUser.Email,
			IsAdmin:  true,
		}
		c.Locals("user", ctxUser)
		return sharesHandler.DeleteShare(c)
	})

	t.Run("Delete share", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/shares/share-1", nil)
		resp, err := appDelete.Test(req)
		assert.NoError(t, err)
		// This will likely return 404 since we're using a mock ID
		// But it should not fail with auth issues
		assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
		assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
	})
}
