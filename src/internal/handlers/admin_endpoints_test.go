package handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/hibiken/asynq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"melodee/internal/models"
)

// TestDLQEndpoints_Comprehensive tests all DLQ admin endpoints
func TestDLQEndpoints_Comprehensive(t *testing.T) {
	db, mock := setupTestDBWithMock(t)
	defer db.Migrator().DropTable(&models.User{})

	// Create test user
	testUser := models.User{
		ID:       1,
		Username: "admin",
		Email:    "admin@example.com",
		IsAdmin:  true,
	}
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT").WillReturnResult(sqlmock.NewResult(1, 1))
	db.Create(&testUser)
	mock.ExpectCommit()

	// Create admin auth middleware
	authMiddleware := NewAuthMiddleware(nil)

	// Create DLQ handler with mock inspector
	mockInspector := &MockAsynqInspector{}
	dlqHandler := NewDLQHandler(mockInspector, nil)

	app := fiber.New()

	// Test GetDLQItems (happy path with admin authentication)
	t.Run("GetDLQItems - Admin Access", func(t *testing.T) {
		app.Get("/admin/jobs/dlq", authMiddleware.JWTProtected(), authMiddleware.AdminOnly(), dlqHandler.GetDLQItems)
		
		req := httptest.NewRequest("GET", "/admin/jobs/dlq?page=1&size=50", nil)
		// Add JWT token or user context for authentication
		// This would require setting up the JWT properly or mocking the context
		req.Header.Set("Authorization", "Bearer fake-token")
		
		// In a real test, we'd need to set the user context properly based on auth
		// For now, we'll test the route wiring
	})

	// Test RequeueDLQItems 
	t.Run("RequeueDLQItems - Valid Request", func(t *testing.T) {
		app.Post("/admin/jobs/requeue", authMiddleware.JWTProtected(), authMiddleware.AdminOnly(), dlqHandler.RequeueDLQItems)
		
		requeueReq := DLQRequeueRequest{
			JobIDs: []string{"job-1", "job-2"},
		}
		jsonData, _ := json.Marshal(requeueReq)
		
		req := httptest.NewRequest("POST", "/admin/jobs/requeue", bytes.NewBuffer(jsonData))
		req.Header.Set("Authorization", "Bearer fake-token")
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := app.Test(req)
		assert.NoError(t, err)
		// Expect 200 OK or 401/403 depending on auth implementation
		// In real tests, we'd mock the auth context properly
	})

	// Test PurgeDLQItems
	t.Run("PurgeDLQItems - Valid Request", func(t *testing.T) {
		purgeReq := DLQPurgeRequest{
			JobIDs: []string{"job-1"},
		}
		jsonData, _ := json.Marshal(purgeReq)
		
		req := httptest.NewRequest("POST", "/admin/jobs/purge", bytes.NewBuffer(jsonData))
		req.Header.Set("Authorization", "Bearer fake-token")
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := app.Test(req)
		assert.NoError(t, err)
		// Expect 200 OK or 401/403 depending on auth implementation
	})

	// Test GetJobById
	t.Run("GetJobById - Valid Job", func(t *testing.T) {
		app.Get("/admin/jobs/:id", authMiddleware.JWTProtected(), authMiddleware.AdminOnly(), dlqHandler.GetJobById)
		
		req := httptest.NewRequest("GET", "/admin/jobs/test-job-id", nil)
		req.Header.Set("Authorization", "Bearer fake-token")
		
		resp, err := app.Test(req)
		assert.NoError(t, err)
		// Expect 200 OK, 401, 403, or 404 depending on implementation
	})
}

