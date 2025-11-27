package handlers

import (
	"encoding/xml"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"melodee/internal/models"
	"melodee/open_subsonic/utils"
)

// SearchHandler handles OpenSubsonic search endpoints
type SearchHandler struct {
	db *gorm.DB
}

// NewSearchHandler creates a new search handler
func NewSearchHandler(db *gorm.DB) *SearchHandler {
	return &SearchHandler{
		db: db,
	}
}

// Search performs basic search for artists, albums, and songs
func (h *SearchHandler) Search(c *fiber.Ctx) error {
	query := c.Query("query", "")
	if query == "" {
		return utils.SendOpenSubsonicError(c, 10, "Missing required parameter query")
	}

	// Get pagination parameters with stricter limits for search operations
	offset, size := utils.ParseSearchPaginationParams(c)

	// Get results for each type
	artists, err := h.searchArtists(query, offset, size)
	if err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to search artists")
	}

	albums, err := h.searchAlbums(query, offset, size)
	if err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to search albums")
	}

	songs, err := h.searchSongs(query, offset, size)
	if err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to search songs")
	}

	// Create response
	response := utils.SuccessResponse()
	searchResult2 := utils.SearchResult2{
		Offset: offset,
		Size:   len(artists) + len(albums) + len(songs), // This is the number of results returned in this batch
		Artists: artists,
		Albums:  albums,
		Songs:   songs,
	}

	// Get total counts
	totalArtists, _ := h.countSearchArtists(query)
	totalAlbums, _ := h.countSearchAlbums(query)
	totalSongs, _ := h.countSearchSongs(query)
	searchResult2.TotalHits = totalArtists + totalAlbums + totalSongs

	response.SearchResult2 = &searchResult2

	return utils.SendResponse(c, response)
}

// Search2 performs enhanced search (OpenSubsonic 1.8.0+)
func (h *SearchHandler) Search2(c *fiber.Ctx) error {
	query := c.Query("query", "")
	if query == "" {
		return utils.SendOpenSubsonicError(c, 10, "Missing required parameter query")
	}

	// Get pagination parameters with stricter limits for search operations
	offset, size := utils.ParseSearchPaginationParams(c)

	// Get results for each type
	artists, err := h.searchArtists(query, offset, size)
	if err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to search artists")
	}

	albums, err := h.searchAlbums(query, offset, size)
	if err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to search albums")
	}

	songs, err := h.searchSongs(query, offset, size)
	if err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to search songs")
	}

	// Create response
	response := utils.SuccessResponse()
	searchResult2 := utils.SearchResult2{
		Offset:  offset,
		Size:    len(artists) + len(albums) + len(songs),
		Artists: artists,
		Albums:  albums,
		Songs:   songs,
	}

	// Get total counts
	totalArtists, _ := h.countSearchArtists(query)
	totalAlbums, _ := h.countSearchAlbums(query)
	totalSongs, _ := h.countSearchSongs(query)
	searchResult2.TotalHits = totalArtists + totalAlbums + totalSongs

	response.SearchResult2 = &searchResult2

	return utils.SendResponse(c, response)
}

// Search3 performs more comprehensive search (OpenSubsonic 1.11.0+)
func (h *SearchHandler) Search3(c *fiber.Ctx) error {
	query := c.Query("query", "")
	if query == "" {
		return utils.SendOpenSubsonicError(c, 10, "Missing required parameter query")
	}

	// Get pagination parameters with stricter limits for search operations
	offset, size := utils.ParseSearchPaginationParams(c)

	// Enforce maximum size limit for performance
	maxSize := 50 // Limit for each type to avoid too many results
	if size > maxSize {
		size = maxSize
	}

	// Get results for each type
	artists, err := h.searchArtists(query, offset, size)
	if err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to search artists")
	}

	albums, err := h.searchAlbums(query, offset, size)
	if err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to search albums")
	}

	songs, err := h.searchSongs(query, offset, size)
	if err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Failed to search songs")
	}

	// Create response
	response := utils.SuccessResponse()
	searchResult3 := utils.SearchResult3{
		XMLName: xml.Name{Local: "searchResult3"},
		Offset:  offset,
		Size:    len(artists) + len(albums) + len(songs),
		Artists: artists,
		Albums:  albums,
		Songs:   songs,
	}

	// Get total counts
	totalArtists, _ := h.countSearchArtists(query)
	totalAlbums, _ := h.countSearchAlbums(query)
	totalSongs, _ := h.countSearchSongs(query)
	searchResult3.TotalHits = totalArtists + totalAlbums + totalSongs

	response.SearchResult3 = &searchResult3

	return utils.SendResponse(c, response)
}

