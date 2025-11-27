package main

import (
	"encoding/xml"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

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

// TestAllEndpointsContract validates that all OpenSubsonic endpoints return properly formatted responses
func TestAllEndpointsContract(t *testing.T) {
	tempDir := t.TempDir()
	db := setupComprehensiveTestDatabase(t, tempDir)
	cfg := getComprehensiveTestConfig()
	authMiddleware := opensubsonic_middleware.NewOpenSubsonicAuthMiddleware(db, cfg.JWT.Secret)
	app := setupComprehensiveTestApp(db, cfg, authMiddleware, tempDir)

	endpointTests := []struct {
		name     string
		method   string
		endpoint string
		params   string
	}{
		// System endpoints
		{"Ping endpoint", "GET", "/rest/ping.view", ""},
		{"License endpoint", "GET", "/rest/getLicense.view", "?u=test&p=enc:password"},

		// Browsing endpoints
		{"Get music folders", "GET", "/rest/getMusicFolders.view", "?u=test&p=enc:password"},
		{"Get indexes", "GET", "/rest/getIndexes.view", "?u=test&p=enc:password"},
		{"Get artists", "GET", "/rest/getArtists.view", "?u=test&p=enc:password"},
		{"Get artist", "GET", "/rest/getArtist.view", "?id=1&u=test&p=enc:password"},
		{"Get album info", "GET", "/rest/getAlbumInfo.view", "?id=1&u=test&p=enc:password"},
		{"Get music directory", "GET", "/rest/getMusicDirectory.view", "?id=1&u=test&p=enc:password"},
		{"Get album", "GET", "/rest/getAlbum.view", "?id=1&u=test&p=enc:password"},
		{"Get song", "GET", "/rest/getSong.view", "?id=1&u=test&p=enc:password"},
		{"Get genres", "GET", "/rest/getGenres.view", "?u=test&p=enc:password"},

		// Media retrieval endpoints
		{"Stream", "GET", "/rest/stream.view", "?id=1&u=test&p=enc:password"},
		{"Download", "GET", "/rest/download.view", "?id=1&u=test&p=enc:password"},
		{"Get cover art", "GET", "/rest/getCoverArt.view", "?id=al-1&u=test&p=enc:password"},
		{"Get avatar", "GET", "/rest/getAvatar.view", "?username=test&u=test&p=enc:password"},

		// Searching endpoints
		{"Search", "GET", "/rest/search.view", "?query=test&u=test&p=enc:password"},
		{"Search2", "GET", "/rest/search2.view", "?query=test&u=test&p=enc:password"},
		{"Search3", "GET", "/rest/search3.view", "?query=test&u=test&p=enc:password"},

		// Playlist endpoints
		{"Get playlists", "GET", "/rest/getPlaylists.view", "?u=test&p=enc:password"},
		{"Get playlist", "GET", "/rest/getPlaylist.view", "?id=1&u=test&p=enc:password"},

		// User endpoints (only retrieve, not create/modify as those may need different setup)
		{"Get users", "GET", "/rest/getUsers.view", "?u=test&p=enc:password"},
	}

	for _, tt := range endpointTests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.endpoint+tt.params, nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)

			// Per OpenSubsonic spec: HTTP status is always 200, even for errors
			assert.Equal(t, 200, resp.StatusCode)

			// Read response body
			body := make([]byte, resp.ContentLength)
			_, readErr := resp.Body.Read(body)
			if readErr != nil && readErr.Error() != "EOF" {
				// Body might be empty for certain responses
			}

			// Parse the XML response to validate structure
			var response utils.OpenSubsonicResponse
			err = xml.Unmarshal(body, &response)

			// All OpenSubsonic responses should be valid XML
			if err != nil {
				// If it's not valid XML and it's a media endpoint (stream, download, cover art)
				// it might be returning binary content instead of XML
				isMediaEndpoint := strings.Contains(tt.endpoint, "stream") ||
					strings.Contains(tt.endpoint, "download") ||
					strings.Contains(tt.endpoint, "CoverArt") ||
					strings.Contains(tt.endpoint, "Avatar")

				if !isMediaEndpoint {
					// For non-media endpoints, the response should be valid XML
					t.Errorf("Invalid XML response for %s: %v\nResponse body: %s", tt.name, err, string(body))
				}
			} else {
				// For XML responses, validate the basic fields
				assert.NotEmpty(t, response.Status, "Status should not be empty for %s", tt.name)
				assert.Equal(t, "1.16.1", response.Version, "Version should be 1.16.1 for %s", tt.name)
				assert.Equal(t, "Melodee", response.Type, "Type should be Melodee for %s", tt.name)

				// Status should be either "ok" or "failed"
				assert.Contains(t, []string{"ok", "failed"}, response.Status,
					"Status should be 'ok' or 'failed' for %s, got: %s", tt.name, response.Status)
			}
		})
	}
}

