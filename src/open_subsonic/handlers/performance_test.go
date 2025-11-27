package handlers

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"melodee/internal/models"
)

// TestGetIndexesPerformance tests the performance and behavior under large datasets
func TestGetIndexesPerformance(t *testing.T) {
	// Create a real test database for this performance test
	db, err := setupRealTestDatabase(t)
	require.NoError(t, err)
	defer func() {
		// Clean up test database
		db.Migrator().DropTable(&models.Artist{})
	}()

	// Populate with many artists to create a large response
	numArtists := 2000 // Large but reasonable number for testing
	err = populateArtistsForPerformanceTest(db, numArtists)
	require.NoError(t, err)

	// Create the handler
	handler := NewBrowsingHandler(db)

	// Create a Fiber app
	app := fiber.New()

	// Test the endpoint
	c := app.AcquireCtx(httptest.NewRequest("GET", "/getIndexes.view?username=testuser", nil))
	defer app.ReleaseCtx(c)

	err = handler.GetIndexes(c)
	assert.NoError(t, err)
	assert.Equal(t, 200, c.Response().StatusCode())

	// Verify that response is well-formed
	// The response is XML, so we can't easily parse it without importing xml package in this test
	// But we can at least verify that the request completed without errors
}

// TestGetArtistsPerformance tests the performance and behavior under large datasets
func TestGetArtistsPerformance(t *testing.T) {
	// Create a real test database for this performance test
	db, err := setupRealTestDatabase(t)
	require.NoError(t, err)
	defer func() {
		// Clean up test database
		db.Migrator().DropTable(&models.Artist{})
	}()

	// Populate with many artists to create a large response
	numArtists := 3000
	err = populateArtistsForPerformanceTest(db, numArtists)
	require.NoError(t, err)

	// Create the handler
	handler := NewBrowsingHandler(db)

	// Create a Fiber app
	app := fiber.New()

	// Test the endpoint with large offset to simulate pagination through large dataset
	c := app.AcquireCtx(httptest.NewRequest("GET", "/getArtists.view?offset=1500&size=100", nil))
	defer app.ReleaseCtx(c)

	err = handler.GetArtists(c)
	assert.NoError(t, err)
	assert.Equal(t, 200, c.Response().StatusCode())
}

// TestGetAlbumPerformance tests the performance of getAlbum.view with large number of songs
func TestGetAlbumPerformance(t *testing.T) {
	// Create a real test database for this performance test
	db, err := setupRealTestDatabase(t)
	require.NoError(t, err)
	defer func() {
		// Clean up test database
		db.Migrator().DropTable(&models.Artist{}, &models.Album{}, &models.Track{})
	}()

	// Create an artist
	artist := models.Artist{
		Name:           "Test Artist",
		NameNormalized: "test artist",
		AlbumCount:     1,
		IsLocked:       false,
	}
	err = db.Create(&artist).Error
	require.NoError(t, err)

	// Create an album
	album := models.Album{
		Name:         "Test Album",
		NameNormalized: "test album",
		ArtistID:     artist.ID,
		AlbumStatus:  "Ok",
		TrackCount:    0,
		Duration:     0,
	}
	err = db.Create(&album).Error
	require.NoError(t, err)

	// Populate with many songs for this album
	numSongs := 1000
	for i := 0; i < numSongs; i++ {
		song := models.Track{
			Name:         fmt.Sprintf("Song %d", i),
			NameNormalized: fmt.Sprintf("song %d", i),
			AlbumID:      album.ID,
			ArtistID:     artist.ID,
			SortOrder:    int64(i),
			Duration:     180000, // 3 minutes in milliseconds
			FileName:     fmt.Sprintf("song_%d.mp3", i),
			RelativePath: fmt.Sprintf("artist/test_album/song_%d.mp3", i),
		}
		err = db.Create(&song).Error
		if err != nil {
			t.Fatalf("Failed to create song %d: %v", i, err)
		}
	}

	// Update album counts
	album.TrackCount = int64(numSongs)
	err = db.Save(&album).Error
	require.NoError(t, err)

	// Create the handler
	handler := NewBrowsingHandler(db)

	// Create a Fiber app
	app := fiber.New()

	// Test the endpoint
	c := app.AcquireCtx(httptest.NewRequest("GET", "/getAlbum.view?id=1", nil))
	defer app.ReleaseCtx(c)

	err = handler.GetAlbum(c)
	assert.NoError(t, err)
	assert.Equal(t, 200, c.Response().StatusCode())
}

