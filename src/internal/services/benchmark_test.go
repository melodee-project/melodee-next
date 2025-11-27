package services

import (
	"testing"
	"time"

	"melodee/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupBenchmarkDB creates an in-memory SQLite database for benchmarking
func setupBenchmarkDB() (*gorm.DB, func(), error) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}

	// Auto-migrate all models
	err = db.AutoMigrate(
		&models.User{}, &models.Library{}, &models.Artist{}, &models.Album{}, &models.Track{},
		&models.Playlist{}, &models.PlaylistTrack{}, &models.UserTrack{}, &models.UserAlbum{},
		&models.UserArtist{}, &models.UserPin{}, &models.Bookmark{}, &models.Player{},
		&models.PlayQueue{}, &models.SearchHistory{}, &models.Share{}, &models.ShareActivity{},
		&models.LibraryScanHistory{}, &models.Setting{}, &models.ArtistRelation{}, &models.RadioStation{},
		&models.Contributor{}, &models.CapacityStatus{},
	)
	if err != nil {
		return nil, nil, err
	}

	// Return cleanup function
	cleanup := func() {
		// In-memory SQLite doesn't require explicit cleanup
	}

	return db, cleanup, nil
}

// BenchmarkGetPlaylistsWithUser benchmarks the GetPlaylistsWithUser function with different sizes
func BenchmarkGetPlaylistsWithUser(b *testing.B) {
	db, cleanup, err := setupBenchmarkDB()
	if err != nil {
		b.Fatal(err)
	}
	defer cleanup()

	repo := NewRepository(db)

	// Setup: Create test data
	userID := int64(1)
	for i := 0; i < 100; i++ {
		playlist := &models.Playlist{
			UserID:    userID,
			Name:      "Benchmark Playlist " + string(rune(i+65)),
			Public:    i%2 == 0,
			CreatedAt: time.Now(),
			ChangedAt: time.Now(),
		}

		err := repo.CreatePlaylist(playlist)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limit := 10
		offset := (i % 10) * 10 // Cycle through different offsets
		_, _, err := repo.GetPlaylistsWithUser(limit, offset)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSearchArtistsPaginated benchmarks the SearchArtistsPaginated function
func BenchmarkSearchArtistsPaginated(b *testing.B) {
	db, cleanup, err := setupBenchmarkDB()
	if err != nil {
		b.Fatal(err)
	}
	defer cleanup()

	repo := NewRepository(db)

	// Setup: Create test data
	for i := 0; i < 200; i++ {
		artist := &models.Artist{
			Name:           "Benchmark Artist " + string(rune(i+65)),
			NameNormalized: "benchmark artist " + string(rune(i+65)),
			CreatedAt:      time.Now(),
		}

		err := repo.CreateArtist(artist)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := "benchmark"
		limit := 10
		offset := (i % 20) * 10 // Cycle through different offsets
		_, _, err := repo.SearchArtistsPaginated(query, limit, offset)
		if err != nil {
			b.Fatal(err)
		}
	}
}
