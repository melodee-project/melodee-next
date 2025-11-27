package handlers

import (
	"net/http"
	"strconv"

	"melodee/internal/pagination"
	"melodee/internal/services"
	"melodee/internal/utils"

	"github.com/gofiber/fiber/v2"
)

// SearchHandler handles search-related requests
type SearchHandler struct {
	repo *services.Repository
}

// NewSearchHandler creates a new search handler
func NewSearchHandler(repo *services.Repository) *SearchHandler {
	return &SearchHandler{
		repo: repo,
	}
}

// Search performs a search across artists, albums, and tracks
func (h *SearchHandler) Search(c *fiber.Ctx) error {
	// Get query parameters
	entityType := c.Query("type", "any") // artist, album, track, or any
	query := c.Query("q")                // search query
	offset, err := strconv.Atoi(c.Query("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}
	limit, err := strconv.Atoi(c.Query("limit", "50"))
	if err != nil || limit < 1 {
		limit = 50
	}
	if limit > 500 { // Max limit as per spec
		limit = 500
	}

	if query == "" {
		return utils.SendError(c, http.StatusBadRequest, "Search query is required")
	}

	// Perform the search based on entity type
	switch entityType {
	case "artist", "artists":
		artists, total, err := h.repo.SearchArtistsPaginated(query, limit, offset)
		if err != nil {
			return utils.SendInternalServerError(c, "Failed to search artists")
		}

		// Calculate pagination metadata according to OpenAPI spec
		paginationMeta := pagination.CalculateWithOffset(total, offset, limit)

		return c.JSON(fiber.Map{
			"data":       artists,
			"pagination": paginationMeta,
		})

	case "album", "albums":
		albums, total, err := h.repo.SearchAlbumsPaginated(query, limit, offset)
		if err != nil {
			return utils.SendInternalServerError(c, "Failed to search albums")
		}

		// Calculate pagination metadata according to OpenAPI spec
		paginationMeta := pagination.CalculateWithOffset(total, offset, limit)

		return c.JSON(fiber.Map{
			"data":       albums,
			"pagination": paginationMeta,
		})

	case "song", "songs", "track", "tracks":
		tracks, total, err := h.repo.SearchTracksPaginated(query, limit, offset)
		if err != nil {
			return utils.SendInternalServerError(c, "Failed to search tracks")
		}

		// Calculate pagination metadata according to OpenAPI spec
		paginationMeta := pagination.CalculateWithOffset(total, offset, limit)

		return c.JSON(fiber.Map{
			"data":       tracks,
			"pagination": paginationMeta,
		})

	case "any", "all", "":
		// Search across all entities - return a combined result
		// In a real implementation, we might want to create a more sophisticated combined search
		// For now, we'll just search each type separately and return them organized
		artists, _, err := h.repo.SearchArtistsPaginated(query, limit/3+1, offset)
		if err != nil {
			return utils.SendInternalServerError(c, "Failed to search artists")
		}

		albums, _, err := h.repo.SearchAlbumsPaginated(query, limit/3+1, offset)
		if err != nil {
			return utils.SendInternalServerError(c, "Failed to search albums")
		}

		tracks, total, err := h.repo.SearchTracksPaginated(query, limit/3+1, offset)
		if err != nil {
			return utils.SendInternalServerError(c, "Failed to search tracks")
		}

		// Calculate pagination metadata according to OpenAPI spec
		paginationMeta := pagination.CalculateWithOffset(total, offset, limit)

		return c.JSON(fiber.Map{
			"data": fiber.Map{
				"artists": artists,
				"albums":  albums,
				"songs":   tracks, // keep key for backward compatibility
			},
			"pagination": paginationMeta,
		})

	default:
		return utils.SendError(c, http.StatusBadRequest, "Invalid search type. Use 'artist', 'album', 'track', or 'any'")
	}
}
