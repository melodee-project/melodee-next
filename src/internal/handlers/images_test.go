package handlers

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"melodee/internal/services"
	"melodee/internal/test"
)

func TestImageHandler_UploadAvatar(t *testing.T) {
	// Create a test database instance
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Initialize repository with test db
	repo := services.NewRepository(db)

	// Create image handler
	handler := NewImageHandler(repo)

	app := fiber.New()
	app.Post("/avatar", handler.UploadAvatar)

	// Create a test image file (small PNG for testing)
	imageData := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89\x00\x00\x00\nIDATx\x9cc\xf8\x0f\x00\x00\x01\x00\x01\x00\x00\x00\x00IEND\xaeB`\x82")
	
	// Create a multipart form for the file upload
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	
	part, err := writer.CreateFormFile("file", "test.png")
	assert.NoError(t, err)
	
	_, err = part.Write(imageData)
	assert.NoError(t, err)
	
	err = writer.Close()
	assert.NoError(t, err)

	req := httptest.NewRequest("POST", "/avatar", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestImageHandler_UploadAvatarInvalidFile(t *testing.T) {
	// Create a test database instance
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Initialize repository with test db
	repo := services.NewRepository(db)

	// Create image handler
	handler := NewImageHandler(repo)

	app := fiber.New()
	app.Post("/avatar", handler.UploadAvatar)

	// Create an invalid file (too large)
	largeData := make([]byte, 3*1024*1024) // 3MB file (over 2MB limit)
	
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	
	part, err := writer.CreateFormFile("file", "large.jpg")
	assert.NoError(t, err)
	
	_, err = part.Write(largeData)
	assert.NoError(t, err)
	
	err = writer.Close()
	assert.NoError(t, err)

	req := httptest.NewRequest("POST", "/avatar", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 413, resp.StatusCode) // 413 = Request Entity Too Large
}

func TestImageHandler_UploadAvatarInvalidType(t *testing.T) {
	// Create a test database instance
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Initialize repository with test db
	repo := services.NewRepository(db)

	// Create image handler
	handler := NewImageHandler(repo)

	app := fiber.New()
	app.Post("/avatar", handler.UploadAvatar)

	// Create an invalid file type (not JPEG or PNG)
	invalidData := []byte("this is not an image")
	
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	
	part, err := writer.CreateFormFile("file", "test.txt")
	assert.NoError(t, err)
	
	_, err = part.Write(invalidData)
	assert.NoError(t, err)
	
	err = writer.Close()
	assert.NoError(t, err)

	req := httptest.NewRequest("POST", "/avatar", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 415, resp.StatusCode) // 415 = Unsupported Media Type
}

func TestImageHandler_GetImage(t *testing.T) {
	// Create a test database instance
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Initialize repository with test db
	repo := services.NewRepository(db)

	// Create image handler
	handler := NewImageHandler(repo)

	app := fiber.New()
	app.Get("/:id", handler.GetImage)

	// Test with invalid UUID format
	req := httptest.NewRequest("GET", "/invalid-id", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	// Test with valid UUID format but non-existent image
	req = httptest.NewRequest("GET", "/550e8400-e29b-41d4-a716-446655440000", nil)
	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)
}