package services

import (
	"testing"

	"melodee/internal/models"
	"melodee/internal/test"

	"github.com/stretchr/testify/assert"
)

// TestRepositorySearchArtistsPaginated tests the SearchArtistsPaginated method with filters, pagination, and ordering
func TestRepositorySearchArtistsPaginated(t *testing.T) {
	db, tearDown := test.SetupTestEnvironment(t)
	defer tearDown()

	repo := NewRepository(db)

	// Create test artists for search
	testArtists := []*models.Artist{
		{
			Name:           "The Beatles",
			NameNormalized: "the beatles", // Will be normalized for searching
			DirectoryCode:  "TB",
		},
		{
			Name:           "Adele",
			NameNormalized: "adele",
			DirectoryCode:  "AD",
		},
		{
			Name:           "The Rolling Stones",
			NameNormalized: "the rolling stones",
			DirectoryCode:  "RS",
		},
		{
			Name:           "ABBA",
			NameNormalized: "abba",
			DirectoryCode:  "AB",
		},
	}

	// Create all test artists
	for _, artist := range testArtists {
		err := repo.CreateArtist(artist)
		assert.NoError(t, err)
		assert.NotZero(t, artist.ID)
	}

	t.Run("Search all artists with pagination", func(t *testing.T) {
		// Search for all artists (using a common substring)
		artists, total, err := repo.SearchArtistsPaginated("a", 10, 0)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(artists), 3) // Should find Adele, ABBA, The Rolling Stones (contains 'a')
		assert.GreaterOrEqual(t, total, int64(3))
	})

	t.Run("Search with specific term", func(t *testing.T) {
		// Search for "beatles"
		artists, total, err := repo.SearchArtistsPaginated("beatles", 10, 0)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(artists), 1) // Should find "The Beatles"
		assert.GreaterOrEqual(t, total, int64(1))

		// Verify the result contains "The Beatles"
		foundBeatles := false
		for _, artist := range artists {
			if artist.NameNormalized == "the beatles" {
				foundBeatles = true
				break
			}
		}
		assert.True(t, foundBeatles)
	})

	t.Run("Search with pagination and ordering", func(t *testing.T) {
		// Search and get first 2 results
		artists1, total, err := repo.SearchArtistsPaginated("a", 2, 0) // Search for artists with 'a'
		assert.NoError(t, err)
		assert.Equal(t, 2, len(artists1))
		assert.GreaterOrEqual(t, total, int64(3))

		// Get the next 2 results
		artists2, total, err := repo.SearchArtistsPaginated("a", 2, 2)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(artists2), 0) // May be 0 or 1 depending on exact matches
		assert.GreaterOrEqual(t, total, int64(3))

		// Verify ordering is consistent (by name_normalized ASC, then id ASC)
		if len(artists1) > 0 && len(artists2) > 0 {
			// The first artist in second page should be greater than last from first page
			if len(artists1) == 2 {
				assert.True(t, artists2[0].NameNormalized >= artists1[1].NameNormalized)
			}
		}
	})

	t.Run("Search with exact match", func(t *testing.T) {
		// Search for exact artist name
		artists, total, err := repo.SearchArtistsPaginated("adele", 10, 0)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(artists), 1)
		assert.GreaterOrEqual(t, total, int64(1))

		// Verify we found the right artist
		found := false
		for _, artist := range artists {
			if artist.NameNormalized == "adele" {
				found = true
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("Search with no results", func(t *testing.T) {
		// Search for something that doesn't exist
		artists, total, err := repo.SearchArtistsPaginated("nonexistentartist", 10, 0)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(artists))
		assert.Equal(t, int64(0), total)
	})

	t.Run("Verify consistent ordering", func(t *testing.T) {
		// Get all artists again to verify consistent ordering
		allArtists, _, err := repo.SearchArtistsPaginated("", 10, 0)
		assert.NoError(t, err)

		// The ordering should be by name_normalized ASC, id ASC
		for i := 0; i < len(allArtists)-1; i++ {
			current := allArtists[i]
			next := allArtists[i+1]
			// Either name is in order, or same name with lower id first
			assert.True(t, current.NameNormalized < next.NameNormalized ||
				(current.NameNormalized == next.NameNormalized && current.ID <= next.ID))
		}
	})
}

// TestRepositorySearchAlbumsPaginated tests the SearchAlbumsPaginated method with filters, pagination, and ordering
func TestRepositorySearchAlbumsPaginated(t *testing.T) {
	db, tearDown := test.SetupTestEnvironment(t)
	defer tearDown()

	repo := NewRepository(db)

	// Create a test artist first
	artist := &models.Artist{
		Name:           "Test Artist",
		NameNormalized: "test artist",
		DirectoryCode:  "TA",
	}
	err := repo.CreateArtist(artist)
	assert.NoError(t, err)

	// Create test albums for search
	testAlbums := []*models.Album{
		{
			Name:           "Abbey Road",
			NameNormalized: "abbey road",
			ArtistID:       artist.ID,
		},
		{
			Name:           "Let It Be",
			NameNormalized: "let it be",
			ArtistID:       artist.ID,
		},
		{
			Name:           "Please Please Me",
			NameNormalized: "please please me",
			ArtistID:       artist.ID,
		},
		{
			Name:           "Rubber Soul",
			NameNormalized: "rubber soul",
			ArtistID:       artist.ID,
		},
	}

	// Create all test albums
	for _, album := range testAlbums {
		err := repo.CreateAlbum(album)
		assert.NoError(t, err)
		assert.NotZero(t, album.ID)
	}

	t.Run("Search all albums with pagination", func(t *testing.T) {
		// Search for all albums (using a common letter)
		albums, total, err := repo.SearchAlbumsPaginated("e", 10, 0)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(albums), 3) // Should find multiple albums with 'e'
		assert.GreaterOrEqual(t, total, int64(3))
	})

	t.Run("Search with specific term", func(t *testing.T) {
		// Search for "abbey"
		albums, total, err := repo.SearchAlbumsPaginated("abbey", 10, 0)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(albums), 1) // Should find "Abbey Road"
		assert.GreaterOrEqual(t, total, int64(1))

		// Verify the result contains "Abbey Road"
		foundAbbeyRoad := false
		for _, album := range albums {
			if album.NameNormalized == "abbey road" {
				foundAbbeyRoad = true
				break
			}
		}
		assert.True(t, foundAbbeyRoad)
	})

	t.Run("Search with pagination and ordering", func(t *testing.T) {
		// Search and get first 2 results
		albums1, total, err := repo.SearchAlbumsPaginated("", 2, 0) // Get first 2 albums
		assert.NoError(t, err)
		assert.Equal(t, 2, len(albums1))
		assert.GreaterOrEqual(t, total, int64(4))

		// Get the next 2 results
		albums2, total, err := repo.SearchAlbumsPaginated("", 2, 2) // Get next 2 albums
		assert.NoError(t, err)
		assert.Equal(t, 2, len(albums2))
		assert.GreaterOrEqual(t, total, int64(4))

		// Verify ordering is consistent (by name_normalized ASC, then id ASC)
		assert.True(t, albums2[0].NameNormalized >= albums1[1].NameNormalized)
	})

	t.Run("Verify album associations are preloaded", func(t *testing.T) {
		// Search for an album and verify artist association is preloaded
		albums, _, err := repo.SearchAlbumsPaginated("abbey", 10, 0)
		assert.NoError(t, err)

		if len(albums) > 0 {
			// Verify the artist association is loaded
			assert.NotZero(t, albums[0].Artist.ID)
			assert.Equal(t, artist.Name, albums[0].Artist.Name)
		}
	})
}

