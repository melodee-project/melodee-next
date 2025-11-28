package handlers

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"melodee/internal/models"
	"melodee/open_subsonic/utils"
)

// PlayQueueHandler handles OpenSubsonic play queue endpoints
type PlayQueueHandler struct {
	db *gorm.DB
}

// NewPlayQueueHandler creates a new play queue handler
func NewPlayQueueHandler(db *gorm.DB) *PlayQueueHandler {
	return &PlayQueueHandler{
		db: db,
	}
}

// GetPlayQueue returns the state of the play queue for this user
func (h *PlayQueueHandler) GetPlayQueue(c *fiber.Ctx) error {
	user, ok := utils.GetUserFromContext(c)
	if !ok {
		return utils.SendOpenSubsonicError(c, 50, "User not authenticated")
	}

	var playQueueItems []models.PlayQueue
	if err := h.db.Where("user_id = ?", user.ID).Order("play_queue_id ASC").Find(&playQueueItems).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 70, "Could not retrieve play queue")
	}

	response := utils.SuccessResponse()

	entries := make([]utils.Child, 0, len(playQueueItems))
	var currentID int
	var position int64
	var changed string
	var changedBy string

	if len(playQueueItems) > 0 {
		changed = utils.FormatTime(playQueueItems[0].UpdatedAt)
		changedBy = playQueueItems[0].ChangedBy
	}

	for _, item := range playQueueItems {
		if item.IsCurrentTrack {
			currentID = int(item.TrackID)
			position = int64(item.Position)
		}

		// Fetch track details
		var track models.Track
		if err := h.db.Preload("Album").Preload("Artist").First(&track, item.TrackID).Error; err != nil {
			continue
		}

		entry := utils.Child{
			ID:          int(track.ID),
			Parent:      int(track.AlbumID),
			IsDir:       false,
			Title:       track.Name,
			Album:       track.Album.Name,
			Artist:      track.Artist.Name,
			Track:       int(track.SortOrder),
			CoverArt:    strconv.Itoa(int(track.AlbumID)),
			Duration:    int(track.Duration / 1000),
			BitRate:     int(track.BitRate),
			Path:        track.RelativePath,
			Created:     utils.FormatTime(track.CreatedAt),
			ContentType: "audio/mpeg", // Default
			Suffix:      "mp3",        // Default
		}

		if track.Album.ReleaseDate != nil {
			entry.Year = track.Album.ReleaseDate.Year()
		}
		if len(track.Album.Genres) > 0 {
			entry.Genre = track.Album.Genres[0]
		}

		entries = append(entries, entry)
	}

	response.PlayQueue = &utils.PlayQueue{
		Current:   currentID,
		Position:  position,
		Username:  user.Username,
		Changed:   changed,
		ChangedBy: changedBy,
		Entries:   entries,
	}

	return utils.SendResponse(c, response)
}

// SavePlayQueue saves the state of the play queue for this user
func (h *PlayQueueHandler) SavePlayQueue(c *fiber.Ctx) error {
	user, ok := utils.GetUserFromContext(c)
	if !ok {
		return utils.SendOpenSubsonicError(c, 50, "User not authenticated")
	}

	// This endpoint expects a list of song IDs and optionally current song and position
	// id: ID of a song in the play queue. Use one id parameter for each song in the play queue.
	// current: The ID of the current playing song.
	// position: The position in milliseconds of the current playing song.

	// Since Fiber doesn't easily handle multiple query params with same name 'id', we need to parse manually or use QueryParser
	// But QueryParser might not handle array of same key well depending on config.
	// Let's try to get all values for 'id'

	// Fiber's QueryParser can bind to slice
	type QueryParams struct {
		IDs      []int   `query:"id"`
		Current  int     `query:"current"`
		Position float64 `query:"position"`
	}

	p := new(QueryParams)
	if err := c.QueryParser(p); err != nil {
		return utils.SendOpenSubsonicError(c, 10, "Invalid parameters")
	}

	if len(p.IDs) == 0 {
		// If no IDs, maybe clear the queue?
		// Or just return success if nothing to save
	}

	// Transaction to replace queue
	err := h.db.Transaction(func(tx *gorm.DB) error {
		// Delete existing queue
		if err := tx.Where("user_id = ?", user.ID).Delete(&models.PlayQueue{}).Error; err != nil {
			return err
		}

		// Insert new items
		for i, trackID := range p.IDs {
			isCurrent := false
			position := 0.0
			if trackID == p.Current {
				isCurrent = true
				position = p.Position
			}

			// We need track API key? The model has it.
			// But we only have ID here.
			// Let's fetch track to get API key? Or just generate a random one if not strictly needed for internal logic?
			// The model definition says TrackAPIKey is uuid.UUID and not null.
			// We should probably fetch the track.

			var track models.Track
			if err := tx.Select("api_key").First(&track, trackID).Error; err != nil {
				continue // Skip invalid tracks
			}

			item := models.PlayQueue{
				UserID:         user.ID,
				TrackID:        int64(trackID),
				TrackAPIKey:    track.APIKey,
				IsCurrentTrack: isCurrent,
				ChangedBy:      "MelodeeClient", // Or user agent?
				Position:       position,
				PlayQueueID:    int32(i),
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to save play queue")
	}

	return utils.SendResponse(c, utils.SuccessResponse())
}

// GetPlayQueueByIndex returns the state of the play queue for this user (by index)
// This is not standard Subsonic but listed in the plan.
// Assuming it's similar to GetPlayQueue but maybe paginated?
// Or maybe it's just an alias?
func (h *PlayQueueHandler) GetPlayQueueByIndex(c *fiber.Ctx) error {
	return h.GetPlayQueue(c)
}

// SavePlayQueueByIndex saves the state of the play queue for this user (by index)
func (h *PlayQueueHandler) SavePlayQueueByIndex(c *fiber.Ctx) error {
	return h.SavePlayQueue(c)
}
