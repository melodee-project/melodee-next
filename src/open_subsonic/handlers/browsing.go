package handlers

import (
	"encoding/json"
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
	musicFolders := utils.MusicFolders{
		XMLName: xml.Name{Local: "musicFolders"},
	}
	
	for _, lib := range libraries {
		musicFolders.Folders = append(musicFolders.Folders, utils.MusicFolder{
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
	// Use more efficient query that only selects required fields
	var artists []models.Artist
	if err := h.db.Select("id, name, name_normalized, album_count_cached, created_at, last_scanned_at").Where("is_locked = ?", false).Order("name_normalized ASC").Find(&artists).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve artists")
	}

	// Organize artists by index based on normalized name
	artistMap := make(map[string][]utils.IndexArtist)
	ignoredArticles := "The El Le La Los Las" // As specified in the fixture

	for _, artist := range artists {
		// Get the first character of the normalized name for indexing
		// Apply normalization rules from the directory organization plan
		normalizedDisplayName := normalizeForIndexing(artist.NameNormalized)
		firstChar := getFirstCharForIndex(normalizedDisplayName)
		if firstChar == "" {
			firstChar = "#"  // Use "#" for names that don't start with a letter/number
		}

		// Create index artist
		indexArtist := utils.IndexArtist{
			ID:         int(artist.ID),
			Name:       artist.Name, // Display the original name, not the normalized one
			AlbumCount: int(artist.AlbumCountCached),
		}

		if !artist.CreatedAt.IsZero() {
			indexArtist.Created = utils.FormatTime(artist.CreatedAt)
		}
		if artist.LastScannedAt != nil && !artist.LastScannedAt.IsZero() {
			indexArtist.LastScanned = utils.FormatTime(*artist.LastScannedAt)
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
	var orderedIndexes []utils.Index
	var keys []string
	for k := range artistMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		orderedIndexes = append(orderedIndexes, utils.Index{
			Name:    k,
			Artists: artistMap[k],
		})
	}

	// Create response
	response := utils.SuccessResponse()
	indexes := utils.Indexes{
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

	// Enforce maximum size limit for performance
	if size > 500 {
		size = 500
	}

	// Query artists with pagination using more efficient select
	var artists []models.Artist
	if err := h.db.Select("id, name, album_count_cached, created_at, last_scanned_at").Where("is_locked = ?", false).
		Offset(offset).Limit(size).
		Order("name_normalized ASC"). // Add explicit ordering for consistent pagination
		Find(&artists).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve artists")
	}

	// Get total count
	var totalCount int64
	h.db.Model(&models.Artist{}).Where("is_locked = ?", false).Count(&totalCount)

	// Build response
	response := utils.SuccessResponse()
	artistsResp := utils.Artists{
		XMLName: xml.Name{Local: "artists"},
		Artists: make([]utils.IndexArtist, 0, len(artists)),
	}

	for _, artist := range artists {
		indexArtist := utils.IndexArtist{
			ID:         int(artist.ID),
			Name:       artist.Name,
			AlbumCount: int(artist.AlbumCountCached),
		}

		if !artist.CreatedAt.IsZero() {
			indexArtist.Created = utils.FormatTime(artist.CreatedAt)
		}
		if artist.LastScannedAt != nil && !artist.LastScannedAt.IsZero() {
			indexArtist.LastScanned = utils.FormatTime(*artist.LastScannedAt)
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
			return utils.SendOpenSubsonicError(c, 70, "utils.Artist not found")
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
	artistResp := utils.Artist{
		ID:         int(artist.ID),
		Name:       artist.Name,
		AlbumCount: int(artist.AlbumCountCached),
		Albums:     make([]utils.ArtistAlbum, 0, len(albums)),
	}

	for _, album := range albums {
		artistAlbum := utils.ArtistAlbum{
			ID:        int(album.ID),
			Name:      album.Name,
			Artist:    album.Artist.Name,
			ArtistID:  int(album.ArtistID),
			SongCount: int(album.SongCountCached),
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
			return utils.SendOpenSubsonicError(c, 70, "utils.Album not found")
		}
		return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve album")
	}

	// For now, return a basic response - more detailed implementation needed
	response := utils.SuccessResponse()
	albumInfo := utils.AlbumInfo{
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
		directory := utils.Directory{
			ID:   id,
			Name: artist.Name,
		}

		for _, album := range albums {
			directory.Children = append(directory.Children, utils.Child{
				ID:        int(album.ID),
				Parent:    id,
				IsDir:     true,
				Title:     album.Name,
				Album:     album.Name,
				Artist:    artist.Name,
				CoverArt:  fmt.Sprintf("al-%d", album.ID), // Placeholder
				Created:   utils.FormatTime(album.CreatedAt),
				Starred:   "", // If starred
				Duration:  int(album.DurationCached / 1000), // Convert milliseconds to seconds
			})
		}

		response.Directory = &directory
		return utils.SendResponse(c, response)
	} else if artistErr == gorm.ErrRecordNotFound {
		// Check if it's an album
		var album models.Album
		if err := h.db.Preload("Artist").First(&album, id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return utils.SendOpenSubsonicError(c, 70, "utils.Directory not found")
			}
			return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve directory")
		}

		// Get songs in this album
		var songs []models.Song
		if err := h.db.Where("album_id = ?", album.ID).Find(&songs).Error; err != nil {
			return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve songs")
		}

		response := utils.SuccessResponse()
		directory := utils.Directory{
			ID:     id,
			Parent: int(album.ArtistID),
			Name:   album.Name,
			Artist: album.Artist.Name,
		}

		for _, song := range songs {
			child := utils.Child{
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
				Genre:    extractGenreFromTags(song.Tags), // Extract genre from song tags
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

	return utils.SendOpenSubsonicError(c, 70, "utils.Directory not found")
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
			return utils.SendOpenSubsonicError(c, 70, "utils.Album not found")
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
	albumResp := &utils.Album{
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

	albumResp.Songs = make([]utils.Child, 0, len(songs))
	for _, song := range songs {
		child := utils.Child{
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
			Genre:    extractGenreFromTags(song.Tags), // Extract genre from song tags
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
	child := utils.Child{
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
		Genre:    extractGenreFromTags(song.Tags), // Extract genre from song tags
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
	// Aggregate genres from all songs in the database (primary source)
	genreMap := make(map[string]int)

	// Query songs to extract genres from tags
	var songs []models.Song
	if err := h.db.Select("tags").Find(&songs).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve songs for genre aggregation")
	}

	// Extract genres from all song tags and count occurrences
	for _, song := range songs {
		genre := extractGenreFromTags(song.Tags)
		if genre != "" {
			// Normalize the genre name for consistent counting
			normalizedGenre := normalizeGenreName(genre)
			if normalizedGenre != "" {
				genreMap[normalizedGenre]++
			}
		}
	}

	// Also check albums for genres if they have them
	var albums []models.Album
	if err := h.db.Find(&albums).Error; err != nil {
		// Don't return error, just continue with song-based genres
		fmt.Printf("Warning: Could not retrieve albums for genre aggregation: %v\n", err)
	} else {
		for _, album := range albums {
			for _, genre := range album.Genres {
				if genre != "" {
					normalizedGenre := normalizeGenreName(genre)
					if normalizedGenre != "" {
						genreMap[normalizedGenre]++
					}
				}
			}
		}
	}

	// Create response
	response := utils.SuccessResponse()
	genres := utils.Genres{
		XMLName: xml.Name{Local: "genres"},
		Genres:  []utils.Genre{}, // Will be populated below
	}

	// Convert the genre map to sorted slice
	type genreCount struct {
		Name  string
		Count int
	}
	var sortedGenres []genreCount

	for name, count := range genreMap {
		sortedGenres = append(sortedGenres, genreCount{Name: name, Count: count})
	}

	// Sort by name alphabetically (case-insensitive)
	sort.Slice(sortedGenres, func(i, j int) bool {
		return strings.ToLower(sortedGenres[i].Name) < strings.ToLower(sortedGenres[j].Name)
	})

	// Add to response with proper counts
	for _, gc := range sortedGenres {
		genres.Genres = append(genres.Genres, utils.Genre{
			Name:  gc.Name,
			Count: gc.Count,
		})
	}

	response.Genres = &genres
	return utils.SendResponse(c, response)
}

// normalizeGenreName normalizes genre names to ensure consistent counting
func normalizeGenreName(genre string) string {
	if genre == "" {
		return ""
	}

	// Trim whitespace
	genre = strings.TrimSpace(genre)

	// Handle common variations and clean up the genre name
	// For example: "Rock/Pop" might be treated as separate genres in some systems
	// For now, we'll keep compound genres as they are but normalize whitespace

	// Replace multiple spaces with single space
	for strings.Contains(genre, "  ") {
		genre = strings.ReplaceAll(genre, "  ", " ")
	}

	// Remove any trailing/leading spaces again after cleaning
	genre = strings.TrimSpace(genre)

	return genre
}

// Helper functions

// getFirstCharForIndex gets the first character for alphabetical indexing
// Following OpenSubsonic specification, this follows specific rules for indexing
func getFirstCharForIndex(name string) string {
	if len(name) == 0 {
		return "#"
	}

	// Apply normalization rules from DIRECTORY_ORGANIZATION_PLAN.md
	normalized := normalizeForIndexing(name)

	if len(normalized) == 0 {
		return "#"
	}

	// Get first character
	firstChar := string([]rune(normalized)[0])

	// Handle special characters
	if !isAlphanumeric([]rune(firstChar)[0]) {
		// Non-alphanumeric characters go to '#'
		return "#"
	}

	// Convert to uppercase for consistent indexing
	firstChar = strings.ToUpper(firstChar)

	// If it's a letter or digit, return the uppercase char
	if isLetter([]rune(firstChar)[0]) || isDigit([]rune(firstChar)[0]) {
		return firstChar
	} else {
		return "#" // Everything else goes to '#' group
	}
}

// isAlphanumeric checks if a character is alphanumeric
func isAlphanumeric(char rune) bool {
	return isLetter(char) || isDigit(char)
}

// isLetter checks if a character is a letter
func isLetter(char rune) bool {
	return (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z')
}

// isDigit checks if a character is a digit
func isDigit(char rune) bool {
	return char >= '0' && char <= '9'
}

// normalizeForIndexing applies the same normalization rules used for indexing
func normalizeForIndexing(name string) string {
	// Apply normalization rules similar to those in DIRECTORY_ORGANIZATION_PLAN.md
	// Step 1: Remove leading articles (case-insensitive)
	articles := []string{"the ", "a ", "an ", "le ", "la ", "les ", "el ", "los ", "las "}
	lowerName := strings.ToLower(name)

	for _, article := range articles {
		if strings.HasPrefix(lowerName, article) {
			name = strings.TrimSpace(name[len(article):])
			break
		}
	}

	// Step 2: Replace special characters and normalize
	// Replace & with and
	name = strings.ReplaceAll(name, "&", " and ")

	// Replace / and \ with -
	name = strings.ReplaceAll(name, "/", " - ")
	name = strings.ReplaceAll(name, "\\", " - ")

	// Replace other special characters that affect indexing
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "*", " ")
	name = strings.ReplaceAll(name, "?", "")
	name = strings.ReplaceAll(name, "!", "")
	name = strings.ReplaceAll(name, ":", "")
	name = strings.ReplaceAll(name, ";", "")

	// Remove periods but keep them if they're part of decimals
	// For simplicity, replace them in this context
	name = strings.ReplaceAll(name, ".", " ")

	// Normalize whitespace (multiple spaces, tabs, etc.) to single space
	name = strings.Join(strings.Fields(name), " ")

	return name
}

// extractGenreFromTags extracts genre from song tags
func extractGenreFromTags(tags []byte) string {
	// The tags field is a JSONB field in the database
	if tags == nil || len(tags) == 0 {
		return ""
	}

	// Parse the JSONB field to extract genre
	var tagData map[string]interface{}
	if err := json.Unmarshal(tags, &tagData); err != nil {
		return ""
	}

	// Look for various possible genre field names in the JSON
	possibleKeys := []string{"genre", "Genre", "GENRE", "music_genre", "style", "category", "tags", "GenreID3v1"}

	for _, key := range possibleKeys {
		if genreVal, ok := tagData[key]; ok {
			if genreStr, ok := genreVal.(string); ok {
				return genreStr
			}
			// Handle case where genre is an array
			if genreArr, ok := genreVal.([]interface{}); ok && len(genreArr) > 0 {
				if genreStr, ok := genreArr[0].(string); ok {
					return genreStr
				}
			}
			// Handle numeric genre IDs (common in ID3 tags)
			if genreNum, ok := genreVal.(float64); ok {
				return fmt.Sprintf("%.0f", genreNum) // Convert number to string
			}
		}
	}

	// If no genre found in top-level, check embedded objects
	if genreObj, ok := tagData["common"]; ok {
		if genreCommon, ok := genreObj.(map[string]interface{}); ok {
			if genreVal, ok := genreCommon["genre"]; ok {
				if genreStr, ok := genreVal.(string); ok {
					return genreStr
				}
			}
		}
	}

	return ""
}

// ExtractGenreFromTagsForTesting exports the function for testing
func ExtractGenreFromTagsForTesting(tags []byte) string {
	return extractGenreFromTags(tags)
}

// NormalizeGenreNameForTesting exports the function for testing
func NormalizeGenreNameForTesting(genre string) string {
	return normalizeGenreName(genre)
}

