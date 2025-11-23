package handlers

import (
	"encoding/xml"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"melodee/internal/models"
	"melodee/open_subsonic/utils"
)

// PlaylistHandler handles OpenSubsonic playlist endpoints
type PlaylistHandler struct {
	db *gorm.DB
}

// NewPlaylistHandler creates a new playlist handler
func NewPlaylistHandler(db *gorm.DB) *PlaylistHandler {
	return &PlaylistHandler{
		db: db,
	}
}

// GetPlaylists returns all playlists
func (h *PlaylistHandler) GetPlaylists(c *fiber.Ctx) error {
	// Get username parameter (to filter playlists)
	username := c.Query("username", "")
	
	// Get the authenticated user to handle proper permissions
	authenticatedUser, userOk := utils.GetUserFromContext(c)

	var playlists []models.Playlist

	query := h.db.Preload("User")

	if username != "" {
		// Get playlists for specific user
		var user models.User
		if err := h.db.Where("username = ?", username).First(&user).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return utils.SendOpenSubsonicError(c, 70, "User not found")
			}
			return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve user")
		}

		// Only show user's own private playlists if they're the authenticated user
		if userOk && authenticatedUser.ID == user.ID {
			// Authenticated user viewing their own playlists - show all
			query = query.Where("user_id = ?", user.ID)
		} else {
			// Viewing someone else's playlists - only show public ones
			query = query.Where("user_id = ? AND public = ?", user.ID, true)
		}
	} else {
		// If no specific username, get playlists for the authenticated user
		if userOk {
			query = query.Where("user_id = ? OR public = ?", authenticatedUser.ID, true)
		} else {
			// If not authenticated, only show public playlists
			query = query.Where("public = ?", true)
		}
	}
	
	if err := query.Find(&playlists).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve playlists")
	}

	// Create response
	response := utils.SuccessResponse()
	playlistsResp := utils.Playlists{}
	
	for _, playlist := range playlists {
		playlistResp := utils.Playlist{
			ID:        int(playlist.ID),
			Name:      playlist.Name,
			Comment:   playlist.Comment,
			Public:    playlist.Public,
			Owner:     playlist.User.Username,
			SongCount: int(playlist.SongCount),
			Created:   utils.FormatTime(playlist.CreatedAt),
			Changed:   utils.FormatTime(playlist.ChangedAt),
			Duration:  int(playlist.Duration / 1000), // Convert to seconds
		}
		
		if playlist.CoverArtID != nil {
			playlistResp.CoverArtID = int(*playlist.CoverArtID)
		}
		
		playlistsResp.Playlist = append(playlistsResp.Playlist, playlistResp)
	}
	
	response.Playlists = &playlistsResp
	return utils.SendResponse(c, response)
}

// GetPlaylist returns a specific playlist with its entries
func (h *PlaylistHandler) GetPlaylist(c *fiber.Ctx) error {
	id := c.QueryInt("id", -1)
	if id <= 0 {
		return utils.SendOpenSubsonicError(c, 10, "Missing required parameter id")
	}

	// Get the playlist with user info
	var playlist models.Playlist
	if err := h.db.Preload("User").Where("id = ?", id).First(&playlist).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return utils.SendOpenSubsonicError(c, 70, "Playlist not found")
		}
		return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve playlist")
	}

	// Get the songs in the playlist
	var playlistSongs []models.PlaylistSong
	if err := h.db.Preload("Song.Album").Preload("Song.Artist").Where("playlist_id = ?", id).Order("position").Find(&playlistSongs).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve playlist songs")
	}

	// Create response
	response := utils.SuccessResponse()
	playlistResp := utils.Playlist{
		ID:        int(playlist.ID),
		Name:      playlist.Name,
		Comment:   playlist.Comment,
		Public:    playlist.Public,
		Owner:     playlist.User.Username,
		SongCount: int(playlist.SongCount),
		Created:   utils.FormatTime(playlist.CreatedAt),
		Changed:   utils.FormatTime(playlist.ChangedAt),
		Duration:  int(playlist.Duration / 1000), // Convert to seconds
	}
	
	if playlist.CoverArtID != nil {
		playlistResp.CoverArtID = int(*playlist.CoverArtID)
	}
	
	// Add entries (songs) to the playlist
	for _, playlistSong := range playlistSongs {
		song := playlistSong.Song
		child := utils.Child{
			ID:       int(song.ID),
			Parent:   int(song.AlbumID),
			IsDir:    false,
			Title:    song.Name,
			Album:    song.Album.Name,
			Artist:   song.Artist.Name,
			CoverArt: getCoverArtID(song.AlbumID), // Placeholder
			Created:  utils.FormatTime(song.CreatedAt),
			Duration: int(song.Duration / 1000), // Convert to seconds
			BitRate:  int(song.BitRate),
			Track:    int(song.SortOrder),
			Genre:    "", // Would come from tags
			Size:     0, // Would come from file system
			ContentType: getContentType(song.FileName),
			Suffix:      getSuffix(song.FileName),
			Path:        song.RelativePath,
		}
		playlistResp.Entries = append(playlistResp.Entries, child)
	}
	
	response.Playlist = &playlistResp
	return utils.SendResponse(c, response)
}

