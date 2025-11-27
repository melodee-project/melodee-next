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

// ArtistsV1Handler handles v1 artist-related requests
type ArtistsV1Handler struct {
	repo *services.Repository
}

// NewArtistsV1Handler creates a new v1 artist handler
func NewArtistsV1Handler(repo *services.Repository) *ArtistsV1Handler {
	return &ArtistsV1Handler{
		repo: repo,
	}
}

// GetArtists handles retrieving all artists with pagination
func (h *ArtistsV1Handler) GetArtists(c *fiber.Ctx) error {
	// Check authentication
	_, ok := middleware.GetUserFromContext(c)
	if !ok {
		return utils.SendUnauthorizedError(c, "Authentication required")
	}

	// Get optional query parameter for name filtering
	nameFilter := c.Query("q", "")

	// Get pagination parameters
	page, pageSize := pagination.GetPaginationParams(c, 1, 10)
	offset := pagination.CalculateOffset(page, pageSize)

	// Get optional query parameters for ordering
	orderBy := c.Query("orderBy", "CreatedAt")
	orderDirection := c.Query("orderDirection", "desc")

	// Validate order direction
	if orderDirection != "asc" && orderDirection != "desc" {
		orderDirection = "desc"
	}

	// Build query with optional name filter
	query := h.repo.GetDB().Model(&models.Artist{})
	if nameFilter != "" {
		normalizedFilter := "%" + nameFilter + "%"
		query = query.Where("name_normalized ILIKE ?", normalizedFilter)
	}

	// Count total artists
	var total int64
	err := query.Count(&total).Error
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to count artists")
	}

	// Fetch artists with pagination
	var artists []models.Artist
	err = query.
		Offset(offset).
		Limit(pageSize).
		Order(orderBy + " " + orderDirection + ", id " + orderDirection). // Consistent secondary ordering
		Find(&artists).Error
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to fetch artists")
	}

	// Calculate pagination metadata according to OpenAPI spec
	paginationMeta := pagination.Calculate(total, page, pageSize)

	return c.JSON(fiber.Map{
		"data":       artists,
		"meta":       paginationMeta, // Using 'meta' to match OpenAPI spec
	})
}

// GetArtist handles retrieving a specific artist by ID
func (h *ArtistsV1Handler) GetArtist(c *fiber.Ctx) error {
	// Check authentication
	_, ok := middleware.GetUserFromContext(c)
	if !ok {
		return utils.SendUnauthorizedError(c, "Authentication required")
	}

	artistID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid artist ID")
	}

	artist, err := h.repo.GetArtistByID(artistID)
	if err != nil {
		return utils.SendNotFoundError(c, "Artist")
	}

	return c.JSON(artist)
}

// GetRecentArtists handles retrieving recently added artists
func (h *ArtistsV1Handler) GetRecentArtists(c *fiber.Ctx) error {
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

	// Fetch recent artists from the repository
	var artists []models.Artist
	err := h.repo.GetDB().
		Limit(limit).
		Order("created_at DESC, id DESC").
		Find(&artists).Error
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to fetch recent artists")
	}

	// For the recent endpoint, we return a paginated response with a single page
	total := int64(len(artists))
	paginationMeta := pagination.Calculate(total, 1, limit)

	return c.JSON(fiber.Map{
		"data":       artists,
		"meta":       paginationMeta, // Using 'meta' to match OpenAPI spec
	})
}

// GetArtistAlbums handles retrieving albums for a specific artist
func (h *ArtistsV1Handler) GetArtistAlbums(c *fiber.Ctx) error {
	// Check authentication
	_, ok := middleware.GetUserFromContext(c)
	if !ok {
		return utils.SendUnauthorizedError(c, "Authentication required")
	}

	artistID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid artist ID")
	}

	// Get pagination parameters
	page, pageSize := pagination.GetPaginationParams(c, 1, 10)
	offset := pagination.CalculateOffset(page, pageSize)

	// Fetch albums for the artist from the repository
	var albums []models.Album
	var total int64

	// Count total albums for this artist
	err = h.repo.GetDB().Model(&models.Album{}).Where("artist_id = ?", artistID).Count(&total).Error
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to count artist albums")
	}

	// Fetch albums with pagination
	err = h.repo.GetDB().
		Where("artist_id = ?", artistID).
		Offset(offset).
		Limit(pageSize).
		Order("release_year ASC, name ASC, id ASC"). // Sort by year, then name, then id for consistency
		Preload("Artist").
		Find(&albums).Error
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to fetch artist albums")
	}

	// Calculate pagination metadata according to OpenAPI spec
	paginationMeta := pagination.Calculate(total, page, pageSize)

	return c.JSON(fiber.Map{
		"data":       albums,
		"meta":       paginationMeta, // Using 'meta' to match OpenAPI spec
	})
}

// GetArtistSongs handles retrieving songs for a specific artist
func (h *ArtistsV1Handler) GetArtistSongs(c *fiber.Ctx) error {
	// Check authentication
	_, ok := middleware.GetUserFromContext(c)
	if !ok {
		return utils.SendUnauthorizedError(c, "Authentication required")
	}

	artistID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid artist ID")
	}

	// Get optional query parameter for title filtering
	titleFilter := c.Query("q", "")

	// Get pagination parameters
	page, pageSize := pagination.GetPaginationParams(c, 1, 50)
	offset := pagination.CalculateOffset(page, pageSize)

	// Build query for songs by the artist, with optional title filter
	query := h.repo.GetDB().Model(&models.Track{}).Where("artist_id = ?", artistID)
	if titleFilter != "" {
		normalizedFilter := "%" + titleFilter + "%"
		query = query.Where("name_normalized ILIKE ?", normalizedFilter)
	}

	// Count total songs for this artist
	var total int64
	err = query.Count(&total).Error
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to count artist songs")
	}

	// Fetch songs with pagination
	var songs []models.Track
	err = query.
		Offset(offset).
		Limit(pageSize).
		Order("track_number ASC, created_at ASC, id ASC").
		Preload("Album").
		Preload("Artist").
		Find(&songs).Error
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to fetch artist songs")
	}

	// Calculate pagination metadata according to OpenAPI spec
	paginationMeta := pagination.Calculate(total, page, pageSize)

	return c.JSON(fiber.Map{
		"data":       songs,
		"meta":       paginationMeta, // Using 'meta' to match OpenAPI spec
	})
}