package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"melodee/internal/capacity"
	"melodee/internal/config"
)

// HealthMetricsHandler handles health check and metrics endpoints
type HealthMetricsHandler struct {
	db        *gorm.DB
	config    *config.AppConfig
	capacityProbe *capacity.CapacityProbe
	asynqInspector *asynq.Inspector
}

// NewHealthMetricsHandler creates a new health and metrics handler
func NewHealthMetricsHandler(
	db *gorm.DB,
	config *config.AppConfig,
	capacityProbe *capacity.CapacityProbe,
	asynqInspector *asynq.Inspector,
) *HealthMetricsHandler {
	return &HealthMetricsHandler{
		db:              db,
		config:          config,
		capacityProbe:   capacityProbe,
		asynqInspector:  asynqInspector,
	}
}

// HealthCheck handles the health check endpoint
func (h *HealthMetricsHandler) HealthCheck(c *fiber.Ctx) error {
	// Check database health
	dbStartTime := time.Now()
	dbErr := h.db.Exec("SELECT 1").Error
	dbLatency := time.Since(dbStartTime).Milliseconds()

	dbStatus := "ok"
	if dbErr != nil {
		dbStatus = "down"
	} else if dbLatency > 200 {
		dbStatus = "degraded"
	}

	// Check Redis/Asynq health
	redisStartTime := time.Now()
	_, redisErr := h.asynqInspector.Queues()
	redisLatency := time.Since(redisStartTime).Milliseconds()

	redisStatus := "ok"
	if redisErr != nil {
		redisStatus = "down"
	} else if redisLatency > 100 {
		redisStatus = "degraded"
	}

	// Overall status
	status := "ok"
	if dbStatus == "down" || redisStatus == "down" {
		status = "down"
	} else if dbStatus == "degraded" || redisStatus == "degraded" {
		status = "degraded"
	}

	// Set appropriate HTTP status code
	httpStatusCode := http.StatusOK
	if status != "ok" {
		httpStatusCode = http.StatusServiceUnavailable
	}

	// Set cache control header
	c.Set("Cache-Control", "no-store")
	c.Set("Content-Type", "application/json")

	return c.Status(httpStatusCode).JSON(fiber.Map{
		"status": status,
		"db": fiber.Map{
			"status":      dbStatus,
			"latency_ms":  dbLatency,
		},
		"redis": fiber.Map{
			"status":      redisStatus,
			"latency_ms":  redisLatency,
		},
	})
}

// MetricsEndpoint provides prometheus-style metrics
func (h *HealthMetricsHandler) MetricsEndpoint(c *fiber.Ctx) error {
	// Generate Prometheus metrics format
	metrics := "# HELP melodee_health_status Health status of service dependencies\n"
	metrics += "# TYPE melodee_health_status gauge\n"
	
	// Get DB status
	dbErr := h.db.Exec("SELECT 1").Error
	dbStatus := 1
	if dbErr != nil {
		dbStatus = 0
	} else if dbStatus == 0 && dbErr == nil {
		// If no error but not marked as down, it's OK
		dbStatus = 1
	}
	metrics += fmt.Sprintf("melodee_health_status{dependency=\"db\"} %d\n", dbStatus)

	// Get Redis status
	_, redisErr := h.asynqInspector.Queues()
	redisStatus := 1
	if redisErr != nil {
		redisStatus = 0
	}
	metrics += fmt.Sprintf("melodee_health_status{dependency=\"redis\"} %d\n", redisStatus)

	// Add capacity metrics
	capacityStatuses, err := h.capacityProbe.GetAllCapacityStatuses()
	if err != nil {
		// Log the error but continue with other metrics
		h.logError("Failed to get capacity statuses for metrics: %v", err)
	} else {
		metrics += "# HELP melodee_capacity_percent Percentage of capacity used by library\n"
		metrics += "# TYPE melodee_capacity_percent gauge\n"
		for _, status := range capacityStatuses {
			metrics += fmt.Sprintf("melodee_capacity_percent{path=\"%s\"} %f\n", status.Path, status.UsedPercent)
		}
	}

	// Add job queue metrics
	queues, err := h.asynqInspector.Queues()
	if err != nil {
		h.logError("Failed to get queues for metrics: %v", err)
	} else {
		metrics += "# HELP melodee_queue_size Number of tasks in queue\n"
		metrics += "# TYPE melodee_queue_size gauge\n"
		
		for _, queueName := range queues {
			queueInfo, err := h.asynqInspector.GetQueueInfo(queueName)
			if err != nil {
				h.logError("Failed to get queue info for %s: %v", queueName, err)
				continue
			}

			metrics += fmt.Sprintf("melodee_queue_size{queue=\"%s\"} %d\n", queueName, queueInfo.Active)
		}

		// Add dead letter queue metrics
		metrics += "# HELP melodee_dlq_size Number of tasks in dead letter queue\n"
		metrics += "# TYPE melodee_dlq_size gauge\n"

		for _, queueName := range queues {
			// Get the DLQ name by convention (asynq uses queue_name + ":dlq")
			dlqName := queueName
			if !strings.HasSuffix(dlqName, ":dlq") {
				dlqName = queueName + ":dlq"
			}

			dlqQueueInfo, err := h.asynqInspector.GetQueueInfo(dlqName)
			if err != nil {
				// DLQ might not exist, which is fine
				continue
			}

			metrics += fmt.Sprintf("melodee_dlq_size{queue=\"%s\"} %d\n", queueName, dlqQueueInfo.Pending)
		}
	}

	c.Set("Content-Type", "text/plain; version=0.0.4")
	return c.SendString(metrics)
}

// CapacityStatus provides capacity monitoring information
func (h *HealthMetricsHandler) CapacityStatus(c *fiber.Ctx) error {
	statuses, err := h.capacityProbe.GetAllCapacityStatuses()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get capacity status",
			"details": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status": "ok",
		"data": statuses,
	})
}

// CapacityStatusForLibrary returns capacity status for a specific library
func (h *HealthMetricsHandler) CapacityStatusForLibrary(c *fiber.Ctx) error {
	libraryID := c.Params("id")
	
	// Convert to integer
	id, err := strconv.Atoi(libraryID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid library ID",
		})
	}

	status, err := h.capacityProbe.GetCapacityStatusForLibrary(int32(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{
				"error": "Library not found",
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get capacity status",
			"details": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status": "ok",
		"data": status,
	})
}

// ProbeCapacityNow triggers an immediate capacity probe
func (h *HealthMetricsHandler) ProbeCapacityNow(c *fiber.Ctx) error {
	err := h.capacityProbe.ProbeNow()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to trigger capacity probe",
			"details": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status": "probe_triggered",
		"message": "Capacity probe initiated",
	})
}

// logError logs an error message
func (h *HealthMetricsHandler) logError(format string, args ...interface{}) {
	// In a real implementation, use proper logging
	fmt.Printf("ERROR [HealthMetricsHandler]: "+format+"\n", args...)
}