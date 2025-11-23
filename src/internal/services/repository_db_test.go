package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"melodee/internal/models"
	"melodee/internal/test"
)

// TestDBBackedRepository tests the repository with an actual database connection
func TestDBBackedRepository(t *testing.T) {
	db, tearDown := test.SetupTestEnvironment(t)
	defer tearDown()

	repo := NewRepository(db)

	t.Run("User operations", func(t *testing.T) {
		// Test Create User
		user := &models.User{
			Username:     "testuser",
			Email:        "test@example.com",
			PasswordHash: "hashed_password",
			APIKey:       uuid.New(),
		}

		err := repo.CreateUser(user)
		assert.NoError(t, err)
		assert.NotZero(t, user.ID)

		// Test Get User By ID
		retrievedUser, err := repo.GetUserByID(user.ID)
		assert.NoError(t, err)
		assert.Equal(t, user.Username, retrievedUser.Username)
		assert.Equal(t, user.Email, retrievedUser.Email)

		// Test Get User By Username
		retrievedUser, err = repo.GetUserByUsername("testuser")
		assert.NoError(t, err)
		assert.Equal(t, user.ID, retrievedUser.ID)
		assert.Equal(t, user.Username, retrievedUser.Username)

		// Test Update User
		updatedEmail := "updated@example.com"
		retrievedUser.Email = updatedEmail
		err = repo.UpdateUser(retrievedUser)
		assert.NoError(t, err)

		// Verify update
		updatedUser, err := repo.GetUserByID(user.ID)
		assert.NoError(t, err)
		assert.Equal(t, updatedEmail, updatedUser.Email)

		// Test Delete User
		err = repo.DeleteUser(user.ID)
		assert.NoError(t, err)

		// Verify deletion
		_, err = repo.GetUserByID(user.ID)
		assert.Error(t, err)
	})

	t.Run("Playlist operations", func(t *testing.T) {
		// Create a user first to associate with the playlist
		user := &models.User{
			Username:     "playlist_user",
			Email:        "playlist@example.com",
			PasswordHash: "hashed_password",
			APIKey:       uuid.New(),
		}
		err := repo.CreateUser(user)
		assert.NoError(t, err)

		// Test Create Playlist
		playlist := &models.Playlist{
			Name:      "Test Playlist",
			Public:    true,
			UserID:    user.ID,
			CreatedAt: nil, // Will be set by GORM
			ChangedAt: nil, // Will be set by GORM
		}

		err = repo.CreatePlaylist(playlist)
		assert.NoError(t, err)
		assert.NotZero(t, playlist.ID)

		// Test Get Playlist By ID
		retrievedPlaylist, err := repo.GetPlaylistByID(playlist.ID)
		assert.NoError(t, err)
		assert.Equal(t, playlist.Name, retrievedPlaylist.Name)
		assert.Equal(t, playlist.Public, retrievedPlaylist.Public)
		assert.Equal(t, playlist.UserID, retrievedPlaylist.UserID)

		// Test Update Playlist
		updatedName := "Updated Playlist"
		retrievedPlaylist.Name = updatedName
		err = repo.UpdatePlaylist(retrievedPlaylist)
		assert.NoError(t, err)

		// Verify update
		updatedPlaylist, err := repo.GetPlaylistByID(playlist.ID)
		assert.NoError(t, err)
		assert.Equal(t, updatedName, updatedPlaylist.Name)

		// Test Get Playlists With User Pagination
		playlists, total, err := repo.GetPlaylistsWithUser(10, 0)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(playlists), 1)
		assert.GreaterOrEqual(t, total, int64(1))

		// Test Delete Playlist
		err = repo.DeletePlaylist(playlist.ID)
		assert.NoError(t, err)

		// Verify deletion
		_, err = repo.GetPlaylistByID(playlist.ID)
		assert.Error(t, err)
	})

	t.Run("Artist operations", func(t *testing.T) {
		// Test Create Artist
		artist := &models.Artist{
			Name:           "Test Artist",
			NameNormalized: "Test Artist",
			DirectoryCode:  "TA",
		}

		err := repo.CreateArtist(artist)
		assert.NoError(t, err)
		assert.NotZero(t, artist.ID)

		// Test Get Artist By ID
		retrievedArtist, err := repo.GetArtistByID(artist.ID)
		assert.NoError(t, err)
		assert.Equal(t, artist.Name, retrievedArtist.Name)
		assert.Equal(t, artist.DirectoryCode, retrievedArtist.DirectoryCode)
	})

	t.Run("Album operations", func(t *testing.T) {
		// First create an artist to associate with the album
		artist := &models.Artist{
			Name:           "Album Artist",
			NameNormalized: "Album Artist",
			DirectoryCode:  "AA",
		}
		err := repo.CreateArtist(artist)
		assert.NoError(t, err)

		// Test Create Album
		album := &models.Album{
			Name:           "Test Album",
			NameNormalized: "Test Album",
			ArtistID:       artist.ID,
		}

		err = repo.CreateAlbum(album)
		assert.NoError(t, err)
		assert.NotZero(t, album.ID)

		// Test Get Album By ID
		retrievedAlbum, err := repo.GetAlbumByID(album.ID)
		assert.NoError(t, err)
		assert.Equal(t, album.Name, retrievedAlbum.Name)
		assert.Equal(t, album.ArtistID, retrievedAlbum.ArtistID)
		assert.Equal(t, artist.Name, retrievedAlbum.Artist.Name) // Check preloaded association
	})

	t.Run("Song operations", func(t *testing.T) {
		// First create an artist and album to associate with the song
		artist := &models.Artist{
			Name:           "Song Artist",
			NameNormalized: "Song Artist",
			DirectoryCode:  "SA",
		}
		err := repo.CreateArtist(artist)
		assert.NoError(t, err)

		album := &models.Album{
			Name:           "Song Album",
			NameNormalized: "Song Album",
			ArtistID:       artist.ID,
		}
		err = repo.CreateAlbum(album)
		assert.NoError(t, err)

		// Test Create Song
		song := &models.Song{
			Name:           "Test Song",
			NameNormalized: "Test Song",
			AlbumID:        album.ID,
			ArtistID:       artist.ID,
		}

		err = repo.CreateSong(song)
		assert.NoError(t, err)
		assert.NotZero(t, song.ID)

		// Test Get Song By ID
		retrievedSong, err := repo.GetSongByID(song.ID)
		assert.NoError(t, err)
		assert.Equal(t, song.Name, retrievedSong.Name)
		assert.Equal(t, song.AlbumID, retrievedSong.AlbumID)
		assert.Equal(t, album.Name, retrievedSong.Album.Name) // Check preloaded association
		assert.Equal(t, artist.Name, retrievedSong.Artist.Name) // Check preloaded association
	})

	t.Run("Search operations", func(t *testing.T) {
		// First create some test data to search
		artist1 := &models.Artist{
			Name:           "Zebra Artist",
			NameNormalized: "Zebra Artist",
			DirectoryCode:  "ZA",
		}
		err := repo.CreateArtist(artist1)
		assert.NoError(t, err)

		artist2 := &models.Artist{
			Name:           "Alpha Artist",
			NameNormalized: "Alpha Artist",
			DirectoryCode:  "AA",
		}
		err = repo.CreateArtist(artist2)
		assert.NoError(t, err)

		album1 := &models.Album{
			Name:           "Zebra Album",
			NameNormalized: "Zebra Album",
			ArtistID:       artist1.ID,
		}
		err = repo.CreateAlbum(album1)
		assert.NoError(t, err)

		// Test artist search
		artists, total, err := repo.SearchArtistsPaginated("artist", 10, 0)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(artists), 2)
		assert.GreaterOrEqual(t, total, int64(2))

		// Test album search
		albums, total, err := repo.SearchAlbumsPaginated("album", 10, 0)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(albums), 1)
		assert.GreaterOrEqual(t, total, int64(1))

		// Test search with pagination - check ordering (should be alphabetical by name_normalized)
		artists, total, err = repo.SearchArtistsPaginated("artist", 10, 0)
		assert.NoError(t, err)
		if len(artists) >= 2 {
			// Should be sorted alphabetically by name_normalized (Alpha Artist should come before Zebra Artist)
			if len(artists) >= 2 {
				assert.True(t, artists[0].NameNormalized <= artists[1].NameNormalized, 
					"Artists should be sorted alphabetically by name_normalized")
			}
		}

		// Test search with limit and offset
		limitedArtists, total, err := repo.SearchArtistsPaginated("artist", 1, 0)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(limitedArtists))
		assert.GreaterOrEqual(t, total, int64(2))
	})
}

