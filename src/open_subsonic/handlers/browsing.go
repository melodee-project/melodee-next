package handlers

import (
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"melodee/internal/models"
	"melodee/open_subsonic/utils"
)

// BrowsingHandler handles OpenSubsonic browsing endpoints
type BrowsingHandler struct {
	db *gorm.DB
}

// NewBrowsingHandler creates a new browsing handler
func NewBrowsingHandler(db *gorm.DB) *BrowsingHandler {
	return &BrowsingHandler{
		db: db,
	}
}

// GetMusicFolders returns all configured music folders
func (h *BrowsingHandler) GetMusicFolders(c *fiber.Ctx) error {
	// Get all libraries from the database
	var libraries []models.Library
	if err := h.db.Find(&libraries).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve music folders")
	}

	// Create response
	response := utils.SuccessResponse()
	musicFolders := MusicFolders{
		XMLName: xml.Name{Local: "musicFolders"},
	}
	
	for _, lib := range libraries {
		musicFolders.Folders = append(musicFolders.Folders, MusicFolder{
			ID:   int(lib.ID),
			Name: lib.Name,
		})
	}
	response.MusicFolders = &musicFolders

	return utils.SendResponse(c, response)
}

// GetIndexes returns the indexed structure of artists
func (h *BrowsingHandler) GetIndexes(c *fiber.Ctx) error {
	username := c.Query("username", "")
	if username == "" {
		// Use the authenticated user
		// NOTE: In a real implementation, we'd get this from the auth middleware
		// For this implementation, returning an error as this is not fully implemented
		return utils.SendOpenSubsonicError(c, 50, "not authorized")
	}

	// Get last modified time (for now, using current time)
	lastModified := time.Now().UTC()

	// Query artists, organizing by first letter according to normalization rules
	var artists []models.Artist
	if err := h.db.Where("is_locked = ?", false).Find(&artists).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve artists")
	}

	// Organize artists by index based on normalized name
	artistMap := make(map[string][]IndexArtist)
	ignoredArticles := "The El Le La Los Las" // As specified in the fixture

	for _, artist := range artists {
		// Get the first letter of the normalized name for indexing
		firstChar := getFirstCharForIndex(artist.NameNormalized)
		if firstChar == "" {
			firstChar = "#"
		}

		// Create index artist
		indexArtist := IndexArtist{
			ID:         int(artist.ID),
			Name:       artist.Name,
			AlbumCount: artist.AlbumCountCached,
		}

		if !artist.CreatedAt.IsZero() {
			indexArtist.Created = utils.FormatTime(artist.CreatedAt)
		}
		if artist.LastScannedAt != nil && !artist.LastScannedAt.IsZero() {
			indexArtist.UpdatedAt = utils.FormatTime(*artist.LastScannedAt)
		}

		artistMap[firstChar] = append(artistMap[firstChar], indexArtist)
	}

	// Sort each index
	for _, indexArtists := range artistMap {
		sort.Slice(indexArtists, func(i, j int) bool {
			return strings.ToLower(indexArtists[i].Name) < strings.ToLower(indexArtists[j].Name)
		})
	}

	// Create ordered indexes
	var orderedIndexes []Index
	var keys []string
	for k := range artistMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		orderedIndexes = append(orderedIndexes, Index{
			Name:    k,
			Artists: artistMap[k],
		})
	}

	// Create response
	response := utils.SuccessResponse()
	indexes := Indexes{
		XMLName:         xml.Name{Local: "indexes"},
		LastModified:    utils.FormatTime(lastModified),
		IgnoredArticles: ignoredArticles,
		Indexes:         orderedIndexes,
	}
	response.Indexes = &indexes

	return utils.SendResponse(c, response)
}

