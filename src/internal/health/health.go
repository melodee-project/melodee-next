package health

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"melodee/internal/config"
)

// HealthResponse represents the health check response structure
type HealthResponse struct {
	Status string           `json:"status"`
	DB     DependencyStatus `json:"db"`
	Redis  DependencyStatus `json:"redis"`
}

// DependencyStatus represents the status of a dependency
type DependencyStatus struct {
	Status    string `json:"status"`
	LatencyMs int64  `json:"latency_ms"`
}

// RegisterHealthRoutes registers the health check routes
func RegisterHealthRoutes(app *fiber.App, cfg *config.AppConfig) {
	app.Get("/healthz", func(c *fiber.Ctx) error {
		dbStatus := checkDB(cfg)
		redisStatus := checkRedis(cfg)

		status := "ok"
		if dbStatus.Status == "down" || redisStatus.Status == "down" {
			status = "down"
		} else if dbStatus.Status == "degraded" || redisStatus.Status == "degraded" {
			status = "degraded"
		}

		// Set appropriate HTTP status code
		if status == "ok" {
			c.Status(200)
		} else {
			c.Status(503)
		}

		// Set cache control header
		c.Set("Cache-Control", "no-store")
		c.Set("Content-Type", "application/json")

		return c.JSON(HealthResponse{
			Status: status,
			DB:     dbStatus,
			Redis:  redisStatus,
		})
	})
}

// checkDB checks database health
func checkDB(cfg *config.AppConfig) DependencyStatus {
	start := time.Now()

	// Create database connection string
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.Port,
		cfg.Database.SSLMode)

	// Attempt to connect to the database
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return DependencyStatus{
			Status:    "down",
			LatencyMs: time.Since(start).Milliseconds(),
		}
	}

	// Ping the database
	sqlDB, err := db.DB()
	if err != nil {
		return DependencyStatus{
			Status:    "down",
			LatencyMs: time.Since(start).Milliseconds(),
		}
	}

	err = sqlDB.Ping()
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return DependencyStatus{
			Status:    "down",
			LatencyMs: latency,
		}
	}

	// Check latency for degradation (200ms threshold as per spec)
	if latency > 200 {
		return DependencyStatus{
			Status:    "degraded",
			LatencyMs: latency,
		}
	}

	return DependencyStatus{
		Status:    "ok",
		LatencyMs: latency,
	}
}

// checkRedis checks Redis health
func checkRedis(cfg *config.AppConfig) DependencyStatus {
	start := time.Now()

	// Create Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Ping Redis
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := rdb.Ping(ctx).Err()
	latency := time.Since(start).Milliseconds()

	// Close the client
	rdb.Close()

	if err != nil {
		return DependencyStatus{
			Status:    "down",
			LatencyMs: latency,
		}
	}

	// Check latency for degradation (100ms threshold as per spec)
	if latency > 100 {
		return DependencyStatus{
			Status:    "degraded",
			LatencyMs: latency,
		}
	}

	return DependencyStatus{
		Status:    "ok",
		LatencyMs: latency,
	}
}