// TestSearch3Performance tests the performance and behavior of search3 under large datasets
func TestSearch3Performance(t *testing.T) {
	// Create a real test database for this performance test
	db, err := setupRealTestDatabase(t)
	require.NoError(t, err)
	defer func() {
		// Clean up test database
		db.Migrator().DropTable(&models.Artist{}, &models.Album{}, &models.Track{})
	}()

	// Populate with many artists, albums, and songs
	numArtists := 1500
	err = populateArtistsWithAlbumsAndSongsForPerformanceTest(db, numArtists, 3, 5) // 1500 artists, 3 albums each, 5 songs each
	require.NoError(t, err)

	// Create the handler
	handler := NewSearchHandler(db)

	// Create a Fiber app
	app := fiber.New()

	// Test the endpoint with a common search query
	c := app.AcquireCtx(httptest.NewRequest("GET", "/search3.view?query=test&offset=0&size=50", nil))
	defer app.ReleaseCtx(c)

	err = handler.Search3(c)
	assert.NoError(t, err)
	assert.Equal(t, 200, c.Response().StatusCode())
}

// BenchmarkGetIndexesPerformance benchmarks the performance of getIndexes.view endpoint
func BenchmarkGetIndexesPerformance(b *testing.B) {
	// Create a real test database for this benchmark
	db, err := setupRealTestDatabaseForBenchmark(b)
	if err != nil {
		b.Fatal("Failed to setup test DB:", err)
	}
	defer func() {
		// Clean up test database
		db.Migrator().DropTable(&models.Artist{})
	}()

	// Populate with many artists to create a large response
	numArtists := 5000
	err = populateArtistsForPerformanceTest(db, numArtists)
	if err != nil {
		b.Fatal("Failed to populate test data:", err)
	}

	// Create the handler
	handler := NewBrowsingHandler(db)

	// Create a Fiber app
	app := fiber.New()

	b.ResetTimer() // Start timing

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		c := app.AcquireCtx(httptest.NewRequest("GET", "/getIndexes.view?username=testuser", nil))
		b.StartTimer()

		err := handler.GetIndexes(c)
		if err != nil {
			b.Errorf("Handler returned error: %v", err)
		}

		app.ReleaseCtx(c)
	}
}

// BenchmarkGetArtistsPerformance benchmarks the performance of getArtists.view endpoint
func BenchmarkGetArtistsPerformance(b *testing.B) {
	// Create a real test database for this benchmark
	db, err := setupRealTestDatabaseForBenchmark(b)
	if err != nil {
		b.Fatal("Failed to setup test DB:", err)
	}
	defer func() {
		// Clean up test database
		db.Migrator().DropTable(&models.Artist{})
	}()

	// Populate with many artists to create a large response
	numArtists := 10000
	err = populateArtistsForPerformanceTest(db, numArtists)
	if err != nil {
		b.Fatal("Failed to populate test data:", err)
	}

	// Create the handler
	handler := NewBrowsingHandler(db)

	// Create a Fiber app
	app := fiber.New()

	b.ResetTimer() // Start timing

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		c := app.AcquireCtx(httptest.NewRequest("GET", "/getArtists.view?offset=0&size=500", nil))
		b.StartTimer()

		err := handler.GetArtists(c)
		if err != nil {
			b.Errorf("Handler returned error: %v", err)
		}

		app.ReleaseCtx(c)
	}
}

