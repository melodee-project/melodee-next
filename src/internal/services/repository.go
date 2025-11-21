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