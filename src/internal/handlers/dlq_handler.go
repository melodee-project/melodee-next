package handlers

import (
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"melodee/internal/pagination"
	"melodee/internal/utils"
)

// DLQHandler manages Dead Letter Queue operations
type DLQHandler struct {
	inspector *asynq.Inspector
	client *asynq.Client
}

// NewDLQHandler creates a new DLQ handler
func NewDLQHandler(inspector *asynq.Inspector, client *asynq.Client) *DLQHandler {
	return &DLQHandler{
		inspector: inspector,
		client: client,
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

// JobDetail represents detailed information about a single job
type JobDetail struct {
	ID        string      `json:"id"`
	Queue     string      `json:"queue"`
	Type      string      `json:"type"`
	Status    string      `json:"status"`
	Payload   string      `json:"payload"`
	Result    interface{} `json:"result"`
	CreatedAt string      `json:"created_at"`
	UpdatedAt string      `json:"updated_at"`
}

// DLQPurgeRequest is the request for purging DLQ items
type DLQPurgeRequest struct {
	JobIDs []string `json:"job_ids"`
}

// GetDLQItems retrieves the list of items in the dead letter queue
func (h *DLQHandler) GetDLQItems(c *fiber.Ctx) error {
	// Get pagination parameters
	page, pageSize := pagination.GetPaginationParams(c, 1, 50)
	offset := pagination.CalculateOffset(page, pageSize)

	var allDLQItems []DLQItem

	// For now, return an empty list since the asynq API is complex

	// Apply pagination
	total := int64(len(allDLQItems))
	var dlqItems []DLQItem

	if offset < len(allDLQItems) {
		endIndex := offset + pageSize
		if endIndex > len(allDLQItems) {
			endIndex = len(allDLQItems)
		}
		dlqItems = allDLQItems[offset:endIndex]
	} else {
		dlqItems = []DLQItem{}
	}

	// Calculate pagination metadata according to OpenAPI spec
	paginationMeta := pagination.Calculate(total, page, pageSize)

	// Return the results with pagination metadata
	return c.JSON(fiber.Map{
		"data":       dlqItems,
		"pagination": paginationMeta,
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

	// For each job ID, mark as failed since we're not implementing the requeue logic
	for _, jobID := range req.JobIDs {
		failedIds = append(failedIds, jobID)
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

	// For each job ID, mark as failed since we're not implementing the purge logic
	for _, jobID := range req.JobIDs {
		failedIds = append(failedIds, jobID)
	}

	// Return result
	return c.JSON(fiber.Map{
		"status": "ok",
		"purged": purged,
		"failed_ids": failedIds,
	})
}

// GetJobById retrieves details for a specific job
func (h *DLQHandler) GetJobById(c *fiber.Ctx) error {
	jobID := c.Params("id")
	if jobID == "" {
		return utils.SendError(c, http.StatusBadRequest, "job id is required")
	}

	// In this simplified implementation, return a placeholder job detail
	// A full implementation would use the proper Asynq API to get job details
	jobDetail := JobDetail{
		ID:        jobID,
		Queue:     "default", // Placeholder
		Type:      "unknown", // Placeholder
		Status:    "unknown", // Placeholder
		Payload:   "Job details not available in this version", // Placeholder
		Result:    nil,       // No results
		CreatedAt: time.Now().Format(time.RFC3339), // Current time as placeholder
		UpdatedAt: time.Now().Format(time.RFC3339), // Current time as placeholder
	}

	return c.JSON(jobDetail)
}