// BenchmarkGetAlbumPerformance benchmarks the performance of getAlbum.view endpoint
func BenchmarkGetAlbumPerformance(b *testing.B) {
	// Create a real test database for this benchmark
	db, err := setupRealTestDatabaseForBenchmark(b)
	if err != nil {
		b.Fatal("Failed to setup test DB:", err)
	}
	defer func() {
		// Clean up test database
		db.Migrator().DropTable(&models.Artist{}, &models.Album{}, &models.Track{})
	}()

	// Create an artist
	artist := models.Artist{
		Name:           "Test Artist",
		NameNormalized: "test artist",
		AlbumCount:     1,
		IsLocked:       false,
	}
	err = db.Create(&artist).Error
	if err != nil {
		b.Fatal("Failed to create artist:", err)
	}

	// Create an album
	album := models.Album{
		Name:         "Test Album",
		NameNormalized: "test album",
		ArtistID:     artist.ID,
		AlbumStatus:  "Ok",
		TrackCount:    0,
		Duration:     0,
	}
	err = db.Create(&album).Error
	if err != nil {
		b.Fatal("Failed to create album:", err)
	}

	// Populate with many songs for this album
	numSongs := 1000
	for i := 0; i < numSongs; i++ {
		song := models.Track{
			Name:         fmt.Sprintf("Song %d", i),
			NameNormalized: fmt.Sprintf("song %d", i),
			AlbumID:      album.ID,
			ArtistID:     artist.ID,
			SortOrder:    int64(i),
			Duration:     180000, // 3 minutes in milliseconds
			FileName:     fmt.Sprintf("song_%d.mp3", i),
			RelativePath: fmt.Sprintf("artist/test_album/song_%d.mp3", i),
		}
		err = db.Create(&song).Error
		if err != nil {
			b.Fatalf("Failed to create song %d: %v", i, err)
		}
	}

	// Update album counts
	album.TrackCount = int64(numSongs)
	err = db.Save(&album).Error
	if err != nil {
		b.Fatal("Failed to update album count:", err)
	}

	// Create the handler
	handler := NewBrowsingHandler(db)

	// Create a Fiber app
	app := fiber.New()

	b.ResetTimer() // Start timing

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		c := app.AcquireCtx(httptest.NewRequest("GET", "/getAlbum.view?id=1", nil))
		b.StartTimer()

		err := handler.GetAlbum(c)
		if err != nil {
			b.Errorf("Handler returned error: %v", err)
		}

		app.ReleaseCtx(c)
	}
}

// BenchmarkSearch3Performance benchmarks the performance of search3.view endpoint
func BenchmarkSearch3Performance(b *testing.B) {
	// Create a real test database for this benchmark
	db, err := setupRealTestDatabaseForBenchmark(b)
	if err != nil {
		b.Fatal("Failed to setup test DB:", err)
	}
	defer func() {
		// Clean up test database
		db.Migrator().DropTable(&models.Artist{}, &models.Album{}, &models.Track{})
	}()

	// Populate with many artists, albums, and songs
	numArtists := 5000
	err = populateArtistsWithAlbumsAndSongsForPerformanceTest(db, numArtists, 5, 10) // 5000 artists, 5 albums each, 10 songs each
	if err != nil {
		b.Fatal("Failed to populate test data:", err)
	}

	// Create the handler
	handler := NewSearchHandler(db)

	// Create a Fiber app
	app := fiber.New()

	b.ResetTimer() // Start timing

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		c := app.AcquireCtx(httptest.NewRequest("GET", "/search3.view?query=test&offset=0&size=50", nil))
		b.StartTimer()

		err := handler.Search3(c)
		if err != nil {
			b.Errorf("Handler returned error: %v", err)
		}

		app.ReleaseCtx(c)
	}
}

// Helper functions for creating test data