// GetArtists returns all artists
func (h *BrowsingHandler) GetArtists(c *fiber.Ctx) error {
	// Get pagination parameters
	offset, size := utils.ParsePaginationParams(c)

	// Query artists with pagination
	var artists []models.Artist
	if err := h.db.Where("is_locked = ?", false).
		Offset(offset).Limit(size).
		Find(&artists).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve artists")
	}

	// Get total count
	var totalCount int64
	h.db.Model(&models.Artist{}).Where("is_locked = ?", false).Count(&totalCount)

	// Build response
	response := utils.SuccessResponse()
	artistsResp := Artists{
		XMLName: xml.Name{Local: "artists"},
		Artists: make([]IndexArtist, 0, len(artists)),
	}
	
	for _, artist := range artists {
		indexArtist := IndexArtist{
			ID:         int(artist.ID),
			Name:       artist.Name,
			AlbumCount: artist.AlbumCountCached,
		}

		if !artist.CreatedAt.IsZero() {
			indexArtist.Created = utils.FormatTime(artist.CreatedAt)
		}
		if artist.LastScannedAt != nil && !artist.LastScannedAt.IsZero() {
			indexArtist.UpdatedAt = utils.FormatTime(*artist.LastScannedAt)
		}

		artistsResp.Artists = append(artistsResp.Artists, indexArtist)
	}

	response.Artists = &artistsResp

	return utils.SendResponse(c, response)
}

// GetArtist returns details for a specific artist
func (h *BrowsingHandler) GetArtist(c *fiber.Ctx) error {
	id := c.QueryInt("id", -1)
	if id <= 0 {
		return utils.SendOpenSubsonicError(c, 10, "Missing required parameter id")
	}

	// Get the artist
	var artist models.Artist
	if err := h.db.First(&artist, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return utils.SendOpenSubsonicError(c, 70, "Artist not found")
		}
		return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve artist")
	}

	// Get albums for this artist
	var albums []models.Album
	if err := h.db.Where("artist_id = ? AND album_status = 'Ok'", artist.ID).Find(&albums).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve albums")
	}

	// Build response
	response := utils.SuccessResponse()
	artistResp := Artist{
		ID:         int(artist.ID),
		Name:       artist.Name,
		AlbumCount: artist.AlbumCountCached,
		Albums:     make([]ArtistAlbum, 0, len(albums)),
	}

	for _, album := range albums {
		artistAlbum := ArtistAlbum{
			ID:        int(album.ID),
			Name:      album.Name,
			Artist:    album.Artist.Name,
			ArtistID:  int(album.ArtistID),
			SongCount: album.SongCountCached,
		}

		if album.ReleaseDate != nil {
			artistAlbum.Year = album.ReleaseDate.Year()
		}

		if !album.CreatedAt.IsZero() {
			artistAlbum.Created = utils.FormatTime(album.CreatedAt)
		}

		artistResp.Albums = append(artistResp.Albums, artistAlbum)
	}

	response.Artist = &artistResp

	return utils.SendResponse(c, response)
}

// GetAlbumInfo returns album information
func (h *BrowsingHandler) GetAlbumInfo(c *fiber.Ctx) error {
	id := c.QueryInt("id", -1)
	if id <= 0 {
		return utils.SendOpenSubsonicError(c, 10, "Missing required parameter id")
	}

	// Get the album
	var album models.Album
	if err := h.db.Preload("Artist").First(&album, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return utils.SendOpenSubsonicError(c, 70, "Album not found")
		}
		return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve album")
	}

	// For now, return a basic response - more detailed implementation needed
	response := utils.SuccessResponse()
	albumInfo := AlbumInfo{
		ID: int(album.ID),
		// Add more fields as needed based on specifications
	}
	
	response.AlbumInfo = &albumInfo

	return utils.SendResponse(c, response)
}

