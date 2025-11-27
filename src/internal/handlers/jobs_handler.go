package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"melodee/internal/pagination"
	"melodee/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
)

// JobsHandler manages job queue operations
type JobsHandler struct {
	inspector *asynq.Inspector
	client    *asynq.Client
}

// NewJobsHandler creates a new jobs handler
func NewJobsHandler(inspector *asynq.Inspector, client *asynq.Client) *JobsHandler {
	return &JobsHandler{
		inspector: inspector,
		client:    client,
	}
}

// JobInfo represents information about a job
type JobInfo struct {
	ID            string                 `json:"id"`
	Queue         string                 `json:"queue"`
	Type          string                 `json:"type"`
	Payload       map[string]interface{} `json:"payload"`
	State         string                 `json:"state"`
	MaxRetry      int                    `json:"max_retry"`
	Retried       int                    `json:"retried"`
	LastErr       string                 `json:"last_error,omitempty"`
	LastFailedAt  string                 `json:"last_failed_at,omitempty"`
	Timeout       int                    `json:"timeout"`
	Deadline      string                 `json:"deadline,omitempty"`
	NextProcessAt string                 `json:"next_process_at,omitempty"`
	CompletedAt   string                 `json:"completed_at,omitempty"`
}

// QueueStats represents statistics for a queue
type QueueStats struct {
	Queue       string `json:"queue"`
	Active      int    `json:"active"`
	Pending     int    `json:"pending"`
	Scheduled   int    `json:"scheduled"`
	Retry       int    `json:"retry"`
	Archived    int    `json:"archived"`
	Completed   int    `json:"completed"`
	Aggregating int    `json:"aggregating"`
	Size        int    `json:"size"`
	Paused      bool   `json:"paused"`
}

// GetActiveJobs retrieves currently running jobs
func (h *JobsHandler) GetActiveJobs(c *fiber.Ctx) error {
	queues, err := h.inspector.Queues()
	if err != nil {
		log.Printf("ERROR: Failed to get queues: %v", err)
		return utils.SendInternalServerError(c, "Failed to retrieve queues")
	}

	var allJobs []JobInfo
	for _, queue := range queues {
		// Get active tasks in this queue
		tasks, err := h.inspector.ListActiveTasks(queue)
		if err != nil {
			log.Printf("WARN: Failed to list active tasks for queue %s: %v", queue, err)
			continue
		}

		for _, task := range tasks {
			jobInfo := h.convertTaskToJobInfo(task, "active")
			allJobs = append(allJobs, jobInfo)
		}
	}

	return c.JSON(fiber.Map{
		"data": allJobs,
		"pagination": fiber.Map{
			"total":        int64(len(allJobs)),
			"page":         1,
			"page_size":    len(allJobs),
			"total_pages":  1,
			"has_previous": false,
			"has_next":     false,
		},
	})
}

// GetPendingJobs retrieves jobs waiting in queue
func (h *JobsHandler) GetPendingJobs(c *fiber.Ctx) error {
	page, pageSize := pagination.GetPaginationParams(c, 1, 50)

	queues, err := h.inspector.Queues()
	if err != nil {
		log.Printf("ERROR: Failed to get queues: %v", err)
		return utils.SendInternalServerError(c, "Failed to retrieve queues")
	}

	var allJobs []JobInfo
	for _, queue := range queues {
		// Get pending tasks in this queue
		tasks, err := h.inspector.ListPendingTasks(queue, asynq.PageSize(100))
		if err != nil {
			log.Printf("WARN: Failed to list pending tasks for queue %s: %v", queue, err)
			continue
		}

		for _, task := range tasks {
			jobInfo := h.convertTaskToJobInfo(task, "pending")
			allJobs = append(allJobs, jobInfo)
		}
	}

	total := int64(len(allJobs))
	offset := pagination.CalculateOffset(page, pageSize)

	// Apply pagination
	var paginatedJobs []JobInfo
	if offset < len(allJobs) {
		endIndex := offset + pageSize
		if endIndex > len(allJobs) {
			endIndex = len(allJobs)
		}
		paginatedJobs = allJobs[offset:endIndex]
	}

	paginationMeta := pagination.Calculate(total, page, pageSize)

	return c.JSON(fiber.Map{
		"data":       paginatedJobs,
		"pagination": paginationMeta,
	})
}

// GetScheduledJobs retrieves jobs scheduled for future execution
func (h *JobsHandler) GetScheduledJobs(c *fiber.Ctx) error {
	page, pageSize := pagination.GetPaginationParams(c, 1, 50)

	queues, err := h.inspector.Queues()
	if err != nil {
		log.Printf("ERROR: Failed to get queues: %v", err)
		return utils.SendInternalServerError(c, "Failed to retrieve queues")
	}

	var allJobs []JobInfo
	for _, queue := range queues {
		tasks, err := h.inspector.ListScheduledTasks(queue, asynq.PageSize(100))
		if err != nil {
			log.Printf("WARN: Failed to list scheduled tasks for queue %s: %v", queue, err)
			continue
		}

		for _, task := range tasks {
			jobInfo := h.convertTaskToJobInfo(task, "scheduled")
			allJobs = append(allJobs, jobInfo)
		}
	}

	total := int64(len(allJobs))
	offset := pagination.CalculateOffset(page, pageSize)

	var paginatedJobs []JobInfo
	if offset < len(allJobs) {
		endIndex := offset + pageSize
		if endIndex > len(allJobs) {
			endIndex = len(allJobs)
		}
		paginatedJobs = allJobs[offset:endIndex]
	}

	paginationMeta := pagination.Calculate(total, page, pageSize)

	return c.JSON(fiber.Map{
		"data":       paginatedJobs,
		"pagination": paginationMeta,
	})
}

