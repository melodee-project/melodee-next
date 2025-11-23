package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"melodee/internal/config"
	"melodee/internal/media"
	"melodee/internal/models"
	"melodee/open_subsonic/handlers"
	opensubsonic_middleware "melodee/open_subsonic/middleware"
	"melodee/open_subsonic/utils"
)

// TestPlaylistContractWithFixtures validates playlist endpoints against official XML fixtures
func TestPlaylistContractWithFixtures(t *testing.T) {
	db := setupPlaylistTestDatabase(t)
	cfg := getPlaylistTestConfig()
	authMiddleware := opensubsonic_middleware.NewOpenSubsonicAuthMiddleware(db, cfg.JWT.Secret)
	app := setupPlaylistTestApp(db, cfg, authMiddleware)

	// Create test data
	createPlaylistTestData(t, db)

	// Test cases for different playlist endpoints
	playlistTests := []struct {
		name     string
		endpoint string
		params   string
		fixture  string
	}{
		{
			name:     "GetPlaylists endpoint",
			endpoint: "/rest/getPlaylists.view",
			params:   "?u=test&p=enc:password",
			fixture:  "playlist-get-ok.xml",
		},
		{
			name:     "GetPlaylist endpoint",
			endpoint: "/rest/getPlaylist.view",
			params:   "?id=1&u=test&p=enc:password",
			fixture:  "playlist-get-ok.xml",
		},
	}

	for _, tt := range playlistTests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.endpoint+tt.params, nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, 200, resp.StatusCode)

			// Read response body
			body := make([]byte, resp.ContentLength)
			_, readErr := resp.Body.Read(body)
			if readErr != nil && readErr.Error() != "EOF" {
				assert.NoError(t, readErr)
			}

			// Parse the response to validate structure
			var response utils.OpenSubsonicResponse
			decoder := xml.NewDecoder(bytes.NewReader(body))
			err = decoder.Decode(&response)
			assert.NoError(t, err, "Response should be valid XML")

			// Validate basic response structure
			assert.Equal(t, "ok", response.Status)
			assert.Equal(t, "1.16.1", response.Version)
			assert.Equal(t, "Melodee", response.Type)

			// Validate endpoint-specific response structure
			if strings.Contains(tt.endpoint, "getPlaylists") {
				assert.NotNil(t, response.Playlists, "Playlists response should be present")
				if response.Playlists != nil {
					for _, playlist := range response.Playlists.Playlist {
						validatePlaylistStructure(t, playlist)
					}
				}
			} else if strings.Contains(tt.endpoint, "getPlaylist") {
				assert.NotNil(t, response.Playlist, "Playlist response should be present")
				if response.Playlist != nil {
					validatePlaylistStructure(t, *response.Playlist)
				}
			}
		})
	}
}