// CreatePlaylist creates a new playlist or updates an existing one
func (h *PlaylistHandler) CreatePlaylist(c *fiber.Ctx) error {
	// Get parameters
	playlistID := c.QueryInt("playlistId", -1)
	name := c.Query("name", "")
	songIDsStr := c.Query("songId", "") // Can be a comma-separated list

	// At least one of playlistId or name must be provided
	if playlistID <= 0 && name == "" {
		return utils.SendOpenSubsonicError(c, 10, "Missing required parameter playlistId or name")
	}

	var playlist *models.Playlist
	var newPlaylist bool

	if playlistID > 0 {
		// Update existing playlist
		var existingPlaylist models.Playlist
		if err := h.db.First(&existingPlaylist, playlistID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return utils.SendOpenSubsonicError(c, 70, "Playlist not found")
			}
			return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve playlist")
		}
		playlist = &existingPlaylist
	} else {
		// Create new playlist
		// Get the authenticated user from context
		user, ok := utils.GetUserFromContext(c)
		if !ok {
			return utils.SendOpenSubsonicError(c, 50, "Authentication required")
		}

		newPlaylist = true
		playlist = &models.Playlist{
			UserID: user.ID,
			Name:   name,
			Public: false, // Default to private
		}
	}

	// If a name is provided, update the playlist name
	if name != "" {
		playlist.Name = name
	}

	// If song IDs are provided, update the playlist songs
	if songIDsStr != "" {
		songIDList, err := parseCommaSeparatedInts(songIDsStr)
		if err != nil {
			return utils.SendOpenSubsonicError(c, 10, "Invalid songId format")
		}

		// Clear existing playlist songs
		if !newPlaylist {
			if err := h.db.Where("playlist_id = ?", playlist.ID).Delete(&models.PlaylistSong{}).Error; err != nil {
				return utils.SendOpenSubsonicError(c, 0, "Failed to clear existing playlist songs")
			}
		}

		// Add new songs to the playlist
		for pos, songID := range songIDList {
			playlistSong := models.PlaylistSong{
				PlaylistID: int32(playlist.ID),
				SongID:     songID,
				Position:   int32(pos),
			}

			if err := h.db.Create(&playlistSong).Error; err != nil {
				return utils.SendOpenSubsonicError(c, 0, "Failed to add song to playlist")
			}
		}

		// Update song count
		playlist.SongCount = int32(len(songIDList))
	}

	// Save the playlist
	var result *gorm.DB
	if newPlaylist {
		result = h.db.Create(playlist)
	} else {
		result = h.db.Save(playlist)
	}

	if result.Error != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to save playlist")
	}

	// Return success response
	response := utils.SuccessResponse()
	// Get the user to populate owner information
	var ownerUser models.User
	if err := h.db.First(&ownerUser, playlist.UserID).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve playlist owner")
	}

	playlistResp := utils.Playlist{
		ID:        int(playlist.ID),
		Name:      playlist.Name,
		Public:    playlist.Public,
		Owner:     ownerUser.Username, // Use actual owner's username
		SongCount: int(playlist.SongCount),
		Created:   utils.FormatTime(playlist.CreatedAt),
		Changed:   utils.FormatTime(playlist.ChangedAt),
		Duration:  int(playlist.Duration / 1000), // Convert to seconds
	}

	if playlist.CoverArtID != nil {
		playlistResp.CoverArtID = int(*playlist.CoverArtID)
	}

	response.Playlist = &playlistResp
	return utils.SendResponse(c, response)
}

// UpdatePlaylist updates an existing playlist
func (h *PlaylistHandler) UpdatePlaylist(c *fiber.Ctx) error {
	// This is typically handled the same way as CreatePlaylist in OpenSubsonic
	return h.CreatePlaylist(c)
}

// DeletePlaylist deletes a playlist
func (h *PlaylistHandler) DeletePlaylist(c *fiber.Ctx) error {
	playlistID := c.QueryInt("id", -1)
	if playlistID <= 0 {
		return utils.SendOpenSubsonicError(c, 10, "Missing required parameter id")
	}

	// Check if playlist exists
	var playlist models.Playlist
	if err := h.db.First(&playlist, playlistID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return utils.SendOpenSubsonicError(c, 70, "Playlist not found")
		}
		return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve playlist")
	}

	// Delete the playlist and its associated songs
	if err := h.db.Where("playlist_id = ?", playlistID).Delete(&models.PlaylistSong{}).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to delete playlist songs")
	}

	if err := h.db.Delete(&models.Playlist{}, playlistID).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to delete playlist")
	}

	// Return success response (empty body for delete operations)
	response := utils.SuccessResponse()
	return utils.SendResponse(c, response)
}

// parseCommaSeparatedInts parses a comma-separated string of integers
func parseCommaSeparatedInts(s string) ([]int64, error) {
	if s == "" {
		return nil, nil
	}

	parts := strings.Split(s, ",")
	result := make([]int64, 0, len(parts))

	for _, part := range parts {
		num, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil {
			return nil, err
		}
		result = append(result, num)
	}

	return result, nil
}

// getContentType returns content type based on file extension
func getContentType(filename string) string {
	switch {
	case strings.HasSuffix(strings.ToLower(filename), ".mp3"):
		return "audio/mpeg"
	case strings.HasSuffix(strings.ToLower(filename), ".flac"):
		return "audio/flac"
	case strings.HasSuffix(strings.ToLower(filename), ".m4a"):
		return "audio/mp4"
	case strings.HasSuffix(strings.ToLower(filename), ".mp4"):
		return "audio/mp4"
	case strings.HasSuffix(strings.ToLower(filename), ".aac"):
		return "audio/aac"
	case strings.HasSuffix(strings.ToLower(filename), ".ogg"):
		return "audio/ogg"
	case strings.HasSuffix(strings.ToLower(filename), ".opus"):
		return "audio/opus"
	case strings.HasSuffix(strings.ToLower(filename), ".wav"):
		return "audio/wav"
	default:
		return "audio/mpeg" // Default
	}
}

// getSuffix returns file extension without the dot
func getSuffix(filename string) string {
	parts := strings.Split(filename, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return "mp3" // Default
}

// getCoverArtID returns a cover art ID for an album ID
func getCoverArtID(albumID int64) string {
	return "al-" + strconv.FormatInt(albumID, 10)
}