package handlers

import (
	"net/http"
	"strconv"
	"time"

	"melodee/internal/logging"
	"melodee/internal/pagination"
	"melodee/internal/utils"

	"github.com/gofiber/fiber/v2"
)

// LogsHandler handles log-related operations
type LogsHandler struct {
	storage *logging.LogStorage
}

// NewLogsHandler creates a new logs handler
func NewLogsHandler(storage *logging.LogStorage) *LogsHandler {
	return &LogsHandler{
		storage: storage,
	}
}

// GetLogs retrieves logs with filtering and pagination
func (h *LogsHandler) GetLogs(c *fiber.Ctx) error {
	page, pageSize := pagination.GetPaginationParams(c, 1, 100)
	offset := pagination.CalculateOffset(page, pageSize)

	filters := logging.LogFilters{
		Level:     c.Query("level"),
		Module:    c.Query("module"),
		JobType:   c.Query("job_type"),
		RequestID: c.Query("request_id"),
		Search:    c.Query("search"),
		Offset:    offset,
		Limit:     pageSize,
	}

	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if userID, err := strconv.ParseInt(userIDStr, 10, 64); err == nil {
			filters.UserID = userID
		}
	}

	if libIDStr := c.Query("library_id"); libIDStr != "" {
		if libID, err := strconv.ParseInt(libIDStr, 10, 32); err == nil {
			filters.LibraryID = int32(libID)
		}
	}

	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			filters.StartTime = startTime
		}
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			filters.EndTime = endTime
		}
	}

	logs, total, err := h.storage.Query(c.Context(), filters)
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to query logs: "+err.Error())
	}

	paginationMeta := pagination.Calculate(total, page, pageSize)

	return c.JSON(fiber.Map{
		"data":       logs,
		"pagination": paginationMeta,
	})
}

// GetLogStats retrieves log statistics
func (h *LogsHandler) GetLogStats(c *fiber.Ctx) error {
	since := time.Now().Add(-24 * time.Hour)

	if sinceStr := c.Query("since"); sinceStr != "" {
		if parsedSince, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since = parsedSince
		}
	}

	stats, err := h.storage.GetLogStats(c.Context(), since)
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to retrieve log stats: "+err.Error())
	}

	return c.JSON(stats)
}

// CleanupOldLogs removes old log entries
func (h *LogsHandler) CleanupOldLogs(c *fiber.Ctx) error {
	daysStr := c.Query("older_than_days", "30")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days <= 0 {
		return utils.SendError(c, http.StatusBadRequest, "Invalid older_than_days parameter")
	}

	olderThan := time.Duration(days) * 24 * time.Hour
	deleted, err := h.storage.DeleteOldLogs(c.Context(), olderThan)
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to cleanup logs: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"status":  "ok",
		"deleted": deleted,
		"message": "Old logs cleaned up successfully",
	})
}

// DownloadLogs exports logs as JSON
func (h *LogsHandler) DownloadLogs(c *fiber.Ctx) error {
	filters := logging.LogFilters{
		Level:   c.Query("level"),
		Module:  c.Query("module"),
		JobType: c.Query("job_type"),
		Search:  c.Query("search"),
		Limit:   10000,
	}

	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			filters.StartTime = startTime
		}
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			filters.EndTime = endTime
		}
	}

	logs, _, err := h.storage.Query(c.Context(), filters)
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to query logs: "+err.Error())
	}

	c.Set("Content-Type", "application/json")
	c.Set("Content-Disposition", "attachment; filename=melodee-logs-"+time.Now().Format("2006-01-02-150405")+".json")

	return c.JSON(logs)
}
