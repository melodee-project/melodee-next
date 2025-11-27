package services

import (
	"testing"
	"time"

	"melodee/internal/models"
	"melodee/internal/test"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestServiceIntegration tests the integration between different services
func TestServiceIntegration(t *testing.T) {
	// Create a test database instance
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Initialize all services
	repo := NewRepository(db)
	authService := NewAuthService(db, "test-jwt-secret-key-change-in-production")

	// Test service integration by creating a user and authenticating
	username := "integration-test-user"
	password := "ValidPass123!" // Meets our password requirements

	// First, create a user in the repository
	user := &models.User{
		Username: username,
		Email:    "integration@test.com",
		APIKey:   uuid.New(),
	}

	// Hash the password before storing
	hashedPassword, err := authService.HashPassword(password)
	assert.NoError(t, err)
	user.PasswordHash = hashedPassword

	err = repo.CreateUser(user)
	assert.NoError(t, err)

	// Now try to authenticate using the auth service
	authToken, authenticatedUser, err := authService.Login(username, password)
	assert.NoError(t, err)
	assert.NotNil(t, authToken)
	assert.Equal(t, username, authenticatedUser.Username)
	assert.Equal(t, user.ID, authenticatedUser.ID)

	// Verify token contents
	token, err := jwt.ParseWithClaims(authToken.AccessToken, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte("test-jwt-secret-key-change-in-production"), nil
	})
	assert.NoError(t, err)
	assert.True(t, token.Valid)

	claims, ok := token.Claims.(*Claims)
	assert.True(t, ok)
	assert.Equal(t, authenticatedUser.ID, claims.UserID)
	assert.Equal(t, username, claims.Username)
}

// TestMediaServiceIntegration tests media service integration
func TestMediaServiceIntegration(t *testing.T) {
	// Create a test database instance
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Initialize services
	repo := NewRepository(db)
	authService := NewAuthService(db, "test-jwt-secret-key-change-in-production")

	// Create test user
	user := &models.User{
		Username: "media-test-user",
		Email:    "media@test.com",
		APIKey:   uuid.New(),
	}

	password := "ValidPass123!"
	hashedPassword, err := authService.HashPassword(password)
	assert.NoError(t, err)
	user.PasswordHash = hashedPassword

	err = repo.CreateUser(user)
	assert.NoError(t, err)

	// Verify user can be retrieved from the repository
	retrievedUser, err := repo.GetUserByID(user.ID)
	assert.NoError(t, err)
	assert.Equal(t, user.Username, retrievedUser.Username)
	assert.Equal(t, user.Email, retrievedUser.Email)
	assert.Equal(t, user.APIKey, retrievedUser.APIKey)
}

// TestLibraryServiceIntegration tests library service integration
func TestLibraryServiceIntegration(t *testing.T) {
	// Create a test database instance
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Initialize services
	repo := NewRepository(db)

	// Create test library
	library := &models.Library{
		Name: "Test Library",
		Path: "/test/library/path",
		Type: "production",
	}

	err := repo.CreateLibrary(library)
	assert.NoError(t, err)
	assert.NotZero(t, library.ID)

	// Retrieve and verify the library
	retrievedLibrary, err := repo.GetLibraryByID(int32(library.ID))
	assert.NoError(t, err)
	assert.Equal(t, library.Name, retrievedLibrary.Name)
	assert.Equal(t, library.Path, retrievedLibrary.Path)
	assert.Equal(t, library.Type, retrievedLibrary.Type)

	// Test updating the library
	updatedName := "Updated Test Library"
	retrievedLibrary.Name = updatedName
	err = repo.UpdateLibrary(retrievedLibrary)
	assert.NoError(t, err)

	// Verify the update
	updatedLibrary, err := repo.GetLibraryByID(int32(library.ID))
	assert.NoError(t, err)
	assert.Equal(t, updatedName, updatedLibrary.Name)
}

