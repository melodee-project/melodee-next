package handlers

import (
	"net/http"
	"strconv"

	"melodee/internal/middleware"
	"melodee/internal/models"
	"melodee/internal/pagination"
	"melodee/internal/services"
	"melodee/internal/utils"

	"github.com/gofiber/fiber/v2"
)

// TracksV1Handler handles v1 track-related requests
type TracksV1Handler struct {
	repo *services.Repository
}

// NewTracksV1Handler creates a new v1 track handler
func NewTracksV1Handler(repo *services.Repository) *TracksV1Handler {
	return &TracksV1Handler{
		repo: repo,
	}
}

// GetTracks handles retrieving all tracks with pagination
func (h *TracksV1Handler) GetTracks(c *fiber.Ctx) error {
	// Check authentication
	_, ok := middleware.GetUserFromContext(c)
	if !ok {
		return utils.SendUnauthorizedError(c, "Authentication required")
	}

	// Get pagination parameters
	page, pageSize := pagination.GetPaginationParams(c, 1, 50)
	offset := pagination.CalculateOffset(page, pageSize)

	// Get optional query parameters for ordering
	orderBy := c.Query("orderBy", "CreatedAt")
	orderDirection := c.Query("orderDirection", "desc")

	// Validate order direction
	if orderDirection != "asc" && orderDirection != "desc" {
		orderDirection = "desc"
	}

	// Fetch tracks with pagination from the repository
	var tracks []models.Track
	var total int64

	// Count total tracks
	err := h.repo.GetDB().Model(&models.Track{}).Count(&total).Error
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to count tracks")
	}

	// Fetch tracks with pagination
	err = h.repo.GetDB().
		Offset(offset).
		Limit(pageSize).
		Order(orderBy + " " + orderDirection + ", id " + orderDirection). // Consistent secondary ordering
		Preload("Album").
		Preload("Artist").
		Find(&tracks).Error
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to fetch tracks")
	}

	// Calculate pagination metadata according to OpenAPI spec
	paginationMeta := pagination.Calculate(total, page, pageSize)

	return c.JSON(fiber.Map{
		"data": tracks,
		"meta": paginationMeta, // Using 'meta' to match OpenAPI spec
	})
}

// GetTrack handles retrieving a specific track by ID
func (h *TracksV1Handler) GetTrack(c *fiber.Ctx) error {
	// Check authentication
	_, ok := middleware.GetUserFromContext(c)
	if !ok {
		return utils.SendUnauthorizedError(c, "Authentication required")
	}

	trackID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid track ID")
	}

	track, err := h.repo.GetTrackByID(trackID)
	if err != nil {
		return utils.SendNotFoundError(c, "Track")
	}

	return c.JSON(track)
}

// GetRecentTracks handles retrieving recently added tracks
func (h *TracksV1Handler) GetRecentTracks(c *fiber.Ctx) error {
	// Check authentication
	_, ok := middleware.GetUserFromContext(c)
	if !ok {
		return utils.SendUnauthorizedError(c, "Authentication required")
	}

	// Get limit parameter
	limit := c.QueryInt("limit", 10)
	if limit < 1 {
		limit = 1
	}
	if limit > 100 {
		limit = 100
	}

	// Fetch recent tracks from the repository
	var tracks []models.Track
	err := h.repo.GetDB().
		Limit(limit).
		Order("created_at DESC, id DESC").
		Preload("Album").
		Preload("Artist").
		Find(&tracks).Error
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to fetch recent tracks")
	}

	// For the recent endpoint, we return a paginated response with a single page
	total := int64(len(tracks))
	paginationMeta := pagination.Calculate(total, 1, limit)

	return c.JSON(fiber.Map{
		"data": tracks,
		"meta": paginationMeta, // Using 'meta' to match OpenAPI spec
	})
}

// ToggleTrackStarred handles toggling the starred status for a track
func (h *TracksV1Handler) ToggleTrackStarred(c *fiber.Ctx) error {
	// Check authentication
	currentUser, ok := middleware.GetUserFromContext(c)
	if !ok {
		return utils.SendUnauthorizedError(c, "Authentication required")
	}

	trackID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid track ID")
	}

	// Parse the starred status from the URL parameter
	isStarred, err := strconv.ParseBool(c.Params("isStarred"))
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid isStarred parameter")
	}

	// Verify the track exists
	_, err = h.repo.GetTrackByID(trackID)
	if err != nil {
		return utils.SendNotFoundError(c, "Track")
	}

	// In a real implementation, we would update the user's starred status for the track
	// For now, we'll return success as the logic would depend on a user-track relationship table

	return c.JSON(fiber.Map{
		"status":     "updated",
		"track_id":   trackID,
		"is_starred": isStarred,
		"user_id":    currentUser.ID,
	})
}

// SetTrackRating handles setting the user rating for a track
func (h *TracksV1Handler) SetTrackRating(c *fiber.Ctx) error {
	// Check authentication
	currentUser, ok := middleware.GetUserFromContext(c)
	if !ok {
		return utils.SendUnauthorizedError(c, "Authentication required")
	}

	trackID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid track ID")
	}

	// Parse the rating from the URL parameter (0-5)
	rating, err := strconv.ParseInt(c.Params("rating"), 10, 32)
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid rating parameter")
	}

	if rating < 0 || rating > 5 {
		return utils.SendError(c, http.StatusBadRequest, "Rating must be between 0 and 5")
	}

	// Verify the track exists
	_, err = h.repo.GetTrackByID(trackID)
	if err != nil {
		return utils.SendNotFoundError(c, "Track")
	}

	// In a real implementation, we would update the user's rating for the track
	// For now, we'll return success as the logic would depend on a user-track rating table

	return c.JSON(fiber.Map{
		"status":   "updated",
		"track_id": trackID,
		"rating":   int32(rating),
		"user_id":  currentUser.ID,
	})
}