// GetMusicDirectory returns files in a music directory
func (h *BrowsingHandler) GetMusicDirectory(c *fiber.Ctx) error {
	id := c.QueryInt("id", -1)
	if id <= 0 {
		return utils.SendOpenSubsonicError(c, 10, "Missing required parameter id")
	}

	// This endpoint can return either an artist or album if using the IDs from our system
	// Determine if it's an artist or album
	var artist models.Artist
	artistErr := h.db.First(&artist, id).Error
	
	if artistErr == nil {
		// It's an artist, return their albums
		var albums []models.Album
		if err := h.db.Where("artist_id = ? AND album_status = 'Ok'", artist.ID).Find(&albums).Error; err != nil {
			return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve albums")
		}

		response := utils.SuccessResponse()
		directory := Directory{
			ID:   id,
			Name: artist.Name,
		}

		for _, album := range albums {
			directory.Children = append(directory.Children, Child{
				ID:        int(album.ID),
				Parent:    id,
				IsDir:     true,
				Title:     album.Name,
				Album:     album.Name,
				Artist:    artist.Name,
				CoverArt:  fmt.Sprintf("al-%d", album.ID), // Placeholder
				Created:   utils.FormatTime(album.CreatedAt),
				Starred:   "", // If starred
				Duration:  album.DurationCached / 1000, // Convert milliseconds to seconds
				SongCount: album.SongCountCached,
			})
		}

		response.Directory = &directory
		return utils.SendResponse(c, response)
	} else if artistErr == gorm.ErrRecordNotFound {
		// Check if it's an album
		var album models.Album
		if err := h.db.Preload("Artist").First(&album, id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return utils.SendOpenSubsonicError(c, 70, "Directory not found")
			}
			return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve directory")
		}

		// Get songs in this album
		var songs []models.Song
		if err := h.db.Where("album_id = ?", album.ID).Find(&songs).Error; err != nil {
			return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve songs")
		}

		response := utils.SuccessResponse()
		directory := Directory{
			ID:     id,
			Parent: int(album.ArtistID),
			Name:   album.Name,
			Artist: album.Artist.Name,
		}

		for _, song := range songs {
			child := Child{
				ID:       int(song.ID),
				Parent:   id,
				IsDir:    false,
				Title:    song.Name,
				Album:    album.Name,
				Artist:   album.Artist.Name,
				CoverArt: fmt.Sprintf("al-%d", album.ID), // Placeholder
				Created:  utils.FormatTime(song.CreatedAt),
				Duration: int(song.Duration / 1000), // Convert milliseconds to seconds
				BitRate:  int(song.BitRate),
				Track:    int(song.SortOrder), // Assuming SortOrder is used as track number
				DiscNumber: int(song.SortOrder), // Simplified
				Year:     0, // Would come from album
				Genre:    "", // Would come from tags
				Size:     0, // Would need to get from file system
				ContentType: getContentType(song.FileName),
				Suffix:      getSuffix(song.FileName),
				Path:        song.RelativePath,
			}
			directory.Children = append(directory.Children, child)
		}

		response.Directory = &directory
		return utils.SendResponse(c, response)
	}

	return utils.SendOpenSubsonicError(c, 70, "Directory not found")
}

// GetAlbum returns album details with songs
func (h *BrowsingHandler) GetAlbum(c *fiber.Ctx) error {
	id := c.QueryInt("id", -1)
	if id <= 0 {
		return utils.SendOpenSubsonicError(c, 10, "Missing required parameter id")
	}

	// Get the album with artist info
	var album models.Album
	if err := h.db.Preload("Artist").First(&album, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return utils.SendOpenSubsonicError(c, 70, "Album not found")
		}
		return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve album")
	}

	// Get songs in this album
	var songs []models.Song
	if err := h.db.Where("album_id = ?", album.ID).Order("sort_order").Find(&songs).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve songs")
	}

	// Build response
	response := utils.SuccessResponse()
	albumResp := &Album{
		ID:        int(album.ID),
		Title:     album.Name,
		Album:     album.Name,
		Artist:    album.Artist.Name,
		ArtistID:  int(album.ArtistID),
		CoverArt:  fmt.Sprintf("al-%d", album.ID), // Placeholder
		SongCount: len(songs),
		Created:   utils.FormatTime(album.CreatedAt),
		Duration:  int(album.DurationCached / 1000), // Convert to seconds
	}

	if album.ReleaseDate != nil {
		albumResp.Year = album.ReleaseDate.Year()
	}

	albumResp.Songs = make([]Child, 0, len(songs))
	for _, song := range songs {
		child := Child{
			ID:       int(song.ID),
			Parent:   int(album.ID),
			IsDir:    false,
			Title:    song.Name,
			Album:    album.Name,
			Artist:   album.Artist.Name,
			CoverArt: fmt.Sprintf("al-%d", album.ID),
			Created:  utils.FormatTime(song.CreatedAt),
			Duration: int(song.Duration / 1000), // Convert to seconds
			BitRate:  int(song.BitRate),
			Track:    int(song.SortOrder),
			Year:     albumResp.Year, // Inherit year from album
			Genre:    "", // Would come from tags
			Size:     0, // Would need to get from file system
			ContentType: getContentType(song.FileName),
			Suffix:      getSuffix(song.FileName),
			Path:        song.RelativePath,
		}
		albumResp.Songs = append(albumResp.Songs, child)
	}

	response.Album = albumResp
	return utils.SendResponse(c, response)
}