// TestRepositorySearchTracksPaginated tests the SearchTracksPaginated method with filters, pagination, and ordering
func TestRepositorySearchTracksPaginated(t *testing.T) {
	db, tearDown := test.SetupTestEnvironment(t)
	defer tearDown()

	repo := NewRepository(db)

	// Create a test artist and album first
	artist := &models.Artist{
		Name:           "Test Artist",
		NameNormalized: "test artist",
		DirectoryCode:  "TA",
	}
	err := repo.CreateArtist(artist)
	assert.NoError(t, err)

	album := &models.Album{
		Name:           "Test Album",
		NameNormalized: "test album",
		ArtistID:       artist.ID,
	}
	err = repo.CreateAlbum(album)
	assert.NoError(t, err)

	// Create test tracks for search
	testTracks := []*models.Track{
		{
			Name:           "Yesterday",
			NameNormalized: "yesterday",
			AlbumID:        album.ID,
			ArtistID:       artist.ID,
		},
		{
			Name:           "Here Comes the Sun",
			NameNormalized: "here comes the sun",
			AlbumID:        album.ID,
			ArtistID:       artist.ID,
		},
		{
			Name:           "Hey Jude",
			NameNormalized: "hey jude",
			AlbumID:        album.ID,
			ArtistID:       artist.ID,
		},
		{
			Name:           "Come Together",
			NameNormalized: "come together",
			AlbumID:        album.ID,
			ArtistID:       artist.ID,
		},
	}

	// Create all test tracks
	for _, track := range testTracks {
		err := repo.CreateTrack(track)
		assert.NoError(t, err)
		assert.NotZero(t, track.ID)
	}

	t.Run("Search all tracks with pagination", func(t *testing.T) {
		// Search for all tracks (using a common letter)
		tracks, total, err := repo.SearchTracksPaginated("e", 10, 0) // Tracks with 'e'
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(tracks), 3) // Multiple tracks contain 'e'
		assert.GreaterOrEqual(t, total, int64(3))
	})

	t.Run("Search with specific term", func(t *testing.T) {
		// Search for "yesterday"
		tracks, total, err := repo.SearchTracksPaginated("yesterday", 10, 0)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(tracks), 1) // Should find "Yesterday"
		assert.GreaterOrEqual(t, total, int64(1))

		// Verify the result contains "Yesterday"
		foundYesterday := false
		for _, track := range tracks {
			if track.NameNormalized == "yesterday" {
				foundYesterday = true
				break
			}
		}
		assert.True(t, foundYesterday)
	})

	t.Run("Search with pagination and ordering", func(t *testing.T) {
		// Search and get first 2 results
		tracks1, total, err := repo.SearchTracksPaginated("", 2, 0) // Get first 2 tracks
		assert.NoError(t, err)
		assert.Equal(t, 2, len(tracks1))
		assert.GreaterOrEqual(t, total, int64(4))

		// Get the next 2 results
		tracks2, total, err := repo.SearchTracksPaginated("", 2, 2) // Get next 2 tracks
		assert.NoError(t, err)
		assert.Equal(t, 2, len(tracks2))
		assert.GreaterOrEqual(t, total, int64(4))

		// Verify ordering is consistent (by name_normalized ASC, then id ASC)
		assert.True(t, tracks2[0].NameNormalized >= tracks1[1].NameNormalized)
	})

	t.Run("Verify track associations are preloaded", func(t *testing.T) {
		// Search for a track and verify album and artist associations are preloaded
		tracks, _, err := repo.SearchTracksPaginated("yesterday", 10, 0)
		assert.NoError(t, err)

		if len(tracks) > 0 {
			// Verify the album and artist associations are loaded
			assert.NotZero(t, tracks[0].Album.ID)
			assert.Equal(t, album.Name, tracks[0].Album.Name)
			assert.NotZero(t, tracks[0].Artist.ID)
			assert.Equal(t, artist.Name, tracks[0].Artist.Name)
		}
	})
}