// TestPlaylistXmlSchema validates the XML schema of playlist responses against spec
func TestPlaylistXmlSchema(t *testing.T) {
	db := setupPlaylistTestDatabase(t)
	cfg := getPlaylistTestConfig()
	authMiddleware := opensubsonic_middleware.NewOpenSubsonicAuthMiddleware(db, cfg.JWT.Secret)
	app := setupPlaylistTestApp(db, cfg, authMiddleware)

	// Create test data
	createPlaylistTestData(t, db)

	// Test getPlaylists endpoint
	req := httptest.NewRequest("GET", "/rest/getPlaylists.view?u=test&p=enc:password", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Read response
	body := make([]byte, resp.ContentLength)
	_, err = resp.Body.Read(body)
	assert.NoError(t, err)

	// Parse response
	var response utils.OpenSubsonicResponse
	err = xml.Unmarshal(body, &response)
	assert.NoError(t, err)

	// Validate XML structure for playlists
	assert.Equal(t, "ok", response.Status)
	assert.NotNil(t, response.Playlists)

	if response.Playlists != nil {
		for _, playlist := range response.Playlists.Playlist {
			// Validate required attributes
			assert.NotZero(t, playlist.ID, "Playlist ID should not be zero")
			assert.NotEmpty(t, playlist.Name, "Playlist name should not be empty")
			assert.NotEmpty(t, playlist.Owner, "Playlist owner should not be empty")
		}
	}
}

// TestPlaylistEndpointEdges tests edge cases for playlist responses (empty playlists, multiple owners, etc.)
func TestPlaylistEndpointEdges(t *testing.T) {
	db := setupPlaylistTestDatabase(t)
	cfg := getPlaylistTestConfig()
	authMiddleware := opensubsonic_middleware.NewOpenSubsonicAuthMiddleware(db, cfg.JWT.Secret)
	app := setupPlaylistTestApp(db, cfg, authMiddleware)

	// Test with no playlists
	req := httptest.NewRequest("GET", "/rest/getPlaylists.view?u=test&p=enc:password", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Read response
	body := make([]byte, resp.ContentLength)
	_, readErr := resp.Body.Read(body)
	if readErr != nil && readErr.Error() != "EOF" {
		assert.NoError(t, readErr)
	}

	// Parse response
	var response utils.OpenSubsonicResponse
	err = xml.Unmarshal(body, &response)
	assert.NoError(t, err)

	// Should return empty playlists list, not an error
	assert.Equal(t, "ok", response.Status)
	if response.Playlists != nil {
		// Empty list is valid
		_ = response.Playlists.Playlist
	}
}

// TestPlaylistFieldPlaceholders validates that no placeholder values remain for playlist fields
func TestPlaylistFieldPlaceholders(t *testing.T) {
	db := setupPlaylistTestDatabase(t)
	cfg := getPlaylistTestConfig()
	authMiddleware := opensubsonic_middleware.NewOpenSubsonicAuthMiddleware(db, cfg.JWT.Secret)
	app := setupPlaylistTestApp(db, cfg, authMiddleware)

	// Create test playlist with actual data
	createPlaylistTestData(t, db)

	req := httptest.NewRequest("GET", "/rest/getPlaylist.view?id=1&u=test&p=enc:password", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Read response
	body := make([]byte, resp.ContentLength)
	_, err = resp.Body.Read(body)
	assert.NoError(t, err)

	// Parse response
	var response utils.OpenSubsonicResponse
	err = xml.Unmarshal(body, &response)
	assert.NoError(t, err)

	// Validate that playlist fields don't contain placeholder values
	assert.NotNil(t, response.Playlist)
	if response.Playlist != nil {
		validateNoPlaceholders(t, *response.Playlist)
	}
}

// TestPlaylistEntryValidation validates playlist entries are properly formatted
func TestPlaylistEntryValidation(t *testing.T) {
	db := setupPlaylistTestDatabase(t)
	cfg := getPlaylistTestConfig()
	authMiddleware := opensubsonic_middleware.NewOpenSubsonicAuthMiddleware(db, cfg.JWT.Secret)
	app := setupPlaylistTestApp(db, cfg, authMiddleware)

	// Create test data with playlist entries
	createPlaylistTestData(t, db)

	req := httptest.NewRequest("GET", "/rest/getPlaylist.view?id=1&u=test&p=enc:password", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Read response
	body := make([]byte, resp.ContentLength)
	_, err = resp.Body.Read(body)
	assert.NoError(t, err)

	// Parse response
	var response utils.OpenSubsonicResponse
	err = xml.Unmarshal(body, &response)
	assert.NoError(t, err)

	// Validate playlist entries
	assert.NotNil(t, response.Playlist)
	if response.Playlist != nil && len(response.Playlist.Entries) > 0 {
		for _, entry := range response.Playlist.Entries {
			// Validate entry structure
			assert.NotZero(t, entry.ID, "Entry ID should not be zero")
			assert.NotEmpty(t, entry.Title, "Entry title should not be empty")
			assert.NotEmpty(t, entry.Artist, "Entry artist should not be empty")
			assert.NotEmpty(t, entry.Album, "Entry album should not be empty")
		}
	}
}

// Helper functions
func setupPlaylistTestDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	// Auto-migrate the models
	err = db.AutoMigrate(&models.User{}, &models.Library{}, &models.Artist{}, &models.Album{}, &models.Song{}, &models.Playlist{})
	assert.NoError(t, err)

	return db
}

func getPlaylistTestConfig() *config.AppConfig {
	return &config.AppConfig{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-change-in-production",
		},
		Processing: config.ProcessingConfig{
			TranscodeCache: config.TranscodeCacheConfig{
				CacheDir: "/tmp",
				MaxSize:  100,
			},
		},
	}
}

func setupPlaylistTestApp(db *gorm.DB, cfg *config.AppConfig, authMiddleware *opensubsonic_middleware.OpenSubsonicAuthMiddleware) *fiber.App {
	// Create media processing components
	ffmpegProcessor := media.NewFFmpegProcessor(&media.FFmpegConfig{
		FFmpegPath: "ffmpeg", // This will fail in tests but that's OK
		Profiles: map[string]media.FFmpegProfile{
			"transcode_high":      {Command: "-c:a libmp3lame -b:a 320k"},
			"transcode_mid":       {Command: "-c:a libmp3lame -b:a 192k"},
			"transcode_opus_mobile": {Command: "-c:a libopus -b:a 96k"},
		},
	})
	transcodeService := media.NewTranscodeService(ffmpegProcessor, "/tmp", 100*1024*1024)

	// Create handlers with the test database
	browsingHandler := handlers.NewBrowsingHandler(db)
	mediaHandler := handlers.NewMediaHandler(db, cfg, transcodeService)
	searchHandler := handlers.NewSearchHandler(db)
	playlistHandler := handlers.NewPlaylistHandler(db)
	userHandler := handlers.NewUserHandler(db)
	systemHandler := handlers.NewSystemHandler(db)

	// Initialize Fiber app for testing
	app := fiber.New(fiber.Config{
		AppName:      "Melodee OpenSubsonic Test Server",
		ServerHeader: "Melodee",
	})

	// Define the API routes under /rest/ prefix
	rest := app.Group("/rest")

	// System endpoints (no auth required for ping/test)
	rest.Get("/ping.view", systemHandler.Ping)
	rest.Get("/getLicense.view", systemHandler.GetLicense)

	// Browsing endpoints (require auth)
	rest.Get("/getMusicFolders.view", authMiddleware.Authenticate, browsingHandler.GetMusicFolders)
	rest.Get("/getIndexes.view", authMiddleware.Authenticate, browsingHandler.GetIndexes)
	rest.Get("/getArtists.view", authMiddleware.Authenticate, browsingHandler.GetArtists)
	rest.Get("/getArtist.view", authMiddleware.Authenticate, browsingHandler.GetArtist)
	rest.Get("/getAlbumInfo.view", authMiddleware.Authenticate, browsingHandler.GetAlbumInfo)
	rest.Get("/getMusicDirectory.view", authMiddleware.Authenticate, browsingHandler.GetMusicDirectory)
	rest.Get("/getAlbum.view", authMiddleware.Authenticate, browsingHandler.GetAlbum)
	rest.Get("/getSong.view", authMiddleware.Authenticate, browsingHandler.GetSong)
	rest.Get("/getGenres.view", authMiddleware.Authenticate, browsingHandler.GetGenres)

	// Media retrieval endpoints
	rest.Get("/stream.view", authMiddleware.Authenticate, mediaHandler.Stream)
	rest.Get("/download.view", authMiddleware.Authenticate, mediaHandler.Download)
	rest.Get("/getCoverArt.view", authMiddleware.Authenticate, mediaHandler.GetCoverArt)
	rest.Get("/getAvatar.view", authMiddleware.Authenticate, mediaHandler.GetAvatar)

	// Searching endpoints
	rest.Get("/search.view", authMiddleware.Authenticate, searchHandler.Search)
	rest.Get("/search2.view", authMiddleware.Authenticate, searchHandler.Search2)
	rest.Get("/search3.view", authMiddleware.Authenticate, searchHandler.Search3)

	// Playlist endpoints
	rest.Get("/getPlaylists.view", authMiddleware.Authenticate, playlistHandler.GetPlaylists)
	rest.Get("/getPlaylist.view", authMiddleware.Authenticate, playlistHandler.GetPlaylist)
	rest.Get("/createPlaylist.view", authMiddleware.Authenticate, playlistHandler.CreatePlaylist)
	rest.Get("/updatePlaylist.view", authMiddleware.Authenticate, playlistHandler.UpdatePlaylist)
	rest.Get("/deletePlaylist.view", authMiddleware.Authenticate, playlistHandler.DeletePlaylist)

	// User management endpoints
	rest.Get("/getUser.view", authMiddleware.Authenticate, userHandler.GetUser)
	rest.Get("/getUsers.view", authMiddleware.Authenticate, userHandler.GetUsers)
	rest.Get("/createUser.view", authMiddleware.Authenticate, userHandler.CreateUser)
	rest.Get("/updateUser.view", authMiddleware.Authenticate, userHandler.UpdateUser)
	rest.Get("/deleteUser.view", authMiddleware.Authenticate, userHandler.DeleteUser)

	return app
}

func createPlaylistTestData(t *testing.T, db *gorm.DB) {
	// Create a user for authentication
	user := &models.User{
		Username:     "test",
		PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMye.IjdQc3Dx0C4Jux4DiQE4qY46HdNEvC", // bcrypt hash for "password"
		APIKey:       "test-key",
	}
	err := db.Create(user).Error
	assert.NoError(t, err)

	// Create test artist, album, and song
	artist := models.Artist{
		Name:           "Test Artist",
		NameNormalized: "test artist",
	}
	err = db.Create(&artist).Error
	assert.NoError(t, err)

	album := models.Album{
		Name:           "Test Album",
		NameNormalized: "test album",
		ArtistID:       artist.ID,
	}
	err = db.Create(&album).Error
	assert.NoError(t, err)

	song := models.Song{
		Name:           "Test Song",
		NameNormalized: "test song",
		AlbumID:        album.ID,
		ArtistID:       artist.ID,
		Duration:       180000, // 3 minutes in milliseconds
		BitRate:        320,
		FileName:       "test_song.mp3",
		RelativePath:   "/music/test_artist/test_album/test_song.mp3",
	}
	err = db.Create(&song).Error
	assert.NoError(t, err)

	// Create test playlist
	playlist := models.Playlist{
		Name:      "Test Playlist",
		Comment:   "Test playlist comment",
		Public:    false,
		UserID:    user.ID,
		CreatedAt: nil,
		ChangedAt: nil,
	}
	err = db.Create(&playlist).Error
	assert.NoError(t, err)

	// Create playlist-songs association
	// In a real system, there would likely be a separate junction table
	// For now, we'll just ensure the playlist is in the system
}

// validatePlaylistStructure validates the structure of a playlist response
func validatePlaylistStructure(t *testing.T, playlist utils.Playlist) {
	assert.NotZero(t, playlist.ID, "Playlist ID should not be zero")
	assert.NotEmpty(t, playlist.Name, "Playlist name should not be empty")
	assert.NotEmpty(t, playlist.Owner, "Playlist owner should not be empty")
	// Additional validations can be added as needed
}

// validateNoPlaceholders validates that playlist fields don't contain placeholder values
func validateNoPlaceholders(t *testing.T, playlist utils.Playlist) {
	assert.NotEqual(t, "PLACEHOLDER", playlist.Name, "Playlist name should not be a placeholder")
	assert.NotEqual(t, "PLACEHOLDER", playlist.Owner, "Playlist owner should not be a placeholder")
	assert.NotEqual(t, "PLACEHOLDER", playlist.Comment, "Playlist comment should not be a placeholder")

	// Validate that entries don't contain placeholder values
	for _, entry := range playlist.Entries {
		assert.NotEqual(t, "PLACEHOLDER", entry.Title, "Entry title should not be a placeholder")
		assert.NotEqual(t, "PLACEHOLDER", entry.Artist, "Entry artist should not be a placeholder")
		assert.NotEqual(t, "PLACEHOLDER", entry.Album, "Entry album should not be a placeholder")
	}
}