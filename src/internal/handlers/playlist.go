package handlers

import (
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"melodee/internal/middleware"
	"melodee/internal/models"
	"melodee/internal/pagination"
	"melodee/internal/services"
	"melodee/internal/utils"
)

// PlaylistHandler handles playlist-related requests
type PlaylistHandler struct {
	repo *services.Repository
}

// NewPlaylistHandler creates a new playlist handler
func NewPlaylistHandler(repo *services.Repository) *PlaylistHandler {
	return &PlaylistHandler{
		repo: repo,
	}
}

// GetPlaylists handles retrieving playlists
func (h *PlaylistHandler) GetPlaylists(c *fiber.Ctx) error {
	// Check authentication
	currentUser, ok := middleware.GetUserFromContext(c)
	if !ok {
		return utils.SendUnauthorizedError(c, "Authentication required")
	}

	// Get pagination parameters
	page, pageSize := pagination.GetPaginationParams(c, 1, 10)
	offset := pagination.CalculateOffset(page, pageSize)

	playlists, total, err := h.repo.GetPlaylistsWithUser(pageSize, offset)
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to fetch playlists")
	}

	// Filter to user's playlists or public playlists
	filteredPlaylists := []models.Playlist{}
	for _, playlist := range playlists {
		if playlist.UserID == currentUser.ID || playlist.Public {
			// Only show full details to admin or owner
			if currentUser.IsAdmin || playlist.UserID == currentUser.ID {
				filteredPlaylists = append(filteredPlaylists, playlist)
			} else {
				// For public playlists, we might want to limit what we show
				filteredPlaylists = append(filteredPlaylists, playlist)
			}
		}
	}

	// Calculate pagination metadata according to OpenAPI spec
	paginationMeta := pagination.Calculate(total, page, pageSize)

	return c.JSON(fiber.Map{
		"data":       filteredPlaylists,
		"pagination": paginationMeta,
	})
}

// GetPlaylist handles retrieving a specific playlist
func (h *PlaylistHandler) GetPlaylist(c *fiber.Ctx) error {
	// Check authentication
	_, ok := middleware.GetUserFromContext(c)
	if !ok {
		return utils.SendUnauthorizedError(c, "Authentication required")
	}

	playlistID, err := c.ParamsInt("id")
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid playlist ID")
	}

	playlist, err := h.repo.GetPlaylistByID(int32(playlistID))
	if err != nil {
		return utils.SendNotFoundError(c, "Playlist")
	}

	// Check if user has permission to access this playlist
	currentUser, _ := middleware.GetUserFromContext(c)
	if !currentUser.IsAdmin && playlist.UserID != currentUser.ID && !playlist.Public {
		return utils.SendForbiddenError(c, "Access denied")
	}

	return c.JSON(playlist)
}

