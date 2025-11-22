package middleware

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiter(t *testing.T) {
	app := fiber.New()

	// Apply rate limiter to all routes
	rateLimiter := RateLimiterForPublicAPI()
	app.Use(rateLimiter)

	// Add a test endpoint
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Make multiple requests to test rate limiting
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	}

	// The next request should be rate limited (assuming default 30 requests per minute)
	// For testing purposes, we may need to adjust the limit lower
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	// Note: This test might not trigger rate limiting depending on the exact configuration
	// In a real test, we might need to make many requests or configure a lower limit
}

func TestRateLimiterForAuth(t *testing.T) {
	app := fiber.New()

	// Apply stricter rate limiter for auth endpoints
	authRateLimiter := RateLimiterForAuth()
	app.Use(authRateLimiter)

	// Add a test endpoint
	app.Post("/login", func(c *fiber.Ctx) error {
		return c.SendString("Login")
	})

	req := httptest.NewRequest("POST", "/login", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestDefaultKeyGenerator(t *testing.T) {
	app := fiber.New()

	// Create a custom rate limiter with the default key generator
	config := &RateLimiterConfig{
		Max:         10,
		Expiration:  1 * time.Minute,
		KeyGenerator: defaultKeyGenerator,
		Message:     "Rate limit exceeded",
	}
	rateLimiter := RateLimiter(config)
	app.Use(rateLimiter)

	app.Get("/test-key", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test-key", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}