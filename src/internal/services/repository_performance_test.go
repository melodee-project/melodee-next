package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"melodee/internal/models"
)

// TestRepositoryPerformanceWithLargeOffsets tests performance with large offsets
func TestGetPlaylistsWithUserPerformance(t *testing.T) {
	db, cleanup, err := setupTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	repo := NewRepository(db)

	// Create a large number of playlists for performance testing
	for i := 0; i < 1000; i++ {
		userID := int64((i % 10) + 1) // Create playlists for 10 different users
		playlist := &models.Playlist{
			UserID:    userID,
			Name:      "Test Playlist " + string(rune(i+65)),
			Public:    i%3 == 0, // Make some public
			CreatedAt: time.Now(),
			ChangedAt: time.Now(),
		}

		err := repo.CreatePlaylist(playlist)
		assert.NoError(t, err)
		assert.Greater(t, playlist.ID, int32(0))
	}

	// Test with large offset to ensure performance
	limit := 50
	// Test with offset 500 (page 11 if page size is 50) - this is a large but reasonable offset
	offset := 500

	// Measure execution time
	start := time.Now()
	playlists, total, err := repo.GetPlaylistsWithUser(limit, offset)
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.Less(t, elapsed.Milliseconds(), int64(2000)) // Should complete in under 2 seconds
	assert.GreaterOrEqual(t, total, int64(1000))       // Should have at least 1000 playlists
	assert.LessOrEqual(t, len(playlists), limit)       // Should return at most 'limit' results

	t.Logf("Retrieved %d playlists with offset %d in %v. Total: %d",
		len(playlists), offset, elapsed, total)
}

// TestRepositoryPerformanceWithLargeSearch tests performance with large search result sets
func TestSearchPerformance(t *testing.T) {
	db, cleanup, err := setupTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	repo := NewRepository(db)

	// Create multiple test artists to test search performance
	for i := 0; i < 500; i++ {
		artist := &models.Artist{
			Name:           "Performance Test Artist " + string(rune(i+65)),
			NameNormalized: "performance test artist " + string(rune(i+65)),
			CreatedAt:      time.Now(),
		}

		err := repo.CreateArtist(artist)
		assert.NoError(t, err)
		assert.Greater(t, artist.ID, int64(0))
	}

	// Test search with pagination to ensure performance
	query := "performance test"
	limit := 25
	offset := 200 // Large offset to test pagination performance

	// Measure execution time for artist search
	start := time.Now()
	artists, total, err := repo.SearchArtistsPaginated(query, limit, offset)
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.Less(t, elapsed.Milliseconds(), int64(2000)) // Should complete in under 2 seconds
	assert.GreaterOrEqual(t, total, int64(500))        // Should have at least 500 results
	assert.LessOrEqual(t, len(artists), limit)         // Should return at most 'limit' results

	t.Logf("Searched artists with query '%s', offset %d in %v. Total: %d",
		query, offset, elapsed, total)

	// Test search albums performance
	start = time.Now()
	albums, total, err := repo.SearchAlbumsPaginated(query, limit, offset)
	elapsed = time.Since(start)

	assert.NoError(t, err)
	assert.Less(t, elapsed.Milliseconds(), int64(2000)) // Should complete in under 2 seconds

	t.Logf("Searched albums with query '%s', offset %d in %v. Total: %d",
		query, offset, elapsed, total)

	// Test search songs performance
	start = time.Now()
	songs, total, err := repo.SearchSongsPaginated(query, limit, offset)
	elapsed = time.Since(start)

	assert.NoError(t, err)
	assert.Less(t, elapsed.Milliseconds(), int64(2000)) // Should complete in under 2 seconds

	t.Logf("Searched songs with query '%s', offset %d in %v. Total: %d",
		query, offset, elapsed, total)
}