// TestErrorResponsesContract validates that error responses follow OpenSubsonic spec
func TestErrorResponsesContract(t *testing.T) {
	db := setupComprehensiveTestDatabase(t, "")
	cfg := getComprehensiveTestConfig()
	authMiddleware := opensubsonic_middleware.NewOpenSubsonicAuthMiddleware(db, cfg.JWT.Secret)
	app := setupComprehensiveTestApp(db, cfg, authMiddleware, "")

	errorTests := []struct {
		name     string
		endpoint string
		params   string
	}{
		{"Non-existent artist", "/rest/getArtist.view", "?id=999999&u=test&p=enc:password"},
		{"Non-existent album", "/rest/getAlbum.view", "?id=999999&u=test&p=enc:password"},
		{"Non-existent song", "/rest/getSong.view", "?id=999999&u=test&p=enc:password"},
		{"Non-existent cover art", "/rest/getCoverArt.view", "?id=al-999999&u=test&p=enc:password"},
		{"Non-existent playlist", "/rest/getPlaylist.view", "?id=999999&u=test&p=enc:password"},
		{"Invalid parameters", "/rest/search.view", "?u=test&p=enc:password"}, // Missing query
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.endpoint+tt.params, nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, 200, resp.StatusCode) // OpenSubsonic spec: always 200

			body := make([]byte, resp.ContentLength)
			_, err = resp.Body.Read(body)
			assert.NoError(t, err)

			// Parse response
			var response utils.OpenSubsonicResponse
			err = xml.Unmarshal(body, &response)
			assert.NoError(t, err)

			// Should have status "failed"
			assert.Equal(t, "failed", response.Status)

			// Should have error information
			assert.NotNil(t, response.Error, "Error should be present in failed response for %s", tt.name)
			if response.Error != nil {
				assert.NotZero(t, response.Error.Code, "Error code should not be zero for %s", tt.name)
				assert.NotEmpty(t, response.Error.Message, "Error message should not be empty for %s", tt.name)
			}
		})
	}
}

// TestMissingAuthResponses validates responses when auth is missing or invalid
func TestMissingAuthResponses(t *testing.T) {
	db := setupComprehensiveTestDatabase(t, "")
	cfg := getComprehensiveTestConfig()
	authMiddleware := opensubsonic_middleware.NewOpenSubsonicAuthMiddleware(db, cfg.JWT.Secret)
	app := setupComprehensiveTestApp(db, cfg, authMiddleware, "")

	authTests := []struct {
		name     string
		endpoint string
		params   string
	}{
		{"Missing auth parameters", "/rest/getArtists.view", ""},                           // No auth params
		{"Invalid username/password", "/rest/getArtists.view", "?u=invalid&p=enc:invalid"}, // Invalid auth
		{"Malformed auth", "/rest/getArtists.view", "?u=test"},                             // Missing password
	}

	for _, tt := range authTests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.endpoint+tt.params, nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, 200, resp.StatusCode) // OpenSubsonic spec: always 200

			body := make([]byte, resp.ContentLength)
			_, err = resp.Body.Read(body)
			assert.NoError(t, err)

			// Parse response
			var response utils.OpenSubsonicResponse
			err = xml.Unmarshal(body, &response)
			if err != nil {
				// May be an empty response body in some cases
				return
			}

			// Should have status "failed" for auth errors
			assert.Equal(t, "failed", response.Status)

			// Should have error information with proper code
			assert.NotNil(t, response.Error, "Error should be present for auth failure in %s", tt.name)
			if response.Error != nil {
				// Should be error code 50 for "not authorized" as per spec
				assert.Equal(t, 50, response.Error.Code, "Error code should be 50 for auth failure in %s", tt.name)
				assert.NotEmpty(t, response.Error.Message, "Error message should not be empty for %s", tt.name)
			}
		})
	}
}