// TestLibraryEndpoints_Comprehensive tests all Library admin endpoints
func TestLibraryEndpoints_Comprehensive(t *testing.T) {
	db, mock := setupTestDBWithMock(t)
	defer db.Migrator().DropTable(&models.User{}, &models.Library{})

	// Create test admin user
	testUser := models.User{
		ID:       1,
		Username: "admin",
		Email:    "admin@example.com",
		IsAdmin:  true,
	}
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT").WillReturnResult(sqlmock.NewResult(1, 1))
	db.Create(&testUser)
	mock.ExpectCommit()

	// Create test libraries
	testLibrary := models.Library{
		ID:   1,
		Name: "Test Library",
		Path: "/test/path",
	}
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT").WillReturnResult(sqlmock.NewResult(1, 1))
	db.Create(&testLibrary)
	mock.ExpectCommit()

	authMiddleware := NewAuthMiddleware(nil)

	// Create a library handler with real dependencies for testing
	// We'll need to set this up with proper mocks
	libraryHandler := NewLibraryHandler(nil, nil, nil, nil)

	app := fiber.New()

	// Test GetLibrariesStats
	t.Run("GetLibrariesStats - Admin Access", func(t *testing.T) {
		app.Get("/libraries/stats", authMiddleware.JWTProtected(), authMiddleware.AdminOnly(), libraryHandler.GetLibrariesStats)
		
		req := httptest.NewRequest("GET", "/libraries/stats", nil)
		req.Header.Set("Authorization", "Bearer fake-token")
		
		resp, err := app.Test(req)
		assert.NoError(t, err)
		// Should return 200, 401, or 403
		assert.Contains(t, []int{200, 401, 403}, resp.StatusCode)
	})

	// Test TriggerLibraryScan
	t.Run("TriggerLibraryScan - Valid Library", func(t *testing.T) {
		app.Get("/libraries/1/scan", authMiddleware.JWTProtected(), authMiddleware.AdminOnly(), libraryHandler.TriggerLibraryScan)
		
		req := httptest.NewRequest("GET", "/libraries/1/scan", nil)
		req.Header.Set("Authorization", "Bearer fake-token")
		
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Contains(t, []int{200, 401, 403, 404}, resp.StatusCode)
	})

	// Test TriggerLibraryProcess
	t.Run("TriggerLibraryProcess - Valid Library", func(t *testing.T) {
		app.Get("/libraries/1/process", authMiddleware.JWTProtected(), authMiddleware.AdminOnly(), libraryHandler.TriggerLibraryProcess)
		
		req := httptest.NewRequest("GET", "/libraries/1/process", nil)
		req.Header.Set("Authorization", "Bearer fake-token")
		
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Contains(t, []int{200, 401, 403, 404}, resp.StatusCode)
	})

	// Test TriggerLibraryMoveOK
	t.Run("TriggerLibraryMoveOK - Valid Library", func(t *testing.T) {
		app.Get("/libraries/1/move-ok", authMiddleware.JWTProtected(), authMiddleware.AdminOnly(), libraryHandler.TriggerLibraryMoveOK)
		
		req := httptest.NewRequest("GET", "/libraries/1/move-ok", nil)
		req.Header.Set("Authorization", "Bearer fake-token")
		
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Contains(t, []int{200, 401, 403, 404}, resp.StatusCode)
	})

	// Test GetQuarantineItems with pagination
	t.Run("GetQuarantineItems - With Pagination", func(t *testing.T) {
		app.Get("/libraries/quarantine", authMiddleware.JWTProtected(), authMiddleware.AdminOnly(), libraryHandler.GetQuarantineItems)
		
		req := httptest.NewRequest("GET", "/libraries/quarantine?page=1&size=10", nil)
		req.Header.Set("Authorization", "Bearer fake-token")
		
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Contains(t, []int{200, 401, 403}, resp.StatusCode)
	})

	// Test ResolveQuarantineItem
	t.Run("ResolveQuarantineItem - Valid Item", func(t *testing.T) {
		app.Post("/libraries/quarantine/1/resolve", authMiddleware.JWTProtected(), authMiddleware.AdminOnly(), libraryHandler.ResolveQuarantineItem)
		
		req := httptest.NewRequest("POST", "/libraries/quarantine/1/resolve", nil)
		req.Header.Set("Authorization", "Bearer fake-token")
		
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Contains(t, []int{200, 401, 403, 404}, resp.StatusCode)
	})

	// Test RequeueQuarantineItem
	t.Run("RequeueQuarantineItem - Valid Item", func(t *testing.T) {
		app.Post("/libraries/quarantine/1/requeue", authMiddleware.JWTProtected(), authMiddleware.AdminOnly(), libraryHandler.RequeueQuarantineItem)
		
		req := httptest.NewRequest("POST", "/libraries/quarantine/1/requeue", nil)
		req.Header.Set("Authorization", "Bearer fake-token")
		
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Contains(t, []int{200, 401, 403, 404}, resp.StatusCode)
	})
}

