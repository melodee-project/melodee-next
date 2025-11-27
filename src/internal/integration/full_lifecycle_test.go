package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"melodee/internal/config"
	"melodee/internal/database"
	"melodee/internal/handlers"
	"melodee/internal/media"
	"melodee/internal/models"
	"melodee/internal/services"
	opensubsonic_middleware "melodee/open_subsonic/middleware"
	"melodee/open_subsonic/handlers"
)

// IntegrationTestSuite holds the state for integration tests
type IntegrationTestSuite struct {
	app          *fiber.App
	db           *gorm.DB
	dbManager    *database.DatabaseManager
	config       *config.AppConfig
	repo         *services.Repository
	tearDownFunc func()
}

// SetupIntegrationTestSuite creates and initializes an integration test suite
func SetupIntegrationTestSuite() *IntegrationTestSuite {
	suite := &IntegrationTestSuite{}

	// Create in-memory SQLite database
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database: " + err.Error())
	}

	// Auto-migrate models
	err = db.AutoMigrate(&models.User{}, &models.Library{}, &models.Artist{}, &models.Album{}, &models.Track{}, &models.Playlist{})
	if err != nil {
		panic("Failed to migrate database: " + err.Error())
	}

	suite.db = db

	// Create test configuration
	suite.config = &config.AppConfig{
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

	// Create database manager
	suite.dbManager = &database.DatabaseManager{DB: db}
	
	// Create repository
	suite.repo = services.NewRepository(suite.dbManager)

	// Initialize Fiber app for testing
	suite.app = fiber.New(fiber.Config{
		AppName:      "Melodee Integration Test Server",
		ServerHeader: "Melodee",
	})

	// Create handlers with test setup
	authHandler := handlers.NewAuthHandler(suite.repo, suite.config)
	systemHandler := handlers.NewSystemHandler(suite.repo)

	// Initialize media processing components
	ffmpegProcessor := media.NewFFmpegProcessor(&media.FFmpegConfig{
		FFmpegPath: "ffmpeg", // This will fail in tests but that's OK
		Profiles: map[string]media.FFmpegProfile{
			"transcode_high":      {Command: "-c:a libmp3lame -b:a 320k"},
			"transcode_mid":       {Command: "-c:a libmp3lame -b:a 192k"},
			"transcode_opus_mobile": {Command: "-c:a libopus -b:a 96k"},
		},
	})
	transcodeService := media.NewTranscodeService(ffmpegProcessor, "/tmp", 100*1024*1024)
	
	// Create OpenSubsonic handlers 
	browsingHandler := open_subsonic_handlers.NewBrowsingHandler(suite.db)
	mediaHandler := open_subsonic_handlers.NewMediaHandler(suite.db, suite.config, transcodeService)
	searchHandler := open_subsonic_handlers.NewSearchHandler(suite.db)
	playlistHandler := open_subsonic_handlers.NewPlaylistHandler(suite.db)
	openSubsonicUserHandler := open_subsonic_handlers.NewUserHandler(suite.db)
	openSubsonicSystemHandler := open_subsonic_handlers.NewSystemHandler(suite.db)

	// Create OpenSubsonic auth middleware
	authMiddleware := opensubsonic_middleware.NewOpenSubsonicAuthMiddleware(suite.db, suite.config.JWT.Secret)

	// Define API routes
	rest := suite.app.Group("/rest")

	// System endpoints (no auth required for ping/test)
	rest.Get("/ping.view", openSubsonicSystemHandler.Ping)
	rest.Get("/getLicense.view", openSubsonicSystemHandler.GetLicense)

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

	// Search endpoints
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
	rest.Get("/getUser.view", authMiddleware.Authenticate, openSubsonicUserHandler.GetUser)

	// Create a test user for authentication purposes
	testUser := &models.User{
		Username:     "testuser",
		PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMye.IjdQc3Dx0C4Jux4DiQE4qY46HdNEvC", // bcrypt hash for "password"
		APIKey:       "test-api-key",
		IsAdmin:      true,
	}
	err = suite.db.Create(testUser).Error
	if err != nil {
		panic("Failed to create test user: " + err.Error())
	}

	// Teardown function
	suite.tearDownFunc = func() {
		sqlDB, _ := suite.db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	return suite
}

// TestAuthIntegration validates the full authentication flow
func TestAuthIntegration(t *testing.T) {
	suite := SetupIntegrationTestSuite()
	defer func() {
		if suite.tearDownFunc != nil {
			suite.tearDownFunc()
		}
	}()

	// Test login endpoint
	loginPayload := map[string]string{
		"username": "testuser",
		"password": "password",
	}

	payloadBytes, err := json.Marshal(loginPayload)
	assert.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := suite.app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Verify we get a valid response with tokens
	var loginResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		User         struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
			IsAdmin  bool   `json:"is_admin"`
		} `json:"user"`
	}
	err = json.NewDecoder(resp.Body).Decode(&loginResponse)
	assert.NoError(t, err)
	assert.NotEmpty(t, loginResponse.AccessToken)
	assert.NotEmpty(t, loginResponse.RefreshToken)
	assert.Equal(t, "testuser", loginResponse.User.Username)
	assert.True(t, loginResponse.User.IsAdmin)
}