// TestDBBackedRepositoryFilters tests the repository with filters, pagination, and ordering
func TestDBBackedRepositoryFilters(t *testing.T) {
	db, tearDown := test.SetupTestEnvironment(t)
	defer tearDown()

	repo := NewRepository(db)

	// Create multiple artists for testing filters and pagination
	artists := make([]*models.Artist, 5)
	for i := 0; i < 5; i++ {
		artist := &models.Artist{
			Name:           "Artist " + string(rune('A'+i)),
			NameNormalized: "Artist " + string(rune('A'+i)),
			DirectoryCode:  "A" + string(rune('A'+i)),
		}
		err := repo.CreateArtist(artist)
		assert.NoError(t, err)
		artists[i] = artist
	}

	// Test pagination - get first 2 artists
	artists1, total, err := repo.SearchArtistsPaginated("Artist", 2, 0)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(artists1))
	assert.GreaterOrEqual(t, total, int64(5))

	// Test pagination - get next 2 artists (offset 2)
	artists2, total, err := repo.SearchArtistsPaginated("Artist", 2, 2)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(artists2))
	assert.GreaterOrEqual(t, total, int64(5))

	// Ensure no overlap between paginated results
	assert.NotEqual(t, artists1[0].ID, artists2[0].ID)
	assert.NotEqual(t, artists1[1].ID, artists2[1].ID)

	// Test search with non-existent term (should return empty)
	emptyResults, total, err := repo.SearchArtistsPaginated("NonExistent", 10, 0)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(emptyResults))
	assert.Equal(t, int64(0), total)
}

// TestRepositoryErrorHandling tests error cases in the repository
func TestRepositoryErrorHandling(t *testing.T) {
	db, tearDown := test.SetupTestEnvironment(t)
	defer tearDown()

	repo := NewRepository(db)

	t.Run("Get non-existent user", func(t *testing.T) {
		_, err := repo.GetUserByID(999999) // Non-existent ID
		assert.Error(t, err)
	})

	t.Run("Get non-existent user by username", func(t *testing.T) {
		_, err := repo.GetUserByUsername("nonexistent_user")
		assert.Error(t, err)
	})

	t.Run("Get non-existent playlist", func(t *testing.T) {
		_, err := repo.GetPlaylistByID(999999) // Non-existent ID
		assert.Error(t, err)
	})

	t.Run("Get non-existent artist", func(t *testing.T) {
		_, err := repo.GetArtistByID(999999) // Non-existent ID
		assert.Error(t, err)
	})

	t.Run("Get non-existent album", func(t *testing.T) {
		_, err := repo.GetAlbumByID(999999) // Non-existent ID
		assert.Error(t, err)
	})

	t.Run("Get non-existent song", func(t *testing.T) {
		_, err := repo.GetSongByID(999999) // Non-existent ID
		assert.Error(t, err)
	})
}