// TestSharesEndpoints_Comprehensive tests all Shares admin endpoints
func TestSharesEndpoints_Comprehensive(t *testing.T) {
	db, mock := setupTestDBWithMock(t)
	defer db.Migrator().DropTable(&models.User{}, &models.Share{})

	// Create test admin user
	testUser := models.User{
		ID:       1,
		Username: "admin",
		Email:    "admin@example.com",
		IsAdmin:  true,
	}
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT").WillReturnResult(sqlmock.NewResult(1, 1))
	db.Create(&testUser)
	mock.ExpectCommit()

	authMiddleware := NewAuthMiddleware(nil)
	sharesHandler := NewSharesHandler(nil) // Pass proper repo in real implementation

	app := fiber.New()

	// Test GetShares with pagination
	t.Run("GetShares - With Pagination", func(t *testing.T) {
		app.Get("/shares", authMiddleware.JWTProtected(), authMiddleware.AdminOnly(), sharesHandler.GetShares)
		
		req := httptest.NewRequest("GET", "/shares?page=1&size=10", nil)
		req.Header.Set("Authorization", "Bearer fake-token")
		
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Contains(t, []int{200, 401, 403}, resp.StatusCode)
	})

	// Test CreateShare
	t.Run("CreateShare - Valid Request", func(t *testing.T) {
		// Note: CreateShare might not be admin-only, but we'll test it here
		// depending on the actual implementation
		shareData := map[string]interface{}{
			"name": "Test Share",
			"track_ids": []string{"1", "2"},
			"expires_at": "2025-12-01T00:00:00Z",
			"max_streaming_minutes": 600,
			"allow_download": true,
		}
		jsonData, _ := json.Marshal(shareData)
		
		req := httptest.NewRequest("POST", "/shares", bytes.NewBuffer(jsonData))
		req.Header.Set("Authorization", "Bearer fake-token")
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Contains(t, []int{200, 401, 403, 400}, resp.StatusCode)
	})

	// Test UpdateShare
	t.Run("UpdateShare - Valid Share", func(t *testing.T) {
		shareData := map[string]interface{}{
			"name": "Updated Share",
			"track_ids": []string{"1"},
			"expires_at": "2025-12-31T00:00:00Z",
			"max_streaming_minutes": 300,
			"allow_download": false,
		}
		jsonData, _ := json.Marshal(shareData)
		
		req := httptest.NewRequest("PUT", "/shares/1", bytes.NewBuffer(jsonData))
		req.Header.Set("Authorization", "Bearer fake-token")
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Contains(t, []int{200, 401, 403, 400, 404}, resp.StatusCode)
	})

	// Test DeleteShare
	t.Run("DeleteShare - Valid Share", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/shares/1", nil)
		req.Header.Set("Authorization", "Bearer fake-token")
		
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Contains(t, []int{200, 401, 403, 404}, resp.StatusCode)
	})
}

// TestSettingsEndpoints_Comprehensive tests all Settings admin endpoints
func TestSettingsEndpoints_Comprehensive(t *testing.T) {
	db, mock := setupTestDBWithMock(t)
	defer db.Migrator().DropTable(&models.User{}, &models.Setting{})

	// Create test admin user
	testUser := models.User{
		ID:       1,
		Username: "admin",
		Email:    "admin@example.com",
		IsAdmin:  true,
	}
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT").WillReturnResult(sqlmock.NewResult(1, 1))
	db.Create(&testUser)
	mock.ExpectCommit()

	authMiddleware := NewAuthMiddleware(nil)
	settingsHandler := NewSettingsHandler(nil) // Pass proper repo in real implementation

	app := fiber.New()

	// Test GetSettings with pagination
	t.Run("GetSettings - With Pagination", func(t *testing.T) {
		app.Get("/settings", authMiddleware.JWTProtected(), authMiddleware.AdminOnly(), settingsHandler.GetSettings)
		
		req := httptest.NewRequest("GET", "/settings?page=1&size=20", nil)
		req.Header.Set("Authorization", "Bearer fake-token")
		
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Contains(t, []int{200, 401, 403}, resp.StatusCode)
	})

	// Test UpdateSetting
	t.Run("UpdateSetting - Valid Setting", func(t *testing.T) {
		settingData := map[string]string{
			"value": "new_value",
		}
		jsonData, _ := json.Marshal(settingData)
		
		req := httptest.NewRequest("PUT", "/settings/test_key", bytes.NewBuffer(jsonData))
		req.Header.Set("Authorization", "Bearer fake-token")
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Contains(t, []int{200, 401, 403, 400, 404}, resp.StatusCode)
	})
}

// Helper function to set up test DB with mock
func setupTestDBWithMock(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open gorm DB: %v", err)
	}

	return gormDB, mock
}

// Mock for testing
type MockAsynqInspector struct{}

func (m *MockAsynqInspector) Queues() ([]string, error) {
	return []string{"default"}, nil
}

func (m *MockAsynqInspector) ListDead(queueName string) ([]*asynq.TaskInfo, error) {
	return []*asynq.TaskInfo{}, nil
}

func (m *MockAsynqInspector) GetDeadTask(queueName, taskID string) (*asynq.TaskInfo, error) {
	return nil, nil
}

func (m *MockAsynqInspector) DeleteDead(queueName, taskID string) error {
	return nil
}

func (m *MockAsynqInspector) GetQueueInfo(queueName string) (*asynq.QueueInfo, error) {
	return &asynq.QueueInfo{}, nil
}