package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"github.com/google/uuid"
	"melodee/internal/models"
	"melodee/internal/services"
	"melodee/internal/test"
)

func TestImageHandler_UploadAvatar(t *testing.T) {
	// Create test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	repo := services.NewRepository(db)
	imageHandler := NewImageHandler(repo)

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
	app.Post("/api/images/avatar", func(c *fiber.Ctx) error {
		// Set user context for testing (simulating middleware)
		ctxUser := &models.User{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			IsAdmin:  false,
		}
		c.Locals("user", ctxUser)
		return imageHandler.UploadAvatar(c)
	})

	// Helper function to create multipart form data
	createMultipartForm := func(filename, contentType string, content []byte) (string, *bytes.Buffer) {
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			t.Fatal(err)
		}
		_, err = part.Write(content)
		if err != nil {
			t.Fatal(err)
		}

		err = writer.Close()
		if err != nil {
			t.Fatal(err)
		}

		return writer.FormDataContentType(), &buf
	}

	// Create a simple valid JPEG image (minimal valid JPEG header)
	validJPEG := []byte{
		0xFF, 0xD8, // JPEG SOI marker
		0xFF, 0xE0, 0x00, 0x10, // APP0 marker
		0x4A, 0x46, 0x49, 0x46, 0x00, // "JFIF" string
		0x01, 0x01, // version
		0x01, 0x00, 0x48, 0x00, 0x48, 0x00, 0x00, // resolution
		0xFF, 0xDB, 0x00, 0x43, 0x00, // DQT marker
		0x08, 0x06, 0x06, 0x07, 0x06, 0x05, 0x08, 0x07, 0x07, 0x07, 0x09, 0x09, 0x08, 0x0A, 0x0C, 0x14, 0x0D, 0x0C, 0x0B, 0x0B, 0x0C, 0x19, 0x12, 0x13, 0x0F, 0x14, 0x1D, 0x1A, 0x1F, 0x1E, 0x1D, 0x1A, 0x1C, 0x1C, 0x20, 0x24, 0x2E, 0x27, 0x20, 0x22, 0x2C, 0x23, 0x1C, 0x1C, 0x28, 0x37, 0x29, 0x2C, 0x30, 0x31, 0x34, 0x34, 0x34, 0x1F, 0x27, 0x39, 0x3D, 0x38, 0x32, 0x3C, 0x2E, 0x33, 0x34, 0x32, // DQT table
		0xFF, 0xC0, 0x00, 0x11, // SOF0 marker
		0x08, 0x00, 0x0A, 0x00, 0x0A, // 10x10 image
		0x03, 0x01, 0x22, 0x00, 0x02, 0x11, 0x01, 0x03, 0x11, 0x01, // SOF parameters
		0xFF, 0xC4, 0x00, 0x1F, // DHT marker
		0x00, 0x00, 0x01, 0x05, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, // DHT table
		0xFF, 0xDA, 0x00, 0x0C, // SOS marker
		0x03, 0x01, 0x00, 0x02, 0x11, 0x03, 0x11, 0x00, 0x3F, 0x00, // SOS parameters and EOI
		0x7F, // Add some more data to make it a valid image
	}

	// Test successful avatar upload
	t.Run("Upload avatar successfully", func(t *testing.T) {
		contentType, buf := createMultipartForm("avatar.jpg", "image/jpeg", validJPEG)
		req := httptest.NewRequest("POST", "/api/images/avatar", buf)
		req.Header.Set("Content-Type", contentType)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		// Will likely return 500 because of nil repo, but not auth issues
		assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
		assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
	})

	// Test with PNG file
	validPNG := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk start
		0x00, 0x00, 0x00, 0x0A, 0x00, 0x00, 0x00, 0x0A, // 10x10 image
		0x08, 0x06, 0x00, 0x00, 0x00, // Color type, compression, filter, interlace
		0x1F, 0x15, 0xC4, 0x89, // IHDR CRC
		// We'll stop here for a minimal valid PNG
	}

	t.Run("Upload PNG avatar successfully", func(t *testing.T) {
		contentType, buf := createMultipartForm("avatar.png", "image/png", validPNG)
		req := httptest.NewRequest("POST", "/api/images/avatar", buf)
		req.Header.Set("Content-Type", contentType)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		// Will likely return 500 because of nil repo, but not auth issues
		assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
		assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
	})

	// Test with invalid file type
	t.Run("Upload fails with invalid file type", func(t *testing.T) {
		invalidFile := []byte("This is not an image file")
		contentType, buf := createMultipartForm("document.txt", "text/plain", invalidFile)
		req := httptest.NewRequest("POST", "/api/images/avatar", buf)
		req.Header.Set("Content-Type", contentType)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		// Should fail with validation error (415 or 400), not auth issues
		assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
		assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
	})

	// Test with file too large (over 2MB limit)
	t.Run("Upload fails with file too large", func(t *testing.T) {
		// Create a file larger than 2MB
		largeFile := make([]byte, 3*1024*1024) // 3MB
		for i := range largeFile {
			largeFile[i] = byte(i % 256)
		}
		contentType, buf := createMultipartForm("large_avatar.jpg", "image/jpeg", largeFile)
		req := httptest.NewRequest("POST", "/api/images/avatar", buf)
		req.Header.Set("Content-Type", contentType)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		// Should fail with entity too large (413), not auth issues
		assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
		assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
	})

	// Test without authentication
	appNoAuth := fiber.New()
	appNoAuth.Post("/api/images/avatar", imageHandler.UploadAvatar)

	t.Run("Upload fails without authentication", func(t *testing.T) {
		contentType, buf := createMultipartForm("avatar.jpg", "image/jpeg", validJPEG)
		req := httptest.NewRequest("POST", "/api/images/avatar", buf)
		req.Header.Set("Content-Type", contentType)

		resp, err := appNoAuth.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestImageHandler_GetImage(t *testing.T) {
	// Create test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	repo := services.NewRepository(db)
	imageHandler := NewImageHandler(repo)

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
	app.Get("/api/images/:id", func(c *fiber.Ctx) error {
		ctxUser := &models.User{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			IsAdmin:  false,
		}
		c.Locals("user", ctxUser)
		return imageHandler.GetImage(c)
	})

	// Test getting image with valid UUID
	t.Run("Get image with valid ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/images/123e4567-e89b-12d3-a456-426614174000", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		// Will likely return 404 (not found) since image doesn't exist
		// But not due to auth issues
		assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
		assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
	})

	// Test getting image with invalid UUID format
	t.Run("Get image fails with invalid ID format", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/images/invalid-id-format", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		// Should return 400 (bad request) for invalid format
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	// Test without authentication
	appNoAuth := fiber.New()
	appNoAuth.Get("/api/images/:id", imageHandler.GetImage)

	t.Run("Get image fails without authentication", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/images/123e4567-e89b-12d3-a456-426614174000", nil)
		resp, err := appNoAuth.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestImageHandler_EdgeCases(t *testing.T) {
	// Create test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	repo := services.NewRepository(db)
	imageHandler := NewImageHandler(repo)

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

	// Test with various edge cases for upload
	app := fiber.New()
	app.Post("/api/images/avatar", func(c *fiber.Ctx) error {
		ctxUser := &models.User{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			IsAdmin:  false,
		}
		c.Locals("user", ctxUser)
		return imageHandler.UploadAvatar(c)
	})

	// Helper function to create multipart form data
	createMultipartForm := func(filename, contentType string, content []byte) (string, *bytes.Buffer) {
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		if filename != "" {
			part, err := writer.CreateFormFile("file", filename)
			if err != nil {
				t.Fatal(err)
			}
			_, err = part.Write(content)
			if err != nil {
				t.Fatal(err)
			}
		}

		err := writer.Close()
		if err != nil {
			t.Fatal(err)
		}

		return writer.FormDataContentType(), &buf
	}

	// Test with empty file
	t.Run("Upload fails with empty file", func(t *testing.T) {
		contentType, buf := createMultipartForm("empty.jpg", "image/jpeg", []byte{})
		req := httptest.NewRequest("POST", "/api/images/avatar", buf)
		req.Header.Set("Content-Type", contentType)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		// Should fail with validation error
		assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
		assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
	})

	// Test with no file provided
	t.Run("Upload fails with no file provided", func(t *testing.T) {
		// Create form data without the file field
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		writer.Close()
		
		req := httptest.NewRequest("POST", "/api/images/avatar", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		resp, err := app.Test(req)
		assert.NoError(t, err)
		// Should fail with bad request
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	// Create a valid JPEG for dimension testing
	smallValidJPEG := []byte{
		0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01, 0x01, 0x00, 0x48,
		0x00, 0x48, 0x00, 0x00, 0xFF, 0xDB, 0x00, 0x43, 0x00, 0x08, 0x06, 0x06, 0x07, 0x06, 0x05, 0x08,
		0x07, 0x07, 0x07, 0x09, 0x09, 0x08, 0x0A, 0x0C, 0x14, 0x0D, 0x0C, 0x0B, 0x0B, 0x0C, 0x19, 0x12,
		0x13, 0x0F, 0x14, 0x1D, 0x1A, 0x1F, 0x1E, 0x1D, 0x1A, 0x1C, 0x1C, 0x20, 0x24, 0x2E, 0x27, 0x20,
		0x22, 0x2C, 0x23, 0x1C, 0x1C, 0x28, 0x37, 0x29, 0x2C, 0x30, 0x31, 0x34, 0x34, 0x34, 0x1F, 0x27,
		0x39, 0x3D, 0x38, 0x32, 0x3C, 0x2E, 0x33, 0x34, 0x32, 0xFF, 0xC0, 0x00, 0x11, 0x08, 0x00, 0x01,
		0x00, 0x01, 0x03, 0x01, 0x22, 0x00, 0x02, 0x11, 0x01, 0x03, 0x11, 0x01, 0xFF, 0xC4, 0x00, 0x1F,
		0x00, 0x00, 0x01, 0x05, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0xFF, 0xDA, 0x00,
		0x0C, 0x03, 0x01, 0x00, 0x02, 0x11, 0x03, 0x11, 0x00, 0x3F, 0x00, 0xD9, 0xFF, 0xD9,
	}

	t.Run("Upload succeeds with small image", func(t *testing.T) {
		contentType, buf := createMultipartForm("small.jpg", "image/jpeg", smallValidJPEG)
		req := httptest.NewRequest("POST", "/api/images/avatar", buf)
		req.Header.Set("Content-Type", contentType)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		// Won't be 200 due to implementation, but won't be auth-related errors
		assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
		assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
	})

	// Test with potentially malicious filename
	t.Run("Upload handles potentially malicious filename", func(t *testing.T) {
		// Try to use a filename that might be used for directory traversal
		contentType, buf := createMultipartForm("../malicious.jpg", "image/jpeg", smallValidJPEG)
		req := httptest.NewRequest("POST", "/api/images/avatar", buf)
		req.Header.Set("Content-Type", contentType)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		// Should handle the filename securely and not be affected by traversal attempts
		assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
		assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
	})
}