// CreatePlaylist handles creating a new playlist
func (h *PlaylistHandler) CreatePlaylist(c *fiber.Ctx) error {
	// Check authentication
	currentUser, ok := middleware.GetUserFromContext(c)
	if !ok {
		return utils.SendUnauthorizedError(c, "Authentication required")
	}

	var req struct {
		Name     string `json:"name"`
		Comment  string `json:"comment"`
		Public   bool   `json:"public"`
		TrackIDs []int64 `json:"track_ids"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid request body")
	}

	// Create playlist
	playlist := &models.Playlist{
		UserID:    currentUser.ID,
		Name:      req.Name,
		Comment:   req.Comment,
		Public:    req.Public,
		CreatedAt: time.Now(),
		ChangedAt: time.Now(),
	}

	if err := h.repo.CreatePlaylist(playlist); err != nil {
		return utils.SendInternalServerError(c, "Failed to create playlist")
	}

	// Add tracks to the playlist if provided
	if req.TrackIDs != nil && len(req.TrackIDs) > 0 {
		for i, trackID := range req.TrackIDs {
			playlistTrack := &models.PlaylistTrack{
				PlaylistID: playlist.ID,
				TrackID:    trackID,
				Position:   int32(i + 1), // Position starts from 1
			}
			if err := h.repo.AddTrackToPlaylist(playlistTrack); err != nil {
				return utils.SendInternalServerError(c, "Failed to add tracks to playlist")
			}
		}
	}

	// Get the playlist with tracks to return hydrated data
	playlistWithTracks, err := h.repo.GetPlaylistWithTracks(playlist.ID)
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to fetch playlist with tracks")
	}

	return c.Status(http.StatusCreated).JSON(playlistWithTracks)
}

// UpdatePlaylist handles updating a playlist
func (h *PlaylistHandler) UpdatePlaylist(c *fiber.Ctx) error {
	// Check authentication
	currentUser, ok := middleware.GetUserFromContext(c)
	if !ok {
		return utils.SendUnauthorizedError(c, "Authentication required")
	}

	playlistID, err := c.ParamsInt("id")
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid playlist ID")
	}

	var req struct {
		Name     *string `json:"name,omitempty"`
		Comment  *string `json:"comment,omitempty"`
		Public   *bool   `json:"public,omitempty"`
		TrackIDs *[]int64 `json:"track_ids,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid request body")
	}

	playlist, err := h.repo.GetPlaylistByID(int32(playlistID))
	if err != nil {
		return utils.SendNotFoundError(c, "Playlist")
	}

	// Check if user has permission to edit this playlist
	if !currentUser.IsAdmin && playlist.UserID != currentUser.ID {
		return utils.SendForbiddenError(c, "Access denied")
	}

	// Update fields if provided
	if req.Name != nil {
		playlist.Name = *req.Name
	}
	if req.Comment != nil {
		playlist.Comment = *req.Comment
	}
	if req.Public != nil {
		playlist.Public = *req.Public
	}
	playlist.ChangedAt = time.Now()

	if err := h.repo.UpdatePlaylist(playlist); err != nil {
		return utils.SendInternalServerError(c, "Failed to update playlist")
	}

	// Update tracks in the playlist if provided
	if req.TrackIDs != nil {
		// First, clear existing tracks
		if err := h.repo.ClearPlaylistTracks(playlist.ID); err != nil {
			return utils.SendInternalServerError(c, "Failed to clear existing tracks")
		}

		// Add new tracks
		for i, trackID := range *req.TrackIDs {
			playlistTrack := &models.PlaylistTrack{
				PlaylistID: playlist.ID,
				TrackID:    trackID,
				Position:   int32(i + 1), // Position starts from 1
			}
			if err := h.repo.AddTrackToPlaylist(playlistTrack); err != nil {
				return utils.SendInternalServerError(c, "Failed to add tracks to playlist")
			}
		}
	}

	// Get the playlist with tracks to return hydrated data
	playlistWithTracks, err := h.repo.GetPlaylistWithTracks(playlist.ID)
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to fetch playlist with tracks")
	}

	return c.JSON(playlistWithTracks)
}

// DeletePlaylist handles deleting a playlist
func (h *PlaylistHandler) DeletePlaylist(c *fiber.Ctx) error {
	// Check authentication
	currentUser, ok := middleware.GetUserFromContext(c)
	if !ok {
		return utils.SendUnauthorizedError(c, "Authentication required")
	}

	playlistID, err := c.ParamsInt("id")
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid playlist ID")
	}

	playlist, err := h.repo.GetPlaylistByID(int32(playlistID))
	if err != nil {
		return utils.SendNotFoundError(c, "Playlist")
	}

	// Check if user has permission to delete this playlist
	if !currentUser.IsAdmin && playlist.UserID != currentUser.ID {
		return utils.SendForbiddenError(c, "Access denied")
	}

	if err := h.repo.DeletePlaylist(int32(playlistID)); err != nil {
		return utils.SendInternalServerError(c, "Failed to delete playlist")
	}

	return c.JSON(fiber.Map{
		"status": "deleted",
	})
}