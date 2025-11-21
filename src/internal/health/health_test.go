package health

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"

	"melodee/internal/config"
)

func TestHealthResponseStruct(t *testing.T) {
	// Test the response structure without actually connecting to DB/Redis
	status := DependencyStatus{
		Status:    "ok",
		LatencyMs: 50,
	}

	assert.Equal(t, "ok", status.Status)
	assert.Equal(t, int64(50), status.LatencyMs)

	healthResp := HealthResponse{
		Status: "ok",
		DB:     status,
		Redis:  status,
	}

	assert.Equal(t, "ok", healthResp.Status)
	assert.Equal(t, status, healthResp.DB)
	assert.Equal(t, status, healthResp.Redis)
}

// Create a simple test for the response without actual health checks
func TestHealthRouteRegistration(t *testing.T) {
	// Create a mock config
	cfg := &config.AppConfig{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Database: config.DatabaseConfig{
			Host:   "localhost",
			Port:   5432,
			User:   "testuser",
			DBName: "testdb",
			SSLMode: "disable",
		},
		Redis: config.RedisConfig{
			Addr: "localhost:6379",
		},
	}

	// Create a new Fiber app for testing
	app := fiber.New()

	// Register health routes
	RegisterHealthRoutes(app, cfg)

	// Test that the route is registered by checking it doesn't return 404
	req := httptest.NewRequest("GET", "/healthz", nil)
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	// The health check will likely return 503 in test environment
	// because we don't have real DB/Redis connections
	assert.Contains(t, []int{200, 503}, resp.StatusCode)
}

func TestHealthResponseJSON(t *testing.T) {
	// Test that the response structure is valid JSON
	resp := HealthResponse{
		Status: "ok",
		DB: DependencyStatus{
			Status:    "ok",
			LatencyMs: 10,
		},
		Redis: DependencyStatus{
			Status:    "ok",
			LatencyMs: 5,
		},
	}

	jsonBytes, err := json.Marshal(resp)
	assert.NoError(t, err)
	assert.Contains(t, string(jsonBytes), "status")
	assert.Contains(t, string(jsonBytes), "db")
	assert.Contains(t, string(jsonBytes), "redis")
}