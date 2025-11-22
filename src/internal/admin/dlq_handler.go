package admin

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"melodee/internal/middleware"
	"melodee/internal/models"
	"melodee/internal/services"
)

// DLQHandler manages Dead Letter Queue operations
type DLQHandler struct {
	db           *gorm.DB
	repo         *services.Repository
	client       *asynq.Client
	scheduler    *asynq.Scheduler
	inspector    *asynq.Inspector
	authService  *services.AuthService
}

// NewDLQHandler creates a new DLQ handler
func NewDLQHandler(
	db *gorm.DB,
	repo *services.Repository,
	client *asynq.Client,
	scheduler *asynq.Scheduler,
	inspector *asynq.Inspector,
	authService *services.AuthService,
) *DLQHandler {
	return &DLQHandler{
		db:          db,
		repo:        repo,
		client:      client,
		scheduler:   scheduler,
		inspector:   inspector,
		authService: authService,
	}
}

// DLQListResponse is the response for DLQ list endpoint
type DLQListResponse struct {
	Data []DLQItem `json:"data"`
}

// DLQItem represents a single DLQ item
type DLQItem struct {
	ID     string `json:"id"`
	Queue  string `json:"queue"`
	Type   string `json:"type"`
	Reason string `json:"reason"`
	Payload string `json:"payload"`
	RetryCount int `json:"retry_count"`
	ErrorCount int `json:"error_count"`
	ErrorMessage string `json:"error_message"`
	LastFailedAt string `json:"last_failed_at"`
}

// DLQRequeueRequest is the request for requeuing DLQ items
type DLQRequeueRequest struct {
	JobIDs []string `json:"job_ids"`
	TargetQueue string `json:"target_queue"`
}

// DLQRequeueResponse is the response for requeue endpoint
type DLQRequeueResponse struct {
	Requeued []string `json:"requeued"`
	Errors   []JobError `json:"errors"`
}

// DLQPurgeRequest is the request for purging DLQ items
type DLQPurgeRequest struct {
	JobIDs []string `json:"job_ids"`
}

// JobError represents an error for a specific job
type JobError struct {
	ID string `json:"id"`
	Error string `json:"error"`
}

// GetDLQItems retrieves the list of items in the dead letter queue
func (h *DLQHandler) GetDLQItems(c *fiber.Ctx) error {
	// Check if user is admin
	user, ok := middleware.GetUserFromContext(c)
	if !ok || !user.IsAdmin {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	// Get queue name from query parameter, default to asynq's default DLQ
	queueName := c.Query("queue", asynq.Queue("default").DeadLen()) // This will get DLQ for default queue

	// Get all dead letter tasks
	taskTypes, err := h.inspector.Queues()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get queues",
		})
	}

	var dlqItems []DLQItem

	// For each queue, get its dead tasks
	for _, qName := range taskTypes {
		deadTasks, err := h.inspector.ListDead(qName)
		if err != nil {
			continue // Skip this queue if we can't access it
		}

		for _, task := range deadTasks {
			item := DLQItem{
				ID:           task.ID,
				Queue:        qName,
				Type:         task.Type,
				Reason:       task.ErrorMsg,
				Payload:      string(task.Payload),
				RetryCount:   int(task.Retried),
				ErrorCount:   int(task.ErrorCount),
				ErrorMessage: task.ErrorMsg,
				LastFailedAt: task.LastFailedAt.String(),
			}
			dlqItems = append(dlqItems, item)
		}
	}

	response := DLQListResponse{
		Data: dlqItems,
	}

	return c.JSON(response)
}

