package middleware

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

// RateLimiterConfig holds the configuration for rate limiting
type RateLimiterConfig struct {
	// Max requests allowed within the time window
	Max int
	// Time window for rate limiting
	Expiration time.Duration
	// Extract the key to rate limit by
	KeyGenerator func(*fiber.Ctx) string
	// Message returned when rate limit is exceeded
	Message string
	// Rate limiter for in-memory limiting (when Redis is not available)
	InMemoryLimiter *rate.Limiter
}

// DefaultRateLimiterConfig returns a default configuration
func DefaultRateLimiterConfig() *RateLimiterConfig {
	return &RateLimiterConfig{
		Max:         30,            // 30 requests
		Expiration:  1 * time.Minute, // per minute
		KeyGenerator: defaultKeyGenerator,
		Message:     "Too many requests, please try again later",
	}
}

// defaultKeyGenerator generates a key based on the client IP
func defaultKeyGenerator(c *fiber.Ctx) string {
	// Get the real IP address, considering common headers
	ip := c.Get("X-Forwarded-For")
	if ip == "" {
		ip = c.Get("X-Real-IP")
	}
	if ip == "" {
		ip = c.IP()
	}
	return ip
}

// RateLimiter creates a new rate limiter middleware
func RateLimiter(cfg *RateLimiterConfig) fiber.Handler {
	if cfg == nil {
		cfg = DefaultRateLimiterConfig()
	}

	return limiter.New(limiter.Config{
		Max:        cfg.Max,
		Expiration: cfg.Expiration,
		KeyGenerator: cfg.KeyGenerator,
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   "Rate limit exceeded",
				"message": cfg.Message,
				"code":    fiber.StatusTooManyRequests,
			})
		},
	})
}

// RateLimiterForPublicAPI creates a rate limiter for public APIs
func RateLimiterForPublicAPI() fiber.Handler {
	config := &RateLimiterConfig{
		Max:        60,              // 60 requests per minute for public APIs
		Expiration: 1 * time.Minute,
		KeyGenerator: defaultKeyGenerator,
		Message:    "Too many requests to public API, please try again later",
	}
	return RateLimiter(config)
}

// RateLimiterForAuth creates a rate limiter for authentication endpoints
func RateLimiterForAuth() fiber.Handler {
	config := &RateLimiterConfig{
		Max:        5,               // Only 5 attempts per minute for auth endpoints
		Expiration: 1 * time.Minute,
		KeyGenerator: defaultKeyGenerator,
		Message:    "Too many authentication attempts, please try again later",
	}
	return RateLimiter(config)
}

// IPBasedRateLimiter creates a rate limiter based on IP with different limits per IP type
func IPBasedRateLimiter() fiber.Handler {
	config := &RateLimiterConfig{
		Max:        100,             // 100 requests per minute for regular IPs
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			ip := extractClientIP(c)
			// For internal IPs, allow more requests
			if isLocalIP(ip) {
				return "internal:" + ip
			}
			return "external:" + ip
		},
		Message: "Too many requests from your IP address",
	}
	
	// Custom limiter with different rates based on IP
	return func(c *fiber.Ctx) error {
		ip := extractClientIP(c)
		
		// Different rate limits based on IP type
		maxRequests := 100
		if isLocalIP(ip) {
			maxRequests = 500 // Higher limit for local IPs
		} else if isBotIP(ip) {
			maxRequests = 10 // Lower limit for bot-like IPs
		}
		
		// Use the original limiter but with dynamic max
		limiterMiddleware := limiter.New(limiter.Config{
			Max:        maxRequests,
			Expiration: 1 * time.Minute,
			KeyGenerator: func(c *fiber.Ctx) string {
				return extractClientIP(c)
			},
			LimitReached: func(c *fiber.Ctx) error {
				return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
					"error":   "Rate limit exceeded",
					"message": "Too many requests from your IP address",
					"code":    fiber.StatusTooManyRequests,
				})
			},
		})
		
		return limiterMiddleware(c)
	}
}

// extractClientIP extracts the client IP address considering common headers
func extractClientIP(c *fiber.Ctx) string {
	// Check X-Forwarded-For header first (multiple IPs possible)
	forwarded := c.Get("X-Forwarded-For")
	if forwarded != "" {
		// Take the first IP if multiple are provided
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	
	// Check X-Real-IP header
	realIP := c.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}
	
	// Fallback to the direct IP
	return c.IP()
}

// isLocalIP checks if the IP is a local network IP
func isLocalIP(ip string) bool {
	// This is a simplified check - in a real implementation you'd want to parse the IP properly
	return strings.HasPrefix(ip, "10.") || 
		   strings.HasPrefix(ip, "172.") || 
		   strings.HasPrefix(ip, "192.168.") ||
		   ip == "127.0.0.1" || 
		   ip == "::1" ||
		   strings.HasPrefix(ip, "::ffff:127.0.0.1")
}

// isBotIP is a simplified check for potential bot IPs
func isBotIP(ip string) bool {
	// In a real implementation, this would be more sophisticated
	// For now, just return false
	return false
}

// RedisRateLimiter uses Redis to store rate limit data for distributed systems
func RedisRateLimiter(redisClient *redis.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ip := extractClientIP(c)
		
		// Create a key for this IP and current minute
		key := fmt.Sprintf("rate_limit:%s:%d", ip, time.Now().Unix()/60)
		
		// Use Redis to track requests
		current, err := redisClient.Incr(c.Context(), key).Result()
		if err != nil {
			// If Redis fails, we might want to be permissive
			// For now, continue with the request
			return c.Next()
		}
		
		// Set expiration if this is the first increment
		if current == 1 {
			redisClient.Expire(c.Context(), key, 60*time.Second)
		}
		
		// Check if rate limit is exceeded (100 requests per minute per IP)
		if current > 100 {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   "Rate limit exceeded",
				"message": "Too many requests from your IP address",
				"code":    fiber.StatusTooManyRequests,
			})
		}
		
		return c.Next()
	}
}