// GetQueueStats retrieves statistics for all queues
func (h *JobsHandler) GetQueueStats(c *fiber.Ctx) error {
	queues, err := h.inspector.Queues()
	if err != nil {
		log.Printf("ERROR: Failed to get queues: %v", err)
		return utils.SendInternalServerError(c, "Failed to retrieve queues")
	}

	var stats []QueueStats
	for _, queue := range queues {
		queueInfo, err := h.inspector.GetQueueInfo(queue)
		if err != nil {
			log.Printf("WARN: Failed to get queue info for %s: %v", queue, err)
			continue
		}

		stat := QueueStats{
			Queue:       queue,
			Active:      queueInfo.Active,
			Pending:     queueInfo.Pending,
			Scheduled:   queueInfo.Scheduled,
			Retry:       queueInfo.Retry,
			Archived:    queueInfo.Archived,
			Completed:   queueInfo.Completed,
			Aggregating: queueInfo.Aggregating,
			Size:        queueInfo.Size,
			Paused:      queueInfo.Paused,
		}
		stats = append(stats, stat)
	}

	return c.JSON(fiber.Map{
		"data": stats,
	})
}

// convertTaskToJobInfo converts an Asynq TaskInfo to JobInfo
func (h *JobsHandler) convertTaskToJobInfo(task *asynq.TaskInfo, state string) JobInfo {
	var payload map[string]interface{}
	if err := json.Unmarshal(task.Payload, &payload); err != nil {
		log.Printf("WARN: Failed to unmarshal task payload: %v", err)
		payload = map[string]interface{}{"raw": string(task.Payload)}
	}

	jobInfo := JobInfo{
		ID:       task.ID,
		Queue:    task.Queue,
		Type:     task.Type,
		Payload:  payload,
		State:    state,
		MaxRetry: task.MaxRetry,
		Retried:  task.Retried,
		Timeout:  int(task.Timeout.Seconds()),
	}

	if task.LastErr != "" {
		jobInfo.LastErr = task.LastErr
	}

	if !task.LastFailedAt.IsZero() {
		jobInfo.LastFailedAt = task.LastFailedAt.Format(time.RFC3339)
	}

	if !task.Deadline.IsZero() {
		jobInfo.Deadline = task.Deadline.Format(time.RFC3339)
	}

	if !task.NextProcessAt.IsZero() {
		jobInfo.NextProcessAt = task.NextProcessAt.Format(time.RFC3339)
	}

	if !task.CompletedAt.IsZero() {
		jobInfo.CompletedAt = task.CompletedAt.Format(time.RFC3339)
	}

	return jobInfo
}

// CancelJob cancels a specific job
func (h *JobsHandler) CancelJob(c *fiber.Ctx) error {
	jobID := c.Params("id")
	if jobID == "" {
		return utils.SendError(c, 400, "Job ID is required")
	}

	// Try to cancel from all queues
	queues, err := h.inspector.Queues()
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to retrieve queues")
	}

	var canceled bool
	for range queues {
		err := h.inspector.CancelProcessing(jobID)
		if err == nil {
			canceled = true
			break
		}
	}

	if !canceled {
		return utils.SendError(c, 404, "Job not found or already completed")
	}

	return c.JSON(fiber.Map{
		"status":  "ok",
		"message": fmt.Sprintf("Job %s canceled successfully", jobID),
	})
}

// RunTask manually runs/enqueues a task (for testing/admin purposes)
func (h *JobsHandler) RunTask(c *fiber.Ctx) error {
	var req struct {
		TaskType string                 `json:"task_type"`
		Queue    string                 `json:"queue"`
		Payload  map[string]interface{} `json:"payload"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, 400, "Invalid request body")
	}

	if req.TaskType == "" {
		return utils.SendError(c, 400, "task_type is required")
	}

	if req.Queue == "" {
		req.Queue = "default"
	}

	payloadBytes, err := json.Marshal(req.Payload)
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to marshal payload")
	}

	task := asynq.NewTask(req.TaskType, payloadBytes)
	info, err := h.client.EnqueueContext(context.Background(), task, asynq.Queue(req.Queue))
	if err != nil {
		return utils.SendInternalServerError(c, fmt.Sprintf("Failed to enqueue task: %v", err))
	}

	return c.JSON(fiber.Map{
		"status":  "ok",
		"message": "Task enqueued successfully",
		"job_id":  info.ID,
		"queue":   info.Queue,
	})
}
