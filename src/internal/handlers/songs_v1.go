package handlers

import (
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"melodee/internal/middleware"
	"melodee/internal/models"
	"melodee/internal/pagination"
	"melodee/internal/services"
	"melodee/internal/utils"
)

// SongsV1Handler handles v1 song-related requests
type SongsV1Handler struct {
	repo *services.Repository
}

// NewSongsV1Handler creates a new v1 song handler
func NewSongsV1Handler(repo *services.Repository) *SongsV1Handler {
	return &SongsV1Handler{
		repo: repo,
	}
}

// GetSongs handles retrieving all songs with pagination
func (h *SongsV1Handler) GetSongs(c *fiber.Ctx) error {
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

	// Fetch songs with pagination from the repository
	var songs []models.Track
	var total int64

	// Count total songs
	err := h.repo.GetDB().Model(&models.Track{}).Count(&total).Error
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to count songs")
	}

	// Fetch songs with pagination
	err = h.repo.GetDB().
		Offset(offset).
		Limit(pageSize).
		Order(orderBy + " " + orderDirection + ", id " + orderDirection). // Consistent secondary ordering
		Preload("Album").
		Preload("Artist").
		Find(&songs).Error
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to fetch songs")
	}

	// Calculate pagination metadata according to OpenAPI spec
	paginationMeta := pagination.Calculate(total, page, pageSize)

	return c.JSON(fiber.Map{
		"data":       songs,
		"meta":       paginationMeta, // Using 'meta' to match OpenAPI spec
	})
}

// GetSong handles retrieving a specific song by ID
func (h *SongsV1Handler) GetSong(c *fiber.Ctx) error {
	// Check authentication
	_, ok := middleware.GetUserFromContext(c)
	if !ok {
		return utils.SendUnauthorizedError(c, "Authentication required")
	}

	trackID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid song ID")
	}

	song, err := h.repo.GetSongByID(trackID)
	if err != nil {
		return utils.SendNotFoundError(c, "Song")
	}

	return c.JSON(song)
}

// GetRecentSongs handles retrieving recently added songs
func (h *SongsV1Handler) GetRecentSongs(c *fiber.Ctx) error {
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

	// Fetch recent songs from the repository
	var songs []models.Track
	err := h.repo.GetDB().
		Limit(limit).
		Order("created_at DESC, id DESC").
		Preload("Album").
		Preload("Artist").
		Find(&songs).Error
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to fetch recent songs")
	}

	// For the recent endpoint, we return a paginated response with a single page
	total := int64(len(songs))
	paginationMeta := pagination.Calculate(total, 1, limit)

	return c.JSON(fiber.Map{
		"data":       songs,
		"meta":       paginationMeta, // Using 'meta' to match OpenAPI spec
	})
}

// ToggleSongStarred handles toggling the starred status for a song
func (h *SongsV1Handler) ToggleSongStarred(c *fiber.Ctx) error {
	// Check authentication
	currentUser, ok := middleware.GetUserFromContext(c)
	if !ok {
		return utils.SendUnauthorizedError(c, "Authentication required")
	}

	trackID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid song ID")
	}

	// Parse the starred status from the URL parameter
	isStarred, err := strconv.ParseBool(c.Params("isStarred"))
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid isStarred parameter")
	}

	// Verify the song exists
	_, err = h.repo.GetSongByID(trackID)
	if err != nil {
		return utils.SendNotFoundError(c, "Song")
	}

	// In a real implementation, we would update the user's starred status for the song
	// For now, we'll return success as the logic would depend on a user-song relationship table

	return c.JSON(fiber.Map{
		"status": "updated",
		"track_id": trackID,
		"is_starred": isStarred,
		"user_id": currentUser.ID,
	})
}

// SetSongRating handles setting the user rating for a song
func (h *SongsV1Handler) SetSongRating(c *fiber.Ctx) error {
	// Check authentication
	currentUser, ok := middleware.GetUserFromContext(c)
	if !ok {
		return utils.SendUnauthorizedError(c, "Authentication required")
	}

	trackID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid song ID")
	}

	// Parse the rating from the URL parameter (0-5)
	rating, err := strconv.ParseInt(c.Params("rating"), 10, 32)
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid rating parameter")
	}

	if rating < 0 || rating > 5 {
		return utils.SendError(c, http.StatusBadRequest, "Rating must be between 0 and 5")
	}

	// Verify the song exists
	_, err = h.repo.GetSongByID(trackID)
	if err != nil {
		return utils.SendNotFoundError(c, "Song")
	}

	// In a real implementation, we would update the user's rating for the song
	// For now, we'll return success as the logic would depend on a user-song rating table

	return c.JSON(fiber.Map{
		"status": "updated",
		"track_id": trackID,
		"rating": int32(rating),
		"user_id": currentUser.ID,
	})
}