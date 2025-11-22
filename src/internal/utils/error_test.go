package utils

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestSendErrorResponse(t *testing.T) {
	app := fiber.New()

	app.Get("/test-error", func(c *fiber.Ctx) error {
		return SendErrorResponse(c, 400, "Bad Request", "Invalid input")
	})

	req := httptest.NewRequest("GET", "/test-error", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	// Read response body
	body := make([]byte, resp.ContentLength)
	resp.Body.Read(body)
	responseBody := string(body)
	
	assert.Contains(t, responseBody, "Bad Request")
	assert.Contains(t, responseBody, "Invalid input")
}

func TestSendError(t *testing.T) {
	app := fiber.New()

	app.Get("/test-send-error", func(c *fiber.Ctx) error {
		return SendError(c, 404, "Resource not found")
	})

	req := httptest.NewRequest("GET", "/test-send-error", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)

	body := make([]byte, resp.ContentLength)
	resp.Body.Read(body)
	responseBody := string(body)
	
	assert.Contains(t, responseBody, "Resource not found")
}

func TestSendValidationError(t *testing.T) {
	app := fiber.New()

	app.Get("/test-validation-error", func(c *fiber.Ctx) error {
		return SendValidationError(c, "password", "must contain uppercase letter")
	})

	req := httptest.NewRequest("GET", "/test-validation-error", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 422, resp.StatusCode)

	body := make([]byte, resp.ContentLength)
	resp.Body.Read(body)
	responseBody := string(body)
	
	assert.Contains(t, responseBody, "Validation failed")
	assert.Contains(t, responseBody, "password: must contain uppercase letter")
}

func TestSendNotFoundError(t *testing.T) {
	app := fiber.New()

	app.Get("/test-not-found", func(c *fiber.Ctx) error {
		return SendNotFoundError(c, "User")
	})

	req := httptest.NewRequest("GET", "/test-not-found", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)

	body := make([]byte, resp.ContentLength)
	resp.Body.Read(body)
	responseBody := string(body)
	
	assert.Contains(t, responseBody, "Resource not found")
	assert.Contains(t, responseBody, "User does not exist")
}

func TestSendUnauthorizedError(t *testing.T) {
	app := fiber.New()

	app.Get("/test-unauthorized", func(c *fiber.Ctx) error {
		return SendUnauthorizedError(c, "Invalid credentials")
	})

	req := httptest.NewRequest("GET", "/test-unauthorized", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)

	body := make([]byte, resp.ContentLength)
	resp.Body.Read(body)
	responseBody := string(body)
	
	assert.Contains(t, responseBody, "Unauthorized")
	assert.Contains(t, responseBody, "Invalid credentials")
}

func TestSendForbiddenError(t *testing.T) {
	app := fiber.New()

	app.Get("/test-forbidden", func(c *fiber.Ctx) error {
		return SendForbiddenError(c, "Access denied")
	})

	req := httptest.NewRequest("GET", "/test-forbidden", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)

	body := make([]byte, resp.ContentLength)
	resp.Body.Read(body)
	responseBody := string(body)
	
	assert.Contains(t, responseBody, "Forbidden")
	assert.Contains(t, responseBody, "Access denied")
}

func TestSendInternalServerError(t *testing.T) {
	app := fiber.New()

	app.Get("/test-internal", func(c *fiber.Ctx) error {
		return SendInternalServerError(c, "Database connection failed")
	})

	req := httptest.NewRequest("GET", "/test-internal", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)

	body := make([]byte, resp.ContentLength)
	resp.Body.Read(body)
	responseBody := string(body)
	
	assert.Contains(t, responseBody, "Internal server error")
	assert.Contains(t, responseBody, "Database connection failed")
}

func TestErrorWithCode(t *testing.T) {
	err := NewErrorWithCode(nil, 400)
	
	assert.Equal(t, 400, err.GetCode())
	
	// Test with an actual error
	actualErr := NewErrorWithCode(assert.AnError, 404)
	assert.Equal(t, 404, actualErr.GetCode())
	assert.Equal(t, assert.AnError, actualErr.Unwrap())
}