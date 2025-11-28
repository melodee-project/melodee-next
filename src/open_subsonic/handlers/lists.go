package handlers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"

	"melodee/internal/models"
	"melodee/open_subsonic/utils"
)

// GetAlbumList returns a list of albums based on criteria
func (h *BrowsingHandler) GetAlbumList(c *fiber.Ctx) error {
	return h.getAlbumListCommon(c, 1)
}

// GetAlbumList2 returns a list of albums based on criteria (Version 2)
func (h *BrowsingHandler) GetAlbumList2(c *fiber.Ctx) error {
	return h.getAlbumListCommon(c, 2)
}

func (h *BrowsingHandler) getAlbumListCommon(c *fiber.Ctx, version int) error {
	listType := c.Query("type", "alphabetical")
	offset, size := utils.ParsePaginationParams(c)

	// Enforce limits
	if size > 500 {
		size = 500
	}

	var albums []models.Album
	query := h.db.Model(&models.Album{}).Preload("Artist")

	// Apply filters based on type
	switch listType {
	case "random":
		if h.db.Dialector.Name() == "mysql" {
			query = query.Order("RAND()")
		} else {
			query = query.Order("RANDOM()") // Postgres/SQLite
		}
	case "newest":
		query = query.Order("created_at DESC")
	case "alphabetical", "byName":
		query = query.Order("name ASC")
	case "byYear":
		query = query.Order("release_date DESC")
	case "starred":
		// This requires joining with user_albums
		user, ok := utils.GetUserFromContext(c)
		if ok {
			query = query.Joins("JOIN user_albums ON user_albums.album_id = albums.id").
				Where("user_albums.user_id = ? AND user_albums.is_starred = ?", user.ID, true)
		} else {
			// If no user, return empty or error? OpenSubsonic usually requires auth.
			// For now, return empty if no user context (shouldn't happen with auth middleware)
			query = query.Where("1 = 0")
		}
	default:
		// Default to alphabetical
		query = query.Order("name ASC")
	}

	// Apply pagination
	if err := query.Offset(offset).Limit(size).Find(&albums).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve album list")
	}

	response := utils.SuccessResponse()

	if version == 2 {
		albumList := utils.AlbumList2{
			Albums: make([]utils.Album, 0, len(albums)),
		}
		for _, album := range albums {
			albumList.Albums = append(albumList.Albums, h.convertToAlbum(album))
		}
		response.AlbumList2 = &albumList
	} else {
		albumList := utils.AlbumList{
			Albums: make([]utils.Child, 0, len(albums)),
		}
		for _, album := range albums {
			albumList.Albums = append(albumList.Albums, h.convertToChild(album))
		}
		response.AlbumList = &albumList
	}

	return utils.SendResponse(c, response)
}

// GetRandomSongs returns random songs
func (h *BrowsingHandler) GetRandomSongs(c *fiber.Ctx) error {
	size := c.QueryInt("size", 10)
	if size > 500 {
		size = 500
	}

	genre := c.Query("genre")
	fromYear := c.QueryInt("fromYear", 0)
	toYear := c.QueryInt("toYear", 0)

	query := h.db.Model(&models.Track{}).Preload("Album").Preload("Artist")

	if genre != "" {
		// This is tricky because tags are JSONB.
		// Simple implementation: check if tags contain the genre string
		// Postgres specific JSONB query would be better
		if h.db.Dialector.Name() == "postgres" {
			query = query.Where("tags::text ILIKE ?", "%"+genre+"%")
		} else {
			query = query.Where("tags LIKE ?", "%"+genre+"%")
		}
	}

	if fromYear > 0 {
		// Assuming we can get year from album release date or tags
		// This is complex without a direct year column on tracks
		// For now, skipping year filter or implementing basic check if possible
	}

	if toYear > 0 {
		// Same as above
	}

	// Random sort
	if h.db.Dialector.Name() == "mysql" {
		query = query.Order("RAND()")
	} else {
		query = query.Order("RANDOM()")
	}

	var songs []models.Track
	if err := query.Limit(size).Find(&songs).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve random songs")
	}

	response := utils.SuccessResponse()
	randomSongs := utils.RandomSongs{
		Songs: make([]utils.Child, 0, len(songs)),
	}

	for _, song := range songs {
		randomSongs.Songs = append(randomSongs.Songs, h.convertTrackToChild(song))
	}

	response.RandomSongs = &randomSongs
	return utils.SendResponse(c, response)
}

