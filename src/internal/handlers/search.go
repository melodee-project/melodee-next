package handlers

import (
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"melodee/internal/services"
	"melodee/internal/utils"
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

// Search performs a search across artists, albums, and songs
func (h *SearchHandler) Search(c *fiber.Ctx) error {
	// Get query parameters
	entityType := c.Query("type", "any") // artist, album, song, or any
	query := c.Query("q")               // search query
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

		return c.JSON(fiber.Map{
			"data":       artists,
			"pagination": fiber.Map{"offset": offset, "limit": limit, "total": total},
		})

	case "album", "albums":
		albums, total, err := h.repo.SearchAlbumsPaginated(query, limit, offset)
		if err != nil {
			return utils.SendInternalServerError(c, "Failed to search albums")
		}

		return c.JSON(fiber.Map{
			"data":       albums,
			"pagination": fiber.Map{"offset": offset, "limit": limit, "total": total},
		})

	case "song", "songs":
		songs, total, err := h.repo.SearchSongsPaginated(query, limit, offset)
		if err != nil {
			return utils.SendInternalServerError(c, "Failed to search songs")
		}

		return c.JSON(fiber.Map{
			"data":       songs,
			"pagination": fiber.Map{"offset": offset, "limit": limit, "total": total},
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

		songs, total, err := h.repo.SearchSongsPaginated(query, limit/3+1, offset)
		if err != nil {
			return utils.SendInternalServerError(c, "Failed to search songs")
		}

		return c.JSON(fiber.Map{
			"data": fiber.Map{
				"artists": artists,
				"albums":  albums,
				"songs":   songs,
			},
			"pagination": fiber.Map{"offset": offset, "limit": limit, "total": total},
		})

	default:
		return utils.SendError(c, http.StatusBadRequest, "Invalid search type. Use 'artist', 'album', 'song', or 'any'")
	}
}