func setupRealTestDatabase(t *testing.T) (*gorm.DB, error) {
	// For testing purposes, we'll use an in-memory SQLite database
	// In a real application, you might want to use a proper test database
	dsn := "file::memory:?cache=shared"

	// Use SQLite for testing
	db, err := gorm.Open("sqlite", dsn)
	if err != nil {
		// If SQLite fails, try with gorm.Open using SQLite dialect
		db, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})
		if err != nil {
			return nil, err
		}
	}

	// Run migrations
	err = db.AutoMigrate(&models.Artist{}, &models.Album{}, &models.Track{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

func setupRealTestDatabaseForBenchmark(b *testing.B) (*gorm.DB, error) {
	// For benchmarking purposes, we'll use an in-memory SQLite database
	// In a real application, you might want to use a proper test database
	dsn := "file::memory:?cache=shared"

	// Use SQLite for testing
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Run migrations
	err = db.AutoMigrate(&models.Artist{}, &models.Album{}, &models.Track{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

// populateArtistsForPerformanceTest creates many artists in the database for performance testing
func populateArtistsForPerformanceTest(db *gorm.DB, count int) error {
	artists := make([]models.Artist, count)
	for i := 0; i < count; i++ {
		artists[i] = models.Artist{
			Name:           fmt.Sprintf("Artist %d", i),
			NameNormalized: fmt.Sprintf("artist %d", i),
			AlbumCount:     1,
			IsLocked:       false,
		}
	}

	err := db.CreateInBatches(artists, 1000).Error
	if err != nil {
		return err
	}

	return nil
}

// populateArtistsWithAlbumsAndSongsForPerformanceTest creates artists with albums and songs for comprehensive testing
func populateArtistsWithAlbumsAndSongsForPerformanceTest(db *gorm.DB, numArtists, albumsPerArtist, songsPerAlbum int) error {
	// Create artists first
	artists := make([]models.Artist, numArtists)
	for i := 0; i < numArtists; i++ {
		artists[i] = models.Artist{
			Name:           fmt.Sprintf("Artist %d", i),
			NameNormalized: fmt.Sprintf("artist %d", i),
			AlbumCount:     int64(albumsPerArtist),
			IsLocked:       false,
		}
	}

	err := db.CreateInBatches(artists, 1000).Error
	if err != nil {
		return err
	}

	// Create albums for each artist
	albums := make([]models.Album, 0, numArtists*albumsPerArtist)
	for i := 0; i < numArtists; i++ {
		for j := 0; j < albumsPerArtist; j++ {
			albums = append(albums, models.Album{
				Name:         fmt.Sprintf("Album %d for Artist %d", j, i),
				NameNormalized: fmt.Sprintf("album %d for artist %d", j, i),
				ArtistID:     artists[i].ID,
				AlbumStatus:  "Ok",
				TrackCount:    int64(songsPerAlbum),
			})
		}
	}

	err = db.CreateInBatches(albums, 1000).Error
	if err != nil {
		return err
	}

	// Create songs for each album
	songs := make([]models.Track, 0, len(albums)*songsPerAlbum)
	albumIdx := 0
	for i := 0; i < numArtists; i++ {
		for j := 0; j < albumsPerArtist; j++ {
			for k := 0; k < songsPerAlbum; k++ {
				songs = append(songs, models.Track{
					Name:         fmt.Sprintf("Song %d from Album %d Artist %d", k, j, i),
					NameNormalized: fmt.Sprintf("song %d from album %d artist %d", k, j, i),
					AlbumID:      albums[albumIdx].ID,
					ArtistID:     artists[i].ID,
					SortOrder:    int64(k),
					Duration:     180000, // 3 minutes
					FileName:     fmt.Sprintf("artist_%d_album_%d_song_%d.mp3", i, j, k),
					RelativePath: fmt.Sprintf("artist_%d/album_%d/song_%d.mp3", i, j, k),
				})
			}
			albumIdx++
		}
	}

	err = db.CreateInBatches(songs, 1000).Error
	if err != nil {
		return err
	}

	return nil
}