// GetSongsByGenre returns songs by genre
func (h *BrowsingHandler) GetSongsByGenre(c *fiber.Ctx) error {
	genre := c.Query("genre")
	if genre == "" {
		return utils.SendOpenSubsonicError(c, 10, "Missing genre parameter")
	}

	offset, size := utils.ParsePaginationParams(c)

	query := h.db.Model(&models.Track{}).Preload("Album").Preload("Artist")

	if h.db.Dialector.Name() == "postgres" {
		query = query.Where("tags::text ILIKE ?", "%"+genre+"%")
	} else {
		query = query.Where("tags LIKE ?", "%"+genre+"%")
	}

	var songs []models.Track
	if err := query.Offset(offset).Limit(size).Find(&songs).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve songs by genre")
	}

	response := utils.SuccessResponse()
	songsByGenre := utils.SongsByGenre{
		Songs: make([]utils.Child, 0, len(songs)),
	}

	for _, song := range songs {
		songsByGenre.Songs = append(songsByGenre.Songs, h.convertTrackToChild(song))
	}

	response.SongsByGenre = &songsByGenre
	return utils.SendResponse(c, response)
}

// GetNowPlaying returns currently playing songs
func (h *BrowsingHandler) GetNowPlaying(c *fiber.Ctx) error {
	// Look for tracks played in the last 10 minutes
	tenMinutesAgo := time.Now().Add(-10 * time.Minute)

	var userTracks []models.UserTrack
	// We need to join with User to get username
	// And Track/Album/Artist for details
	// Note: UserTrack model might not have relations defined, so we fetch manually
	if err := h.db.Where("last_played_at > ?", tenMinutesAgo).
		Order("last_played_at DESC").
		Find(&userTracks).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve now playing")
	}

	response := utils.SuccessResponse()
	nowPlaying := utils.NowPlaying{
		Entries: make([]utils.NowPlayingEntry, 0, len(userTracks)),
	}

	if len(userTracks) == 0 {
		response.NowPlaying = &nowPlaying
		return utils.SendResponse(c, response)
	}

	// Collect IDs
	trackIDs := make([]int64, 0, len(userTracks))
	userIDs := make([]int64, 0, len(userTracks))
	for _, ut := range userTracks {
		if ut.TrackID > 0 {
			trackIDs = append(trackIDs, ut.TrackID)
		}
		if ut.UserID > 0 {
			userIDs = append(userIDs, ut.UserID)
		}
	}

	// Fetch Tracks
	var tracks []models.Track
	trackMap := make(map[int64]models.Track)
	if len(trackIDs) > 0 {
		if err := h.db.Preload("Album").Preload("Artist").Find(&tracks, trackIDs).Error; err != nil {
			return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve tracks")
		}
		for _, t := range tracks {
			trackMap[t.ID] = t
		}
	}

	// Fetch Users
	var users []models.User
	userMap := make(map[int64]string)
	if len(userIDs) > 0 {
		if err := h.db.Find(&users, userIDs).Error; err != nil {
			return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve users")
		}
		for _, u := range users {
			userMap[u.ID] = u.Username
		}
	}

	// Build response
	for _, ut := range userTracks {
		track, ok := trackMap[ut.TrackID]
		if !ok {
			continue
		}

		username := userMap[ut.UserID]
		minutesAgo := 0
		if ut.LastPlayedAt != nil {
			minutesAgo = int(time.Since(*ut.LastPlayedAt).Minutes())
		}

		albumName := ""
		if track.Album != nil {
			albumName = track.Album.Name
		}
		artistName := ""
		if track.Artist != nil {
			artistName = track.Artist.Name
		}

		entry := utils.NowPlayingEntry{
			ID:          int(track.ID),
			Parent:      int(track.AlbumID),
			IsDir:       false,
			Title:       track.Name,
			Album:       albumName,
			Artist:      artistName,
			CoverArt:    fmt.Sprintf("al-%d", track.AlbumID),
			Created:     utils.FormatTime(track.CreatedAt),
			Duration:    int(track.Duration / 1000),
			BitRate:     int(track.BitRate),
			Track:       int(track.SortOrder),
			Genre:       extractGenreFromTags(track.Tags),
			ContentType: getContentType(track.FileName),
			Suffix:      getSuffix(track.FileName),
			Path:        track.RelativePath,
			Username:    username,
			MinutesAgo:  minutesAgo,
			PlayerId:    1,                // Placeholder
			PlayerName:  "Melodee Player", // Placeholder
		}
		nowPlaying.Entries = append(nowPlaying.Entries, entry)
	}

	response.NowPlaying = &nowPlaying
	return utils.SendResponse(c, response)
}

// GetTopSongs returns top songs for an artist
func (h *BrowsingHandler) GetTopSongs(c *fiber.Ctx) error {
	artistName := c.Query("artist")
	count := c.QueryInt("count", 50)

	if artistName == "" {
		return utils.SendOpenSubsonicError(c, 10, "Missing artist parameter")
	}

	// Find artist first
	var artist models.Artist
	if err := h.db.Where("name = ?", artistName).First(&artist).Error; err != nil {
		// If not found, return empty list or error? OpenSubsonic usually returns empty list if artist not found
		response := utils.SuccessResponse()
		response.TopSongs = &utils.TopSongs{Artist: artistName}
		return utils.SendResponse(c, response)
	}

	// Get songs for artist
	// Ideally we sort by play count. We can aggregate from user_tracks
	// For now, let's just return random songs or first songs if no play stats
	// TODO: Implement proper aggregation of play counts

	var songs []models.Track
	if err := h.db.Where("artist_id = ?", artist.ID).Preload("Album").Preload("Artist").Limit(count).Find(&songs).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve top songs")
	}

	response := utils.SuccessResponse()
	topSongs := utils.TopSongs{
		Artist: artistName,
		Songs:  make([]utils.Child, 0, len(songs)),
	}

	for _, song := range songs {
		topSongs.Songs = append(topSongs.Songs, h.convertTrackToChild(song))
	}

	response.TopSongs = &topSongs
	return utils.SendResponse(c, response)
}