// searchArtists performs a search for artists matching the query
func (h *SearchHandler) searchArtists(query string, offset, size int) ([]utils.IndexArtist, error) {
	var artists []models.Artist

	// Enforce maximum size limit for performance
	if size > 100 {
		size = 100
	}

	// Build query with name normalization and full text search
	normalizedQuery := normalizeSearchQuery(query)

	// Use more efficient query that only selects required fields
	queryStmt := h.db.Select("id, name, album_count_cached, created_at, last_scanned_at").Where("name_normalized ILIKE ?", "%"+normalizedQuery+"%").Order("name_normalized ASC")

	// Apply pagination
	err := queryStmt.Offset(offset).Limit(size).Find(&artists).Error
	if err != nil {
		return nil, err
	}

	// Convert to response format
	result := make([]utils.IndexArtist, 0, len(artists))
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
			indexArtist.Starred = utils.FormatTime(*artist.LastScannedAt)
		}

		result = append(result, indexArtist)
	}

	return result, nil
}

// searchAlbums performs a search for albums matching the query
func (h *SearchHandler) searchAlbums(query string, offset, size int) ([]utils.SearchAlbum, error) {
	// Enforce maximum size limit for performance
	if size > 100 {
		size = 100
	}

	// Use a custom struct to hold the join results
	type AlbumWithArtist struct {
		models.Album
		ArtistName string `gorm:"column:artist_name"`
	}

	var albums []AlbumWithArtist

	// Build query with name normalization
	normalizedQuery := normalizeSearchQuery(query)

	// Use more efficient query that only selects required fields and joins
	queryStmt := h.db.Table("albums").
		Select("albums.id, albums.name, albums.artist_id, albums.release_date, albums.created_at, albums.duration, albums.track_count, artists.name as artist_name").
		Joins("LEFT JOIN artists ON albums.artist_id = artists.id").
		Where("albums.name_normalized ILIKE ?", "%"+normalizedQuery+"%").
		Order("albums.name_normalized ASC")

	// Apply pagination
	err := queryStmt.Offset(offset).Limit(size).Scan(&albums).Error
	if err != nil {
		return nil, err
	}

	// Convert to response format
	result := make([]utils.SearchAlbum, 0, len(albums))
	for _, album := range albums {
		searchAlbum := utils.SearchAlbum{
			ID:       int(album.ID),
			Name:     album.Name,
			Artist:   album.ArtistName, // Use the joined artist name
			ArtistID: int(album.ArtistID),
		}

		if album.ReleaseDate != nil {
			searchAlbum.Year = album.ReleaseDate.Year()
		}
		if !album.CreatedAt.IsZero() {
			searchAlbum.Created = utils.FormatTime(album.CreatedAt)
		}
		if album.DurationCached > 0 {
			searchAlbum.Duration = int(album.DurationCached / 1000) // Convert to seconds
		}
		if int(album.TrackCountCached) > 0 {
			searchAlbum.TrackCount = int(album.TrackCountCached)
		}

		result = append(result, searchAlbum)
	}

	return result, nil
}

// searchSongs performs a search for songs matching the query
func (h *SearchHandler) searchSongs(query string, offset, size int) ([]utils.Child, error) {
	// Enforce maximum size limit for performance
	if size > 100 {
		size = 100
	}

	// Use a custom struct to hold the join results
	type SongWithDetails struct {
		models.Track
		AlbumName  string `gorm:"column:album_name"`
		ArtistName string `gorm:"column:artist_name"`
	}

	var songs []SongWithDetails

	// Build query with name normalization
	normalizedQuery := normalizeSearchQuery(query)

	// Use more efficient query that only selects required fields and joins
	queryStmt := h.db.Table("tracks as songs").
		Select("songs.id, songs.name, songs.album_id, songs.artist_id, songs.duration, songs.bit_rate, songs.sort_order, songs.created_at, songs.relative_path, songs.file_name, albums.name as album_name, artists.name as artist_name").
		Joins("LEFT JOIN albums ON songs.album_id = albums.id").
		Joins("LEFT JOIN artists ON songs.artist_id = artists.id").
		Where("songs.name_normalized ILIKE ?", "%"+normalizedQuery+"%").
		Order("songs.name_normalized ASC")

	// Apply pagination
	err := queryStmt.Offset(offset).Limit(size).Scan(&songs).Error
	if err != nil {
		return nil, err
	}

	// Convert to response format
	result := make([]utils.Child, 0, len(songs))
	for _, song := range songs {
		child := utils.Child{
			ID:          int(song.ID),
			Parent:      int(song.AlbumID),
			IsDir:       false,
			Title:       song.Name,
			Album:       song.AlbumName, // Use the joined album name
			Artist:      song.ArtistName, // Use the joined artist name
			CoverArt:    getCoverArtID("album", song.AlbumID), // Placeholder
			Created:     utils.FormatTime(song.CreatedAt),
			Duration:    int(song.Duration / 1000), // Convert to seconds
			BitRate:     int(song.BitRate),
			Track:       int(song.SortOrder),
			Genre:       "", // Would come from tags
			Size:        0, // Would come from file system
			ContentType: getContentType(song.FileName),
			Suffix:      getSuffix(song.FileName),
			Path:        song.RelativePath,
		}

		result = append(result, child)
	}

	return result, nil
}

