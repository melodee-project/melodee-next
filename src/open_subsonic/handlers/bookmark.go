package handlers

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"melodee/internal/models"
	"melodee/open_subsonic/utils"
)

// BookmarkHandler handles OpenSubsonic bookmark endpoints
type BookmarkHandler struct {
	db *gorm.DB
}

// NewBookmarkHandler creates a new bookmark handler
func NewBookmarkHandler(db *gorm.DB) *BookmarkHandler {
	return &BookmarkHandler{
		db: db,
	}
}

// GetBookmarks returns all bookmarks for the authenticated user
func (h *BookmarkHandler) GetBookmarks(c *fiber.Ctx) error {
	user, ok := utils.GetUserFromContext(c)
	if !ok {
		return utils.SendOpenSubsonicError(c, 50, "User not authenticated")
	}

	var bookmarks []models.Bookmark
	// Preload Track, Album, Artist to build the Entry
	if err := h.db.Preload("Track").Preload("Track.Album").Preload("Track.Artist").Where("user_id = ?", user.ID).Find(&bookmarks).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 70, "Could not retrieve bookmarks")
	}

	response := utils.SuccessResponse()
	response.Bookmarks = &utils.Bookmarks{
		Bookmarks: make([]utils.Bookmark, len(bookmarks)),
	}

	for i, b := range bookmarks {
		// Fetch track details to populate Entry
		var track models.Track
		if err := h.db.Preload("Album").Preload("Artist").First(&track, b.TrackID).Error; err != nil {
			continue // Skip if track not found
		}

		entry := utils.Child{
			ID:          int(track.ID),
			Parent:      int(track.AlbumID),
			IsDir:       false,
			Title:       track.Name,
			Album:       track.Album.Name,
			Artist:      track.Artist.Name,
			Track:       int(track.SortOrder), // Assuming SortOrder is track number
			Year:        0,                    // Need to get year from album release date if available
			Genre:       "",                   // Need to get genre
			CoverArt:    strconv.Itoa(int(track.AlbumID)), // Use album ID for cover art
			Size:        0,                    // File size not in Track model?
			ContentType: "audio/mpeg",         // Defaulting, should be from track
			Suffix:      "mp3",                // Defaulting
			Duration:    int(track.Duration / 1000),
			BitRate:     int(track.BitRate),
			Path:        track.RelativePath,
			Created:     utils.FormatTime(track.CreatedAt),
		}
		
		if track.Album.ReleaseDate != nil {
			entry.Year = track.Album.ReleaseDate.Year()
		}
		if len(track.Album.Genres) > 0 {
			entry.Genre = track.Album.Genres[0]
		}

		response.Bookmarks.Bookmarks[i] = utils.Bookmark{
			Position: int64(b.Position),
			Username: user.Username,
			Comment:  b.Comment,
			Created:  utils.FormatTime(b.CreatedAt),
			Changed:  utils.FormatTime(b.UpdatedAt),
			Entry:    entry,
		}
	}

	return utils.SendResponse(c, response)
}

// CreateBookmark creates or updates a bookmark
func (h *BookmarkHandler) CreateBookmark(c *fiber.Ctx) error {
	user, ok := utils.GetUserFromContext(c)
	if !ok {
		return utils.SendOpenSubsonicError(c, 50, "User not authenticated")
	}

	idStr := c.Query("id")
	positionStr := c.Query("position")
	comment := c.Query("comment")

	if idStr == "" || positionStr == "" {
		return utils.SendOpenSubsonicError(c, 10, "Missing required parameters: id, position")
	}

	trackID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return utils.SendOpenSubsonicError(c, 10, "Invalid id parameter")
	}

	position, err := strconv.ParseInt(positionStr, 10, 32)
	if err != nil {
		return utils.SendOpenSubsonicError(c, 10, "Invalid position parameter")
	}

	// Check if track exists
	var track models.Track
	if err := h.db.First(&track, trackID).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 70, "Track not found")
	}

	// Check if bookmark exists
	var bookmark models.Bookmark
	result := h.db.Where("user_id = ? AND track_id = ?", user.ID, trackID).First(&bookmark)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Create new
			bookmark = models.Bookmark{
				UserID:    user.ID,
				TrackID:   trackID,
				Position:  int32(position),
				Comment:   comment,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := h.db.Create(&bookmark).Error; err != nil {
				return utils.SendOpenSubsonicError(c, 0, "Failed to create bookmark")
			}
		} else {
			return utils.SendOpenSubsonicError(c, 0, "Database error")
		}
	} else {
		// Update existing
		bookmark.Position = int32(position)
		if comment != "" {
			bookmark.Comment = comment
		}
		bookmark.UpdatedAt = time.Now()
		if err := h.db.Save(&bookmark).Error; err != nil {
			return utils.SendOpenSubsonicError(c, 0, "Failed to update bookmark")
		}
	}

	return utils.SendResponse(c, utils.SuccessResponse())
}

// DeleteBookmark deletes a bookmark
func (h *BookmarkHandler) DeleteBookmark(c *fiber.Ctx) error {
	user, ok := utils.GetUserFromContext(c)
	if !ok {
		return utils.SendOpenSubsonicError(c, 50, "User not authenticated")
	}

	idStr := c.Query("id")
	if idStr == "" {
		return utils.SendOpenSubsonicError(c, 10, "Missing required parameter: id")
	}

	trackID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return utils.SendOpenSubsonicError(c, 10, "Invalid id parameter")
	}

	if err := h.db.Where("user_id = ? AND track_id = ?", user.ID, trackID).Delete(&models.Bookmark{}).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to delete bookmark")
	}

	return utils.SendResponse(c, utils.SuccessResponse())
}