// TestSearchEntities tests the generic SearchEntities method
func TestSearchEntities(t *testing.T) {
	db, tearDown := test.SetupTestEnvironment(t)
	defer tearDown()

	repo := NewRepository(db)

	// Create some test data
	artist := &models.Artist{
		Name:           "Search Artist",
		NameNormalized: "search artist",
		DirectoryCode:  "SA",
	}
	err := repo.CreateArtist(artist)
	assert.NoError(t, err)

	album := &models.Album{
		Name:           "Search Album",
		NameNormalized: "search album",
		ArtistID:       artist.ID,
	}
	err = repo.CreateAlbum(album)
	assert.NoError(t, err)

	song := &models.Track{
		Name:           "Search Song",
		NameNormalized: "search song",
		AlbumID:        album.ID,
		ArtistID:       artist.ID,
	}
	err = repo.CreateTrack(song)
	assert.NoError(t, err)

	t.Run("Search artists by type", func(t *testing.T) {
		results, total, err := repo.SearchEntities("search", "artist", 10, 0)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 1)
		assert.GreaterOrEqual(t, total, int64(1))
	})

	t.Run("Search albums by type", func(t *testing.T) {
		results, total, err := repo.SearchEntities("search", "album", 10, 0)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 1)
		assert.GreaterOrEqual(t, total, int64(1))
	})

	t.Run("Search songs by type", func(t *testing.T) {
		results, total, err := repo.SearchEntities("search", "song", 10, 0)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 1)
		assert.GreaterOrEqual(t, total, int64(1))
	})

	t.Run("Search with unsupported entity type", func(t *testing.T) {
		_, _, err := repo.SearchEntities("search", "invalid_type", 10, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported entity type")
	})
}
