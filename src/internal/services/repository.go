package services

import (
	"fmt"

	"gorm.io/gorm"
	"melodee/internal/models"
)

// Repository handles database operations for models
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new repository instance
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		db: db,
	}
}

// User operations
func (r *Repository) CreateUser(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *Repository) GetUserByID(id int64) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) UpdateUser(user *models.User) error {
	return r.db.Save(user).Error
}

func (r *Repository) DeleteUser(id int64) error {
	return r.db.Delete(&models.User{}, id).Error
}

// Playlist operations
func (r *Repository) CreatePlaylist(playlist *models.Playlist) error {
	return r.db.Create(playlist).Error
}

func (r *Repository) GetPlaylistByID(id int32) (*models.Playlist, error) {
	var playlist models.Playlist
	err := r.db.Preload("User").First(&playlist, id).Error
	if err != nil {
		return nil, err
	}
	return &playlist, nil
}

func (r *Repository) UpdatePlaylist(playlist *models.Playlist) error {
	return r.db.Save(playlist).Error
}

func (r *Repository) DeletePlaylist(id int32) error {
	return r.db.Delete(&models.Playlist{}, id).Error
}

// GetPlaylistsWithUser retrieves playlists with user information
func (r *Repository) GetPlaylistsWithUser(limit, offset int) ([]models.Playlist, int64, error) {
	var playlists []models.Playlist
	var total int64

	// Count total records
	err := r.db.Model(&models.Playlist{}).Count(&total).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count playlists: %w", err)
	}

	// Fetch records with associations
	err = r.db.Offset(offset).Limit(limit).
		Preload("User").
		Find(&playlists).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch playlists: %w", err)
	}

	return playlists, total, nil
}

// Artist operations
func (r *Repository) CreateArtist(artist *models.Artist) error {
	return r.db.Create(artist).Error
}

func (r *Repository) GetArtistByID(id int64) (*models.Artist, error) {
	var artist models.Artist
	err := r.db.First(&artist, id).Error
	if err != nil {
		return nil, err
	}
	return &artist, nil
}

// Album operations
func (r *Repository) CreateAlbum(album *models.Album) error {
	return r.db.Create(album).Error
}

func (r *Repository) GetAlbumByID(id int64) (*models.Album, error) {
	var album models.Album
	err := r.db.Preload("Artist").First(&album, id).Error
	if err != nil {
		return nil, err
	}
	return &album, nil
}

// Song operations
func (r *Repository) CreateSong(song *models.Song) error {
	return r.db.Create(song).Error
}

func (r *Repository) GetSongByID(id int64) (*models.Song, error) {
	var song models.Song
	err := r.db.Preload("Album").Preload("Artist").First(&song, id).Error
	if err != nil {
		return nil, err
	}
	return &song, nil
}

// Search operations
// SearchEntities searches for artists, albums, and songs based on the query and type
func (r *Repository) SearchEntities(query string, entityType string, limit, offset int) ([]interface{}, int64, error) {
	var results []interface{}
	var total int64

	// Normalize the query for searching
	normalizedQuery := fmt.Sprintf("%%%s%%", query)

	switch entityType {
	case "artist", "artists":
		// Search for artists
		var artists []models.Artist
		err := r.db.Model(&models.Artist{}).
			Where("name_normalized ILIKE ?", normalizedQuery).
			Count(&total).
			Limit(limit).Offset(offset).
			Order("name_normalized ASC, id ASC").
			Find(&artists).Error
		if err != nil {
			return nil, 0, fmt.Errorf("failed to search artists: %w", err)
		}

		// Convert to interface slice
		for _, artist := range artists {
			results = append(results, artist)
		}

	case "album", "albums":
		// Search for albums
		var albums []models.Album
		err := r.db.Model(&models.Album{}).
			Where("name_normalized ILIKE ?", normalizedQuery).
			Count(&total).
			Limit(limit).Offset(offset).
			Order("name_normalized ASC, id ASC").
			Preload("Artist"). // Preload associated artist
			Find(&albums).Error
		if err != nil {
			return nil, 0, fmt.Errorf("failed to search albums: %w", err)
		}

		// Convert to interface slice
		for _, album := range albums {
			results = append(results, album)
		}

	case "song", "songs":
		// Search for songs
		var songs []models.Song
		err := r.db.Model(&models.Song{}).
			Where("name_normalized ILIKE ?", normalizedQuery).
			Count(&total).
			Limit(limit).Offset(offset).
			Order("name_normalized ASC, id ASC").
			Preload("Album"). // Preload associated album
			Preload("Artist"). // Preload associated artist
			Find(&songs).Error
		if err != nil {
			return nil, 0, fmt.Errorf("failed to search songs: %w", err)
		}

		// Convert to interface slice
		for _, song := range songs {
			results = append(results, song)
		}

	case "any", "all", "":
		// Search across all entity types - this will require multiple queries
		// We'll search artists, albums, and songs separately and combine results if needed
		return r.searchAllEntities(query, limit, offset)

	default:
		return nil, 0, fmt.Errorf("unsupported entity type for search: %s", entityType)
	}

	return results, total, nil
}