// GetSimilarSongs returns similar songs
func (h *BrowsingHandler) GetSimilarSongs(c *fiber.Ctx) error {
	return h.getSimilarSongsCommon(c, 1)
}

// GetSimilarSongs2 returns similar songs (Version 2)
func (h *BrowsingHandler) GetSimilarSongs2(c *fiber.Ctx) error {
	return h.getSimilarSongsCommon(c, 2)
}

func (h *BrowsingHandler) getSimilarSongsCommon(c *fiber.Ctx, version int) error {
	id := c.QueryInt("id", -1)
	count := c.QueryInt("count", 50)

	if id <= 0 {
		// OpenSubsonic allows artist ID or artist name? Spec says "id" (ID of artist)
		// But some clients might send other things. Let's stick to ID.
		return utils.SendOpenSubsonicError(c, 10, "Missing id parameter")
	}

	// For now, just return random songs from the same artist or random songs from DB
	// Real implementation needs similarity graph

	// Let's get the artist first to ensure it exists
	var artist models.Artist
	if err := h.db.First(&artist, id).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 70, "Artist not found")
	}

	// Return random songs from this artist for now (as "similar" to the artist)
	// Or better: random songs from OTHER artists in the same genre?
	// Let's try to find genre of this artist (from their albums/tracks)

	var songs []models.Track
	query := h.db.Model(&models.Track{}).Preload("Album").Preload("Artist")

	if h.db.Dialector.Name() == "mysql" {
		query = query.Order("RAND()")
	} else {
		query = query.Order("RANDOM()")
	}

	if err := query.Limit(count).Find(&songs).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve similar songs")
	}

	response := utils.SuccessResponse()

	childSongs := make([]utils.Child, 0, len(songs))
	for _, song := range songs {
		childSongs = append(childSongs, h.convertTrackToChild(song))
	}

	if version == 2 {
		response.SimilarSongs2 = &utils.SimilarSongs2{Songs: childSongs}
	} else {
		response.SimilarSongs = &utils.SimilarSongs{Songs: childSongs}
	}

	return utils.SendResponse(c, response)
}

// Helper methods

func (h *BrowsingHandler) convertToAlbum(album models.Album) utils.Album {
	a := utils.Album{
		ID:         int(album.ID),
		Title:      album.Name,
		Album:      album.Name,
		Artist:     album.Artist.Name,
		ArtistID:   int(album.ArtistID),
		CoverArt:   fmt.Sprintf("al-%d", album.ID),
		TrackCount: int(album.TrackCountCached),
		Created:    utils.FormatTime(album.CreatedAt),
		Duration:   int(album.DurationCached / 1000),
	}
	if album.ReleaseDate != nil {
		a.Year = album.ReleaseDate.Year()
	}
	return a
}

func (h *BrowsingHandler) convertToChild(album models.Album) utils.Child {
	c := utils.Child{
		ID:       int(album.ID),
		IsDir:    true,
		Title:    album.Name,
		Album:    album.Name,
		Artist:   album.Artist.Name,
		CoverArt: fmt.Sprintf("al-%d", album.ID),
		Created:  utils.FormatTime(album.CreatedAt),
		Duration: int(album.DurationCached / 1000),
	}
	if album.ReleaseDate != nil {
		c.Year = album.ReleaseDate.Year()
	}
	return c
}

func (h *BrowsingHandler) convertTrackToChild(track models.Track) utils.Child {
	albumName := ""
	if track.Album != nil {
		albumName = track.Album.Name
	}
	artistName := ""
	if track.Artist != nil {
		artistName = track.Artist.Name
	}

	return utils.Child{
		ID:          int(track.ID),
		Parent:      int(track.AlbumID),
		IsDir:       false,
		Title:       track.Name,
		Album:       albumName,
		Artist:      artistName,
		CoverArt:    fmt.Sprintf("al-%d", track.AlbumID),
		Created:     utils.FormatTime(track.CreatedAt),
		Duration:    int(track.Duration / 1000),
		BitRate:     int(track.BitRate),
		Track:       int(track.SortOrder),
		Genre:       extractGenreFromTags(track.Tags),
		ContentType: getContentType(track.FileName),
		Suffix:      getSuffix(track.FileName),
		Path:        track.RelativePath,
	}
}