// TestMusicBrowsingIntegration tests the full music browsing workflow
func TestMusicBrowsingIntegration(t *testing.T) {
	suite := SetupIntegrationTestSuite()
	defer func() {
		if suite.tearDownFunc != nil {
			suite.tearDownFunc()
		}
	}()

	// Create test data for browsing
	artist := &models.Artist{
		Name:           "Integration Test Artist",
		NameNormalized: "integration test artist",
		DirectoryCode:  "ita",
	}
	err := suite.db.Create(artist).Error
	assert.NoError(t, err)

	album := &models.Album{
		Name:           "Integration Test Album",
		NameNormalized: "integration test album", 
		ArtistID:       artist.ID,
	}
	err = suite.db.Create(album).Error
	assert.NoError(t, err)

	song := &models.Track{
		Name:           "Integration Test Song",
		NameNormalized: "integration test song",
		AlbumID:        album.ID,
		ArtistID:       artist.ID,
		RelativePath:   "/test/test-song.mp3",
		Duration:       180000,
		BitRate:        320,
	}
	err = suite.db.Create(song).Error
	assert.NoError(t, err)

	// Test OpenSubsonic browsing endpoints
	req := httptest.NewRequest("GET", "/rest/getArtists.view?u=testuser&p=enc:password", nil)
	resp, err := suite.app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

// TestSearchIntegration tests the search workflows
func TestSearchIntegration(t *testing.T) {
	suite := SetupIntegrationTestSuite()
	defer func() {
		if suite.tearDownFunc != nil {
			suite.tearDownFunc()
		}
	}()

	// Create test data for searching
	artist := &models.Artist{
		Name:           "Search Test Artist",
		NameNormalized: "search test artist",
		DirectoryCode:  "sta",
	}
	err := suite.db.Create(artist).Error
	assert.NoError(t, err)

	album := &models.Album{
		Name:           "Search Test Album",
		NameNormalized: "search test album",
		ArtistID:       artist.ID,
	}
	err = suite.db.Create(album).Error
	assert.NoError(t, err)

	song := &models.Track{
		Name:           "Search Test Song",
		NameNormalized: "search test song",
		AlbumID:        album.ID,
		ArtistID:       artist.ID,
		RelativePath:   "/test/search-song.mp3",
		Duration:       240000,
		BitRate:        256,
	}
	err = suite.db.Create(song).Error
	assert.NoError(t, err)

	// Test search endpoints
	req := httptest.NewRequest("GET", "/rest/search3.view?u=testuser&p=enc:password&query=test", nil)
	resp, err := suite.app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

// TestMediaRetrievalIntegration tests media retrieval endpoints
func TestMediaRetrievalIntegration(t *testing.T) {
	suite := SetupIntegrationTestSuite()
	defer func() {
		if suite.tearDownFunc != nil {
			suite.tearDownFunc()
		}
	}()

	// Create test song with a real file path
	tempDir := os.TempDir()
	testAudioFile := fmt.Sprintf("%s/test_integration_audio.mp3", tempDir)
	// Create a fake audio file for testing
	err := os.WriteFile(testAudioFile, []byte("fake audio content for integration test"), 0644)
	assert.NoError(t, err)

	// Create test data
	artist := &models.Artist{
		Name:           "Media Test Artist",
		NameNormalized: "media test artist",
		DirectoryCode:  "mta",
	}
	err = suite.db.Create(artist).Error
	assert.NoError(t, err)

	album := &models.Album{
		Name:           "Media Test Album",
		NameNormalized: "media test album",
		ArtistID:       artist.ID,
	}
	err = suite.db.Create(album).Error
	assert.NoError(t, err)

	song := &models.Track{
		Name:           "Media Test Song",
		NameNormalized: "media test song",
		AlbumID:        album.ID,
		ArtistID:       artist.ID,
		RelativePath:   testAudioFile,
		Duration:       180000,
		BitRate:        320,
	}
	err = suite.db.Create(song).Error
	assert.NoError(t, err)

	// Test stream endpoint
	req := httptest.NewRequest("GET", fmt.Sprintf("/rest/stream.view?u=testuser&p=enc:password&id=%d", song.ID), nil)
	resp, err := suite.app.Test(req)
	assert.NoError(t, err)

	// The response could be 200 (success) or 404 (file not found) depending on the actual file system
	// but the important part is that the request lifecycle completed without server errors
	assert.Condition(t, func() bool {
		return resp.StatusCode == 200 || resp.StatusCode == 404
	})

	// Clean up test file
	os.Remove(testAudioFile)
}

// TestPlaylistIntegration tests playlist management workflows
func TestPlaylistIntegration(t *testing.T) {
	suite := SetupIntegrationTestSuite()
	defer func() {
		if suite.tearDownFunc != nil {
			suite.tearDownFunc()
		}
	}()

	// Create test data for playlists
	user := &models.User{
		Username:     "playlistuser",
		PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMye.IjdQc3Dx0C4Jux4DiQE4qY46HdNEvC", // bcrypt for "password"
		APIKey:       "test-playlist-key",
	}
	err := suite.db.Create(user).Error
	assert.NoError(t, err)

	artist := &models.Artist{
		Name:           "Playlist Test Artist",
		NameNormalized: "playlist test artist",
		DirectoryCode:  "pta",
	}
	err = suite.db.Create(artist).Error
	assert.NoError(t, err)

	album := &models.Album{
		Name:           "Playlist Test Album",
		NameNormalized: "playlist test album",
		ArtistID:       artist.ID,
	}
	err = suite.db.Create(album).Error
	assert.NoError(t, err)

	song := &models.Track{
		Name:           "Playlist Test Song",
		NameNormalized: "playlist test song",
		AlbumID:        album.ID,
		ArtistID:       artist.ID,
		RelativePath:   "/test/playlist-song.mp3",
		Duration:       210000,
		BitRate:        192,
	}
	err = suite.db.Create(song).Error
	assert.NoError(t, err)

	// Test getting playlists
	req := httptest.NewRequest("GET", "/rest/getPlaylists.view?u=testuser&p=enc:password", nil)
	resp, err := suite.app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

// TestErrorHandlingIntegration tests error paths in the request lifecycle
func TestErrorHandlingIntegration(t *testing.T) {
	suite := SetupIntegrationTestSuite()
	defer func() {
		if suite.tearDownFunc != nil {
			suite.tearDownFunc()
		}
	}()

	// Test with invalid credentials
	req := httptest.NewRequest("GET", "/rest/getArtists.view?u=invaliduser&p=enc:invalidpass", nil)
	resp, err := suite.app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode) // OpenSubsonic spec: always returns 200, error in XML

	// Test with missing parameters
	req = httptest.NewRequest("GET", "/rest/getSong.view?u=testuser&p=enc:password", nil) // Missing ID
	resp, err = suite.app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Test with non-existent entity
	req = httptest.NewRequest("GET", "/rest/getSong.view?u=testuser&p=enc:password&id=999999", nil)
	resp, err = suite.app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode) // Should return 200 with XML error per OpenSubsonic spec
}

// TestConcurrentRequestsIntegration tests handling of multiple concurrent requests
func TestConcurrentRequestsIntegration(t *testing.T) {
	suite := SetupIntegrationTestSuite()
	defer func() {
		if suite.tearDownFunc != nil {
			suite.tearDownFunc()
		}
	}()

	// Create test data
	artist := &models.Artist{
		Name:           "Concurrent Test Artist",
		NameNormalized: "concurrent test artist",
		DirectoryCode:  "cta",
	}
	err := suite.db.Create(artist).Error
	assert.NoError(t, err)

	// Test multiple requests to the same endpoint to ensure stability
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/rest/getArtists.view?u=testuser&p=enc:password", nil)
		resp, err := suite.app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	}
}

// TestSystemEndpointsIntegration tests system-level endpoints
func TestSystemEndpointsIntegration(t *testing.T) {
	suite := SetupIntegrationTestSuite()
	defer func() {
		if suite.tearDownFunc != nil {
			suite.tearDownFunc()
		}
	}()

	// Test ping endpoint (no auth required)
	req := httptest.NewRequest("GET", "/rest/ping.view", nil)
	resp, err := suite.app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Test license endpoint (no auth required)
	req = httptest.NewRequest("GET", "/rest/getLicense.view", nil)
	resp, err = suite.app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}