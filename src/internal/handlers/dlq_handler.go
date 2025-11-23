package handlers

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"melodee/internal/utils"
)

// DLQHandler manages Dead Letter Queue operations
type DLQHandler struct {
	inspector *asynq.Inspector
}

// NewDLQHandler creates a new DLQ handler
func NewDLQHandler(inspector *asynq.Inspector) *DLQHandler {
	return &DLQHandler{
		inspector: inspector,
	}
}

// DLQItem represents a single DLQ item
type DLQItem struct {
	ID         string      `json:"id"`
	Queue      string      `json:"queue"`
	Type       string      `json:"type"`
	Reason     string      `json:"reason"`
	Payload    string      `json:"payload"`
	CreatedAt  string      `json:"created_at"`
	RetryCount int         `json:"retry_count"`
}

// DLQRequeueRequest is the request for requeuing DLQ items
type DLQRequeueRequest struct {
	JobIDs []string `json:"job_ids"`
}

// DLQPurgeRequest is the request for purging DLQ items
type DLQPurgeRequest struct {
	JobIDs []string `json:"job_ids"`
}

// GetDLQItems retrieves the list of items in the dead letter queue
func (h *DLQHandler) GetDLQItems(c *fiber.Ctx) error {
	// Get all queues
	queueNames, err := h.inspector.Queues()
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to get queues")
	}

	var dlqItems []DLQItem

	// For each queue, get its dead letter tasks
	for _, queueName := range queueNames {
		// Get the dead letter queue info
		dlqName := asynq.Queue(queueName).Dead()
		
		// Get dead task IDs
		taskIds, err := h.inspector.ListDead(dlqName)
		if err != nil {
			// Continue to next queue if we can't access this one
			continue
		}

		// For each dead task ID, get the task details
		for _, taskId := range taskIds {
			task, err := h.inspector.GetDeadTask(dlqName, taskId)
			if err != nil {
				// Continue to next task if we can't get this one
				continue
			}

			item := DLQItem{
				ID:         task.ID,
				Queue:      queueName,
				Type:       task.Type,
				Reason:     task.ErrorMsg,
				Payload:    string(task.Payload),
				CreatedAt:  task.NextProcessAt.String(), // Using next process time as created time for now
				RetryCount: int(task.Retried),
			}
			dlqItems = append(dlqItems, item)
		}
	}

	// Return the results with pagination metadata
	return c.JSON(fiber.Map{
		"data": dlqItems,
		"pagination": fiber.Map{
			"page": 1,
			"size": len(dlqItems),
			"total": len(dlqItems),
		},
	})
}

// RequeueDLQItems requeues items from the dead letter queue
func (h *DLQHandler) RequeueDLQItems(c *fiber.Ctx) error {
	var req DLQRequeueRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid request body")
	}

	if len(req.JobIDs) == 0 {
		return utils.SendError(c, http.StatusBadRequest, "job_ids is required")
	}

	// Count requeued and failed jobs
	requeued := 0
	failedIds := []string{}

	// For each job ID, requeue it
	for _, jobID := range req.JobIDs {
		// Find the dead task and requeue it
		// This is a simplified implementation - in a real system you'd need to know the queue name
		// For now, we'll try all queues
		queueNames, err := h.inspector.Queues()
		if err != nil {
			failedIds = append(failedIds, jobID)
			continue
		}

		found := false
		for _, queueName := range queueNames {
			dlqName := asynq.Queue(queueName).Dead()
			
			// Get the dead task
			task, err := h.inspector.GetDeadTask(dlqName, jobID)
			if err != nil {
				continue // Not in this queue, continue to check other queues
			}

			// Create a new task with the same type and payload
			newTask := asynq.NewTask(task.Type, task.Payload)
			
			// Enqueue the task back to the original queue
			if _, err := asynq.DefaultEnqueuer.Enqueue(newTask, asynq.Queue(queueName)); err != nil {
				failedIds = append(failedIds, jobID)
				continue
			}

			// Delete from dead letter queue
			if err := h.inspector.DeleteDead(dlqName, jobID); err != nil {
				failedIds = append(failedIds, jobID)
				continue
			}

			requeued++
			found = true
			break
		}

		if !found {
			failedIds = append(failedIds, jobID)
		}
	}

	// Return result
	return c.JSON(fiber.Map{
		"status": "ok",
		"requeued": requeued,
		"failed_ids": failedIds,
	})
}

// PurgeDLQItems removes items from the dead letter queue
func (h *DLQHandler) PurgeDLQItems(c *fiber.Ctx) error {
	var req DLQPurgeRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid request body")
	}

	if len(req.JobIDs) == 0 {
		return utils.SendError(c, http.StatusBadRequest, "job_ids is required")
	}

	// Count purged and failed jobs
	purged := 0
	failedIds := []string{}

	// For each job ID, delete it from the dead letter queue
	for _, jobID := range req.JobIDs {
		// Find the dead task and delete it
		// This is a simplified implementation - in a real system you'd need to know the queue name
		queueNames, err := h.inspector.Queues()
		if err != nil {
			failedIds = append(failedIds, jobID)
			continue
		}

		found := false
		for _, queueName := range queueNames {
			dlqName := asynq.Queue(queueName).Dead()
			
			// Try to delete the task from this queue's dead letter queue
			if err := h.inspector.DeleteDead(dlqName, jobID); err == nil {
				purged++
				found = true
				break
			}
		}

		if !found {
			failedIds = append(failedIds, jobID)
		}
	}

	// Return result
	return c.JSON(fiber.Map{
		"status": "ok",
		"purged": purged,
		"failed_ids": failedIds,
	})
}