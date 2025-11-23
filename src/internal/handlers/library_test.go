package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"melodee/internal/media"
	"melodee/internal/models"
	"melodee/internal/services"
	"melodee/internal/test"
)

func TestLibraryHandler_GetQuarantineItems(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()
	
	// Setup test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Create repository, media service, and quarantine service
	repo := services.NewRepository(db)
	quarantineSvc := media.NewQuarantineService(db, filepath.Join(tempDir, "quarantine"))

	// Create media service with the quarantine service
	mediaSvc := media.NewMediaService(db, nil, nil, quarantineSvc)

	// Create library handler with all required services
	// Note: We're using a mock Asynq client for testing
	handler := &LibraryHandler{
		repo:          repo,
		mediaSvc:      mediaSvc,
		asynqClient:   nil, // Will not be used in these tests
		quarantineSvc: quarantineSvc,
	}

	// Create Fiber app for testing
	app := fiber.New()
	app.Get("/quarantine", handler.GetQuarantineItems)

	// Test with no quarantine items
	req := httptest.NewRequest("GET", "/quarantine", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Add some quarantine items for testing
	testFile := filepath.Join(tempDir, "test_file.mp3")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	assert.NoError(t, err)

	// Quarantine the test file
	err = quarantineSvc.QuarantineFile(testFile, media.ChecksumMismatch, "Test checksum mismatch", 1)
	assert.NoError(t, err)

	// Test with quarantine items present
	req = httptest.NewRequest("GET", "/quarantine", nil)
	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Test with pagination
	req = httptest.NewRequest("GET", "/quarantine?page=1&limit=10", nil)
	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Test with reason filter
	req = httptest.NewRequest("GET", "/quarantine?reason=checksum_mismatch", nil)
	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestLibraryHandler_ResolveQuarantineItem(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()
	
	// Setup test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Create repository, media service, and quarantine service
	repo := services.NewRepository(db)
	quarantineSvc := media.NewQuarantineService(db, filepath.Join(tempDir, "quarantine"))

	// Create media service
	mediaSvc := media.NewMediaService(db, nil, nil, quarantineSvc)

	// Create library handler
	handler := &LibraryHandler{
		repo:          repo,
		mediaSvc:      mediaSvc,
		asynqClient:   nil,
		quarantineSvc: quarantineSvc,
	}

	// Create Fiber app for testing
	app := fiber.New()
	app.Post("/quarantine/:id/resolve", handler.ResolveQuarantineItem)

	// Create a test file to quarantine
	testFile := filepath.Join(tempDir, "test_resolve.mp3")
	err := os.WriteFile(testFile, []byte("test content for resolve"), 0644)
	assert.NoError(t, err)

	// Quarantine the test file
	err = quarantineSvc.QuarantineFile(testFile, media.TagParseError, "Test tag parse error", 1)
	assert.NoError(t, err)

	// Get the quarantine record to get its ID
	var quarantineRecords []media.QuarantineRecord
	err = db.Find(&quarantineRecords).Error
	assert.NoError(t, err)
	assert.NotEmpty(t, quarantineRecords)

	quarantineID := quarantineRecords[0].ID

	// Test resolving the quarantine item
	req := httptest.NewRequest("POST", fmt.Sprintf("/quarantine/%d/resolve", quarantineID), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestLibraryHandler_RequeueQuarantineItem(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()
	
	// Setup test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Create repository, media service, and quarantine service
	repo := services.NewRepository(db)
	quarantineSvc := media.NewQuarantineService(db, filepath.Join(tempDir, "quarantine"))

	// Create media service
	mediaSvc := media.NewMediaService(db, nil, nil, quarantineSvc)

	// Create library handler
	handler := &LibraryHandler{
		repo:          repo,
		mediaSvc:      mediaSvc,
		asynqClient:   nil,
		quarantineSvc: quarantineSvc,
	}

	// Create Fiber app for testing
	app := fiber.New()
	app.Post("/quarantine/:id/requeue", handler.RequeueQuarantineItem)

	// Create a test file to quarantine
	testFile := filepath.Join(tempDir, "test_requeue.mp3")
	err := os.WriteFile(testFile, []byte("test content for requeue"), 0644)
	assert.NoError(t, err)

	// Quarantine the test file
	err = quarantineSvc.QuarantineFile(testFile, media.UnsupportedContainer, "Test unsupported container", 1)
	assert.NoError(t, err)

	// Get the quarantine record to get its ID
	var quarantineRecords []media.QuarantineRecord
	err = db.Find(&quarantineRecords).Error
	assert.NoError(t, err)
	assert.NotEmpty(t, quarantineRecords)

	quarantineID := quarantineRecords[0].ID

	// Test requeuing the quarantine item (restoring it)
	req := httptest.NewRequest("POST", fmt.Sprintf("/quarantine/%d/requeue", quarantineID), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestLibraryHandler_GetQuarantineItemsErrorCases(t *testing.T) {
	// Setup test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Create repository and quarantine service
	repo := services.NewRepository(db)
	quarantineSvc := media.NewQuarantineService(db, "/tmp/test_quarantine")

	// Create media service
	mediaSvc := media.NewMediaService(db, nil, nil, quarantineSvc)

	// Create library handler
	handler := &LibraryHandler{
		repo:          repo,
		mediaSvc:      mediaSvc,
		asynqClient:   nil,
		quarantineSvc: quarantineSvc,
	}

	// Create Fiber app for testing
	app := fiber.New()
	app.Get("/quarantine", handler.GetQuarantineItems)

	// Test with invalid parameters that should be handled gracefully
	req := httptest.NewRequest("GET", "/quarantine?page=-1&limit=0", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode) // Should still return 200 but with defaults
}

func TestLibraryHandler_QuarantineItemNotFound(t *testing.T) {
	// Setup test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Create repository and quarantine service
	repo := services.NewRepository(db)
	quarantineSvc := media.NewQuarantineService(db, "/tmp/test_quarantine")

	// Create media service
	mediaSvc := media.NewMediaService(db, nil, nil, quarantineSvc)

	// Create library handler
	handler := &LibraryHandler{
		repo:          repo,
		mediaSvc:      mediaSvc,
		asynqClient:   nil,
		quarantineSvc: quarantineSvc,
	}

	// Create Fiber app for testing
	app := fiber.New()
	app.Post("/quarantine/:id/resolve", handler.ResolveQuarantineItem)

	// Test resolving a non-existent quarantine item
	req := httptest.NewRequest("POST", "/quarantine/999999/resolve", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode) // Should return 404 for not found
}