// RequeueDLQItems requeues items from the dead letter queue to the target queue
func (h *DLQHandler) RequeueDLQItems(c *fiber.Ctx) error {
	// Check if user is admin
	user, ok := middleware.GetUserFromContext(c)
	if !ok || !user.IsAdmin {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	var req DLQRequeueRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if len(req.JobIDs) == 0 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "job_ids is required",
		})
	}

	if req.TargetQueue == "" {
		req.TargetQueue = "default" // Default to default queue if not specified
	}

	var requeued []string
	var errors []JobError

	for _, jobID := range req.JobIDs {
		// Find the dead task
		deadTask, err := h.inspector.GetDeadTask(asynq.Queue(req.TargetQueue), jobID)
		if err != nil {
			errors = append(errors, JobError{
				ID:    jobID,
				Error: fmt.Sprintf("Failed to get dead task: %v", err),
			})
			continue
		}

		// Requeue the task to the target queue
		task := asynq.NewTask(deadTask.Type, deadTask.Payload)
		_, err = h.client.Enqueue(task, asynq.Queue(req.TargetQueue))
		if err != nil {
			errors = append(errors, JobError{
				ID:    jobID,
				Error: fmt.Sprintf("Failed to requeue: %v", err),
			})
			continue
		}

		// Delete the dead task after it's been requeued
		if err := h.inspector.DeleteDead(asynq.Queue(req.TargetQueue), jobID); err != nil {
			errors = append(errors, JobError{
				ID:    jobID,
				Error: fmt.Sprintf("Failed to delete from DLQ after requeue: %v", err),
			})
			continue
		}

		requeued = append(requeued, jobID)
	}

	response := DLQRequeueResponse{
		Requeued: requeued,
		Errors:   errors,
	}

	return c.JSON(response)
}

// PurgeDLQItems removes items from the dead letter queue
func (h *DLQHandler) PurgeDLQItems(c *fiber.Ctx) error {
	// Check if user is admin
	user, ok := middleware.GetUserFromContext(c)
	if !ok || !user.IsAdmin {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	var req DLQPurgeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if len(req.JobIDs) == 0 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "job_ids is required",
		})
	}

	var errors []JobError

	for _, jobID := range req.JobIDs {
		// Try to delete from all possible DLQs if queue isn't specified
		queues, err := h.inspector.Queues()
		if err != nil {
			errors = append(errors, JobError{
				ID:    jobID,
				Error: fmt.Sprintf("Failed to get queues: %v", err),
			})
			continue
		}

		deleted := false
		for _, queue := range queues {
			if err := h.inspector.DeleteDead(asynq.Queue(queue), jobID); err == nil {
				deleted = true
				break
			}
		}

		if !deleted {
			errors = append(errors, JobError{
				ID:    jobID,
				Error: "Job not found in DLQ",
			})
		}
	}

	// For now, return success - in a real implementation we might want to return specific results
	return c.JSON(fiber.Map{
		"status": "purged",
		"count": len(req.JobIDs) - len(errors),
		"errors": errors,
	})
}

// GetDLQItem retrieves a specific item from the dead letter queue
func (h *DLQHandler) GetDLQItem(c *fiber.Ctx) error {
	// Check if user is admin
	user, ok := middleware.GetUserFromContext(c)
	if !ok || !user.IsAdmin {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	jobID := c.Params("id")
	if jobID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Job ID is required",
		})
	}

	// Look for the job in all DLQs
	queues, err := h.inspector.Queues()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get queues",
		})
	}

	for _, queue := range queues {
		deadTask, err := h.inspector.GetDeadTask(asynq.Queue(queue), jobID)
		if err == nil {
			// Found the task
			item := DLQItem{
				ID:           deadTask.ID,
				Queue:        queue,
				Type:         deadTask.Type,
				Reason:       deadTask.ErrorMsg,
				Payload:      string(deadTask.Payload),
				RetryCount:   int(deadTask.Retried),
				ErrorCount:   int(deadTask.ErrorCount),
				ErrorMessage: deadTask.ErrorMsg,
				LastFailedAt: deadTask.LastFailedAt.String(),
			}
			return c.JSON(item)
		}
	}

	// If we reach here, the job wasn't found in any DLQ
	return c.Status(http.StatusNotFound).JSON(fiber.Map{
		"error": "Job not found in DLQ",
	})
}