// searchAllEntities performs a search across all entity types
func (r *Repository) searchAllEntities(query string, limit, offset int) ([]interface{}, int64, error) {
	// This is a simplified approach - in a real implementation, you might want to
	// use a full-text search or combine results in a more sophisticated way
	normalizedQuery := fmt.Sprintf("%%%s%%", query)

	var total int64

	// Count total across all entities
	var artistCount, albumCount, songCount int64
	r.db.Model(&models.Artist{}).Where("name_normalized ILIKE ?", normalizedQuery).Count(&artistCount)
	r.db.Model(&models.Album{}).Where("name_normalized ILIKE ?", normalizedQuery).Count(&albumCount)
	r.db.Model(&models.Song{}).Where("name_normalized ILIKE ?", normalizedQuery).Count(&songCount)
	total = artistCount + albumCount + songCount

	// For offset/limit pagination, we would need to implement a more complex solution
	// For now, let's get results from each type up to the limit
	var results []interface{}

	// Get artists
	var artists []models.Artist
	artistLimit := limit / 3 // Divide the limit between entity types
	if artistLimit < 1 {
		artistLimit = 1
	}
	r.db.Where("name_normalized ILIKE ?", normalizedQuery).
		Limit(artistLimit).Offset(0).
		Order("name_normalized ASC, id ASC").
		Find(&artists)

	for _, artist := range artists {
		results = append(results, artist)
	}

	// Get albums
	var albums []models.Album
	r.db.Where("name_normalized ILIKE ?", normalizedQuery).
		Limit(artistLimit).Offset(0).
		Order("name_normalized ASC, id ASC").
		Preload("Artist").
		Find(&albums)

	for _, album := range albums {
		results = append(results, album)
	}

	// Get songs
	var songs []models.Song
	r.db.Where("name_normalized ILIKE ?", normalizedQuery).
		Limit(artistLimit).Offset(0).
		Order("name_normalized ASC, id ASC").
		Preload("Album").
		Preload("Artist").
		Find(&songs)

	for _, song := range songs {
		results = append(results, song)
	}

	return results, total, nil
}

// SearchArtistsPaginated searches for artists with pagination
func (r *Repository) SearchArtistsPaginated(query string, limit, offset int) ([]models.Artist, int64, error) {
	var artists []models.Artist
	var total int64

	normalizedQuery := fmt.Sprintf("%%%s%%", query)

	err := r.db.
		Model(&models.Artist{}).
		Where("name_normalized ILIKE ?", normalizedQuery).
		Count(&total).
		Offset(offset).
		Limit(limit).
		Order("name_normalized ASC, id ASC"). // Consistent ordering as per spec
		Find(&artists).Error

	if err != nil {
		return nil, 0, fmt.Errorf("failed to search artists: %w", err)
	}

	return artists, total, nil
}

// SearchAlbumsPaginated searches for albums with pagination
func (r *Repository) SearchAlbumsPaginated(query string, limit, offset int) ([]models.Album, int64, error) {
	var albums []models.Album
	var total int64

	normalizedQuery := fmt.Sprintf("%%%s%%", query)

	err := r.db.
		Model(&models.Album{}).
		Where("name_normalized ILIKE ?", normalizedQuery).
		Count(&total).
		Offset(offset).
		Limit(limit).
		Order("name_normalized ASC, id ASC"). // Consistent ordering as per spec
		Preload("Artist").
		Find(&albums).Error

	if err != nil {
		return nil, 0, fmt.Errorf("failed to search albums: %w", err)
	}

	return albums, total, nil
}

// SearchSongsPaginated searches for songs with pagination
func (r *Repository) SearchSongsPaginated(query string, limit, offset int) ([]models.Song, int64, error) {
	var songs []models.Song
	var total int64

	normalizedQuery := fmt.Sprintf("%%%s%%", query)

	err := r.db.
		Model(&models.Song{}).
		Where("name_normalized ILIKE ?", normalizedQuery).
		Count(&total).
		Offset(offset).
		Limit(limit).
		Order("name_normalized ASC, id ASC"). // Consistent ordering as per spec
		Preload("Album").
		Preload("Artist").
		Find(&songs).Error

	if err != nil {
		return nil, 0, fmt.Errorf("failed to search songs: %w", err)
	}

	return songs, total, nil
}