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

// AlbumsV1Handler handles v1 album-related requests
type AlbumsV1Handler struct {
	repo *services.Repository
}

// NewAlbumsV1Handler creates a new v1 album handler
func NewAlbumsV1Handler(repo *services.Repository) *AlbumsV1Handler {
	return &AlbumsV1Handler{
		repo: repo,
	}
}

// GetAlbums handles retrieving all albums with pagination
func (h *AlbumsV1Handler) GetAlbums(c *fiber.Ctx) error {
	// Check authentication
	_, ok := middleware.GetUserFromContext(c)
	if !ok {
		return utils.SendUnauthorizedError(c, "Authentication required")
	}

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

	// Fetch albums with pagination from the repository
	var albums []models.Album
	var total int64

	// Count total albums
	err := h.repo.GetDB().Model(&models.Album{}).Count(&total).Error
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to count albums")
	}

	// Fetch albums with pagination
	err = h.repo.GetDB().
		Offset(offset).
		Limit(pageSize).
		Order(orderBy + " " + orderDirection + ", id " + orderDirection). // Consistent secondary ordering
		Preload("Artist").
		Find(&albums).Error
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to fetch albums")
	}

	// Calculate pagination metadata according to OpenAPI spec
	paginationMeta := pagination.Calculate(total, page, pageSize)

	return c.JSON(fiber.Map{
		"data":       albums,
		"meta":       paginationMeta, // Using 'meta' to match OpenAPI spec
	})
}

// GetAlbum handles retrieving a specific album by ID
func (h *AlbumsV1Handler) GetAlbum(c *fiber.Ctx) error {
	// Check authentication
	_, ok := middleware.GetUserFromContext(c)
	if !ok {
		return utils.SendUnauthorizedError(c, "Authentication required")
	}

	albumID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid album ID")
	}

	album, err := h.repo.GetAlbumByID(albumID)
	if err != nil {
		return utils.SendNotFoundError(c, "Album")
	}

	return c.JSON(album)
}

// GetRecentAlbums handles retrieving recently added albums
func (h *AlbumsV1Handler) GetRecentAlbums(c *fiber.Ctx) error {
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

	// Fetch recent albums from the repository
	var albums []models.Album
	err := h.repo.GetDB().
		Limit(limit).
		Order("created_at DESC, id DESC").
		Preload("Artist").
		Find(&albums).Error
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to fetch recent albums")
	}

	// For the recent endpoint, we return a paginated response with a single page
	total := int64(len(albums))
	paginationMeta := pagination.Calculate(total, 1, limit)

	return c.JSON(fiber.Map{
		"data":       albums,
		"meta":       paginationMeta, // Using 'meta' to match OpenAPI spec
	})
}

// GetAlbumSongs handles retrieving songs for a specific album
func (h *AlbumsV1Handler) GetAlbumSongs(c *fiber.Ctx) error {
	// Check authentication
	_, ok := middleware.GetUserFromContext(c)
	if !ok {
		return utils.SendUnauthorizedError(c, "Authentication required")
	}

	albumID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid album ID")
	}

	// Verify the album exists
	_, err = h.repo.GetAlbumByID(albumID)
	if err != nil {
		return utils.SendNotFoundError(c, "Album")
	}

	// Get pagination parameters
	page, pageSize := pagination.GetPaginationParams(c, 1, 50)
	offset := pagination.CalculateOffset(page, pageSize)

	// Fetch songs for the album from the repository
	var songs []models.Track
	var total int64

	// Count total songs for this album
	err = h.repo.GetDB().Model(&models.Track{}).Where("album_id = ?", albumID).Count(&total).Error
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to count album songs")
	}

	// Fetch songs with pagination
	err = h.repo.GetDB().
		Where("album_id = ?", albumID).
		Offset(offset).
		Limit(pageSize).
		Order("track_number ASC, created_at ASC, id ASC").
		Preload("Album").
		Preload("Artist").
		Find(&songs).Error
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to fetch album songs")
	}

	// Calculate pagination metadata according to OpenAPI spec
	paginationMeta := pagination.Calculate(total, page, pageSize)

	return c.JSON(fiber.Map{
		"data":       songs,
		"meta":       paginationMeta, // Using 'meta' to match OpenAPI spec
	})
}