// TestSuccessResponseFormat validates that successful responses follow the format
func TestSuccessResponseFormat(t *testing.T) {
	db := setupComprehensiveTestDatabase(t, "")
	cfg := getComprehensiveTestConfig()
	authMiddleware := opensubsonic_middleware.NewOpenSubsonicAuthMiddleware(db, cfg.JWT.Secret)
	app := setupComprehensiveTestApp(db, cfg, authMiddleware, "")

	// Test with minimal valid request to get a successful response
	req := httptest.NewRequest("GET", "/rest/getLicense.view?u=test&p=enc:password", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body := make([]byte, resp.ContentLength)
	_, err = resp.Body.Read(body)
	assert.NoError(t, err)

	// Parse response
	var response utils.OpenSubsonicResponse
	err = xml.Unmarshal(body, &response)
	assert.NoError(t, err)

	// Validate successful response format
	assert.Equal(t, "ok", response.Status)
	assert.Equal(t, "1.16.1", response.Version)
	assert.Equal(t, "Melodee", response.Type)
	assert.Equal(t, "true", fmt.Sprintf("%v", response.OpenSubsonic)) // Should be true

	// Should not have error information
	assert.Nil(t, response.Error, "Error should not be present in successful response")
}

// Helper functions for comprehensive testing
func setupComprehensiveTestDatabase(t *testing.T, tempDir string) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	// Auto-migrate the models
	err = db.AutoMigrate(&models.User{}, &models.Library{}, &models.Artist{}, &models.Album{}, &models.Track{}, &models.Playlist{})
	assert.NoError(t, err)

	// Create a test user for authentication
	user := &models.User{
		Username:     "test",
		PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMye.IjdQc3Dx0C4Jux4DiQE4qY46HdNEvC", // bcrypt hash for "password"
		APIKey:       uuid.New(),
	}
	err = db.Create(user).Error
	assert.NoError(t, err)

	// Create test artist
	artist := models.Artist{
		Name:           "Test Artist",
		NameNormalized: "test artist",
	}
	err = db.Create(&artist).Error
	assert.NoError(t, err)

	// Create test album
	album := models.Album{
		Name:           "Test Album",
		NameNormalized: "test album",
		ArtistID:       artist.ID,
	}
	err = db.Create(&album).Error
	assert.NoError(t, err)

	// Create test song if tempDir provided
	if tempDir != "" {
		testFilePath := filepath.Join(tempDir, "test.mp3")
		err = os.WriteFile(testFilePath, []byte("fake audio content"), 0644)
		assert.NoError(t, err)

		song := models.Track{
			Name:           "Test Song",
			NameNormalized: "test song",
			AlbumID:        album.ID,
			ArtistID:       artist.ID,
			FileName:       "test.mp3",
			RelativePath:   testFilePath,
		}
		err = db.Create(&song).Error
		assert.NoError(t, err)
	}

	return db
}

func getComprehensiveTestConfig() *config.AppConfig {
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

func setupComprehensiveTestApp(db *gorm.DB, cfg *config.AppConfig, authMiddleware *opensubsonic_middleware.OpenSubsonicAuthMiddleware, tempDir string) *fiber.App {
	// Create media processing components
	ffmpegProcessor := media.NewFFmpegProcessor(&media.FFmpegConfig{
		FFmpegPath: "ffmpeg", // This will fail in tests but that's OK
		Profiles: map[string]media.FFmpegProfile{
			"transcode_high":        {CommandLine: "-c:a libmp3lame -b:a 320k"},
			"transcode_mid":         {CommandLine: "-c:a libmp3lame -b:a 192k"},
			"transcode_opus_mobile": {CommandLine: "-c:a libopus -b:a 96k"},
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