// countSearchArtists returns the total count of artists matching the query
func (h *SearchHandler) countSearchArtists(query string) (int, error) {
	var count int64
	normalizedQuery := normalizeSearchQuery(query)
	
	err := h.db.Model(&models.Artist{}).Where("name_normalized ILIKE ?", "%"+normalizedQuery+"%").Count(&count).Error
	return int(count), err
}

// countSearchAlbums returns the total count of albums matching the query
func (h *SearchHandler) countSearchAlbums(query string) (int, error) {
	var count int64
	normalizedQuery := normalizeSearchQuery(query)
	
	err := h.db.Model(&models.Album{}).Where("name_normalized ILIKE ?", "%"+normalizedQuery+"%").Count(&count).Error
	return int(count), err
}

// countSearchSongs returns the total count of songs matching the query
func (h *SearchHandler) countSearchSongs(query string) (int, error) {
	var count int64
	normalizedQuery := normalizeSearchQuery(query)
	
	err := h.db.Model(&models.Track{}).Where("name_normalized ILIKE ?", "%"+normalizedQuery+"%").Count(&count).Error
	return int(count), err
}

// normalizeSearchQuery normalizes a search query according to OpenSubsonic rules
func normalizeSearchQuery(query string) string {
	// Apply normalization rules similar to those in DIRECTORY_ORGANIZATION_PLAN.md
	// Remove leading articles
	articles := []string{"the", "a", "an", "le", "la", "les", "el", "los", "las"}
	lowerQuery := strings.ToLower(query)

	for _, article := range articles {
		if strings.HasPrefix(lowerQuery, article+" ") {
			query = query[len(article)+1:]
			break
		}
	}

	// Replace & with and
	query = strings.ReplaceAll(query, "&", " and ")

	// Replace / with -
	query = strings.ReplaceAll(query, "/", " - ")

	// Remove periods
	query = strings.ReplaceAll(query, ".", " ")

	// Normalize whitespace
	query = strings.Join(strings.Fields(query), " ")

	return query
}

// getContentType returns content type based on file extension

// getSuffix returns file extension without the dot

// getCoverArtID returns a cover art ID for an album ID

// SearchResult2 represents search results for search and search2 endpoints
type SearchResult2 struct {
	XMLName   xml.Name         `xml:"searchResult2"`
	Offset    int              `xml:"offset,attr"`
	Size      int              `xml:"size,attr"`
	TotalHits int              `xml:"totalHits,attr,omitempty"`
	Artists   []utils.IndexArtist `xml:"artist,omitempty"`
	Albums    []utils.SearchAlbum `xml:"album,omitempty"`
	Songs     []utils.Child    `xml:"song,omitempty"`
}

// SearchResult3 represents search results for search3 endpoint
type SearchResult3 struct {
	XMLName   xml.Name         `xml:"searchResult3"`
	Offset    int              `xml:"offset,attr"`
	Size      int              `xml:"size,attr"`
	TotalHits int              `xml:"totalHits,attr,omitempty"`
	Artists   []utils.IndexArtist `xml:"artist,omitempty"`
	Albums    []utils.SearchAlbum `xml:"album,omitempty"`
	Songs     []utils.Child    `xml:"song,omitempty"`
}

// SearchAlbum represents an album in search results
type SearchAlbum struct {
	ID        int    `xml:"id,attr"`
	Name      string `xml:"title,attr"` // In search results, album name is called 'title'
	Artist    string `xml:"artist,attr"`
	ArtistID  int    `xml:"artistId,attr"`
	CoverArt  string `xml:"coverArt,attr,omitempty"`
	TrackCount int    `xml:"songCount,attr"`
	Duration  int    `xml:"duration,attr,omitempty"`
	PlayCount int    `xml:"playCount,attr,omitempty"`
	Created   string `xml:"created,attr,omitempty"`
	Year      int    `xml:"year,attr,omitempty"`
	Genre     string `xml:"genre,attr,omitempty"`
}