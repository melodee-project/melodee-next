package middleware

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"golang.org/x/time/rate"
)


// NewRateLimiter creates a new rate limiter middleware with the provided configuration
func NewRateLimiter(config RateLimiterConfig) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        config.GeneralLimit,
		Expiration: config.GeneralWindow,
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Rate limit exceeded",
				"message": "Too many requests. Please try again later.",
				"retry_after": config.GeneralWindow.Seconds(),
			})
		},
	})
}

// NewAuthRateLimiter creates a rate limiter specifically for authentication endpoints
func NewAuthRateLimiter(config RateLimiterConfig) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        config.AuthLimit,
		Expiration: config.AuthWindow,
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Rate limit exceeded",
				"message": "Too many authentication attempts. Please try again later.",
				"retry_after": config.AuthWindow.Seconds(),
			})
		},
	})
}

// NewSearchRateLimiter creates a rate limiter specifically for search endpoints
func NewSearchRateLimiter(config RateLimiterConfig) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        config.SearchLimit,
		Expiration: config.SearchWindow,
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Rate limit exceeded",
				"message": "Too many search requests. Please try again later.",
				"retry_after": config.SearchWindow.Seconds(),
			})
		},
	})
}

// NewCustomRateLimiter creates a custom rate limiter with specific parameters
func NewCustomRateLimiter(limit int, window time.Duration, message string) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        limit,
		Expiration: window,
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Rate limit exceeded",
				"message": message,
				"retry_after": window.Seconds(),
			})
		},
	})
}

// NewExpensiveEndpointRateLimiter creates a rate limiter specifically for expensive operations like search, large data exports, etc.
func NewExpensiveEndpointRateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        30, // 30 requests per 10 minutes for expensive operations
		Expiration: 10 * time.Minute,
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Rate limit exceeded",
				"message": "Too many requests to expensive operations. Please try again later.",
				"retry_after": 600, // 10 minutes
			})
		},
	})
}

// RateLimitByUser creates a rate limiter that applies limits per user rather than per IP
// This requires that the user is authenticated and the user ID is available
func RateLimitByUser(queriesPerWindow int, window time.Duration) fiber.Handler {
	limiterStore := make(map[string]*rate.Limiter)
	windowStart := time.Now().Truncate(window)

	return func(c *fiber.Ctx) error {
		// Get user ID from context if available
		user, ok := GetUserFromContext(c)
		var userID string
		
		if ok && user.ID != 0 {
			userID = strconv.FormatInt(user.ID, 10)
		} else {
			// Fallback to IP address if user not authenticated
			userID = c.IP()
		}

		// Create a key based on current window to reset counters periodically
		key := userID + ":" + windowStart.Format("2006-01-02-15:04")

		limiter, exists := limiterStore[key]
		if !exists {
			limiter = rate.NewLimiter(rate.Every(window/time.Duration(queriesPerWindow)), queriesPerWindow)
			limiterStore[key] = limiter
		}

		if !limiter.Allow() {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Rate limit exceeded",
				"message": "Too many requests. Please try again later.",
				"retry_after": window.Seconds(),
			})
		}

		return c.Next()
	}
}