// TestArtistAlbumSongIntegration tests the full artist-album-song hierarchy integration
func TestArtistAlbumSongIntegration(t *testing.T) {
	// Create a test database instance
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Initialize repository
	repo := NewRepository(db)

	// Create a test artist
	artist := &models.Artist{
		Name:           "Integration Test Artist",
		NameNormalized: "integration test artist",
		APIKey:         uuid.New(),
	}

	err := repo.CreateArtist(artist)
	assert.NoError(t, err)
	assert.NotZero(t, artist.ID)
	assert.NotEqual(t, uuid.Nil, artist.APIKey)

	// Create an album for the artist
	album := &models.Album{
		Name:           "Integration Test Album",
		NameNormalized: "integration test album",
		ArtistID:       artist.ID,
		APIKey:         uuid.New(),
		Directory:      "test/artist/integration-test-album",
	}

	err = repo.CreateAlbum(album)
	assert.NoError(t, err)
	assert.NotZero(t, album.ID)
	assert.NotEqual(t, uuid.Nil, album.APIKey)

	// Create a song for the album
	song := &models.Track{
		Name:           "Integration Test Song",
		NameNormalized: "integration test song",
		AlbumID:        album.ID,
		ArtistID:       artist.ID,
		APIKey:         uuid.New(),
		Duration:       240000, // 4 minutes in milliseconds
		BitRate:        320,    // 320 kbps
		BitDepth:       16,     // 16-bit
		SampleRate:     44100,  // 44.1 kHz
		RelativePath:   "test/artist/integration-test-album/test-song.mp3",
		CRCHash:        "abcd1234",
	}

	err = repo.CreateTrack(song)
	assert.NoError(t, err)
	assert.NotZero(t, song.ID)
	assert.NotEqual(t, uuid.Nil, song.APIKey)

	// Verify the relationships
	albumWithSongs, err := repo.GetAlbumByID(album.ID)
	assert.NoError(t, err)
	assert.Equal(t, album.Name, albumWithSongs.Name)
	assert.Len(t, albumWithSongs.Songs, 1) // Should include the song we created
	assert.Equal(t, song.Name, albumWithSongs.Songs[0].Name)

	artistWithAlbums, err := repo.GetArtistByID(artist.ID)
	assert.NoError(t, err)
	assert.Equal(t, artist.Name, artistWithAlbums.Name)
	assert.Len(t, artistWithAlbums.Albums, 1) // Should include the album we created
	assert.Equal(t, album.Name, artistWithAlbums.Albums[0].Name)
}

// TestPlaylistIntegration tests playlist service integration with users and songs
func TestPlaylistIntegration(t *testing.T) {
	// Create a test database instance
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Initialize services
	repo := NewRepository(db)
	authService := NewAuthService(db, "test-jwt-secret-key-change-in-production")

	// Create a test user
	user := &models.User{
		Username: "playlist-test-user",
		Email:    "playlist@test.com",
		APIKey:   uuid.New(),
	}

	password := "ValidPass123!"
	hashedPassword, err := authService.HashPassword(password)
	assert.NoError(t, err)
	user.PasswordHash = hashedPassword

	err = repo.CreateUser(user)
	assert.NoError(t, err)

	// Create test artist, album, and song
	artist := &models.Artist{
		Name:           "Playlist Test Artist",
		NameNormalized: "playlist test artist",
		APIKey:         uuid.New(),
	}

	err = repo.CreateArtist(artist)
	assert.NoError(t, err)

	album := &models.Album{
		Name:           "Playlist Test Album",
		NameNormalized: "playlist test album",
		ArtistID:       artist.ID,
		APIKey:         uuid.New(),
		Directory:      "test/artist/playlist-test-album",
	}

	err = repo.CreateAlbum(album)
	assert.NoError(t, err)

	song := &models.Track{
		Name:           "Playlist Test Song",
		NameNormalized: "playlist test song",
		AlbumID:        album.ID,
		ArtistID:       artist.ID,
		APIKey:         uuid.New(),
		Duration:       180000, // 3 minutes in milliseconds
		BitRate:        256,    // 256 kbps
		RelativePath:   "test/artist/playlist-test-album/test-song.mp3",
		CRCHash:        "efgh5678",
	}

	err = repo.CreateTrack(song)
	assert.NoError(t, err)

	// Create a playlist
	playlist := &models.Playlist{
		Name:       "Test Playlist",
		UserID:     user.ID,
		APIKey:     uuid.New(),
		TrackCount: 0,
		Duration:   0,
		CreatedAt:  time.Now(),
		ChangedAt:  time.Now(),
	}

	err = repo.CreatePlaylist(playlist)
	assert.NoError(t, err)
	assert.NotZero(t, playlist.ID)

	// Add the song to the playlist
	playlistSong := &models.PlaylistTrack{
		PlaylistID: playlist.ID,
		TrackID:    song.ID,
		Position:   1,
		CreatedAt:  time.Now(),
	}

	err = repo.AddSongToPlaylist(playlistSong)
	assert.NoError(t, err)

	// Retrieve the playlist with its songs
	playlistWithSongs, err := repo.GetPlaylistWithSongs(playlist.ID)
	assert.NoError(t, err)
	assert.Equal(t, playlist.Name, playlistWithSongs.Name)
	assert.Equal(t, user.ID, playlistWithSongs.UserID)
	assert.Len(t, playlistWithSongs.Songs, 1)
	assert.Equal(t, song.Name, playlistWithSongs.Songs[0].Name)
}