// GetSong returns song details
func (h *BrowsingHandler) GetSong(c *fiber.Ctx) error {
	id := c.QueryInt("id", -1)
	if id <= 0 {
		return utils.SendOpenSubsonicError(c, 10, "Missing required parameter id")
	}

	var song models.Song
	if err := h.db.Preload("Album").Preload("Artist").First(&song, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return utils.SendOpenSubsonicError(c, 70, "Song not found")
		}
		return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve song")
	}

	// Build response
	response := utils.SuccessResponse()
	child := Child{
		ID:       int(song.ID),
		Parent:   int(song.AlbumID),
		IsDir:    false,
		Title:    song.Name,
		Album:    song.Album.Name,
		Artist:   song.Artist.Name,
		CoverArt: fmt.Sprintf("al-%d", song.AlbumID),
		Created:  utils.FormatTime(song.CreatedAt),
		Duration: int(song.Duration / 1000), // Convert to seconds
		BitRate:  int(song.BitRate),
		Track:    int(song.SortOrder),
		Genre:    "", // Would come from tags
		Size:     0, // Would need to get from file system
		ContentType: getContentType(song.FileName),
		Suffix:      getSuffix(song.FileName),
		Path:        song.RelativePath,
	}
	
	response.Song = &child
	return utils.SendResponse(c, response)
}

// GetGenres returns all genres
func (h *BrowsingHandler) GetGenres(c *fiber.Ctx) error {
	// For now, return a simple list of genres
	// In a real implementation, this would aggregate genres from all songs/albums
	
	// This is a simplified implementation - in reality, we'd need to query all 
	// songs and albums to extract their genre information
	response := utils.SuccessResponse()
	genres := Genres{
		XMLName: xml.Name{Local: "genres"},
		Genres:  []Genre{}, // Empty for now
	}
	
	// For demo purposes, adding a few sample genres
	sampleGenres := []string{"Rock", "Pop", "Jazz", "Classical", "Electronic", "Hip-Hop"}
	for _, name := range sampleGenres {
		genres.Genres = append(genres.Genres, Genre{
			Name:  name,
			Count: 1, // Simplified count
		})
	}
	
	response.Genres = &genres
	return utils.SendResponse(c, response)
}

// Helper functions

// getFirstCharForIndex gets the first character for alphabetical indexing
func getFirstCharForIndex(name string) string {
	if len(name) == 0 {
		return ""
	}
	
	// Apply normalization rules from DIRECTORY_ORGANIZATION_PLAN.md
	normalized := normalizeForIndexing(name)
	
	if len(normalized) == 0 {
		return ""
	}
	
	// Get first character
	firstChar := string([]rune(normalized)[0])
	
	// Capitalize
	return strings.ToUpper(firstChar)
}

// normalizeForIndexing applies the same normalization rules used for indexing
func normalizeForIndexing(name string) string {
	// Remove leading articles
	articles := []string{"the", "a", "an", "le", "la", "les", "el", "los", "las"}
	lowerName := strings.ToLower(name)
	
	for _, article := range articles {
		if strings.HasPrefix(lowerName, article+" ") {
			name = name[len(article)+1:]
			break
		}
	}
	
	// Replace & with and
	name = strings.ReplaceAll(name, "&", " and ")
	
	// Replace / with -
	name = strings.ReplaceAll(name, "/", " - ")
	
	// Remove periods
	name = strings.ReplaceAll(name, ".", " ")
	
	// Normalize whitespace
	name = strings.Join(strings.Fields(name), " ")
	
	return name
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

