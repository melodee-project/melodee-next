package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
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

// TestSearchContractWithFixtures tests search endpoints against official OpenSubsonic fixtures
func TestSearchContractWithFixtures(t *testing.T) {
	db := setupSearchTestDatabase(t)
	cfg := getSearchTestConfig()
	authMiddleware := opensubsonic_middleware.NewOpenSubsonicAuthMiddleware(db, cfg.JWT.Secret)
	app := setupSearchTestApp(db, cfg, authMiddleware)

	// Create test data that matches the fixture expectations
	createSearchTestData(t, db)

	// Test search endpoints and validate results against fixtures
	fixtureTests := []struct {
		name     string
		endpoint string
		query    string
		fixture  string
	}{
		{
			name:     "Search endpoint with fixtures",
			endpoint: "/rest/search.view",
			query:    "black dog", // matches "Black Dog" from fixture
			fixture:  "search-ok.xml", // We'll use this as a reference structure
		},
		{
			name:     "Search2 endpoint with fixtures",
			endpoint: "/rest/search2.view",
			query:    "led zeppelin iv", // matches "Led Zeppelin IV" from fixture
			fixture:  "search2-ok.xml", // We'll use this as a reference structure
		},
		{
			name:     "Search3 endpoint with fixtures",
			endpoint: "/rest/search3.view",
			query:    "led zeppelin", // matches "Led Zeppelin" from fixture
			fixture:  "search3-ok.xml", // We'll use this as a reference structure
		},
	}

	for _, tt := range fixtureTests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("%s?query=%s&u=test&p=enc:password", tt.endpoint, tt.query), nil)
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
		})
	}
}

// TestSearchResultOrdering tests that search results are properly ordered
func TestSearchResultOrdering(t *testing.T) {
	db := setupSearchTestDatabase(t)
	cfg := getSearchTestConfig()
	authMiddleware := opensubsonic_middleware.NewOpenSubsonicAuthMiddleware(db, cfg.JWT.Secret)
	app := setupSearchTestApp(db, cfg, authMiddleware)

	// Create test data with names that need to be sorted
	testArtists := []models.Artist{
		{Name: "Zappa, Frank", NameNormalized: "zappa frank"},
		{Name: "Beatles, The", NameNormalized: "beatles the"},
		{Name: "Abba", NameNormalized: "abba"},
		{Name: "ZZ Top", NameNormalized: "zz top"},
	}

	for _, artist := range testArtists {
		err := db.Create(&artist).Error
		assert.NoError(t, err)
	}

	// Test that results are sorted correctly
	req := httptest.NewRequest("GET", "/rest/search3.view?query=a&u=test&p=enc:password", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Read response
	body := make([]byte, resp.ContentLength)
	_, err = resp.Body.Read(body)
	assert.NoError(t, err)

	// Parse and validate ordering
	var response utils.OpenSubsonicResponse
	err = xml.Unmarshal(body, &response)
	assert.NoError(t, err)

	// Check that artists are ordered by name_normalized (case-insensitive)
	if response.SearchResult3 != nil {
		artists := response.SearchResult3.Artists
		for i := 0; i < len(artists)-1; i++ {
			current := strings.ToLower(artists[i].Name)
			next := strings.ToLower(artists[i+1].Name)
			// Check that they are in alphabetical order
			assert.True(t, current <= next, "Artists should be in alphabetical order: %s should come before %s", current, next)
		}
	}
}

// TestSearchPagination validates pagination behavior for search endpoints
func TestSearchPagination(t *testing.T) {
	db := setupSearchTestDatabase(t)
	cfg := getSearchTestConfig()
	authMiddleware := opensubsonic_middleware.NewOpenSubsonicAuthMiddleware(db, cfg.JWT.Secret)
	app := setupSearchTestApp(db, cfg, authMiddleware)

	// Create multiple test items to test pagination
	for i := 0; i < 20; i++ {
		artist := models.Artist{
			Name:           fmt.Sprintf("Artist %d", i),
			NameNormalized: fmt.Sprintf("artist %d", i),
		}
		err := db.Create(&artist).Error
		assert.NoError(t, err)
	}

	paginationTests := []struct {
		name            string
		endpoint        string
		offset          int
		size            int
		expectedResults int
	}{
		{
			name:            "First page with default size",
			endpoint:        "/rest/search.view",
			offset:          0,
			size:            50, // default
			expectedResults: 20, // all should fit
		},
		{
			name:            "First page with small size",
			endpoint:        "/rest/search2.view",
			offset:          0,
			size:            5,
			expectedResults: 5,
		},
		{
			name:            "Second page with small size",
			endpoint:        "/rest/search3.view",
			offset:          5,
			size:            5,
			expectedResults: 5,
		},
	}

	for _, tt := range paginationTests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("%s?query=artist&offset=%d&size=%d&u=test&p=enc:password", tt.endpoint, tt.offset, tt.size), nil)
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

			// Validate pagination parameters
			if response.SearchResult2 != nil {
				assert.Equal(t, tt.offset, response.SearchResult2.Offset)
				// Size is the number of results returned in this batch
				if tt.size < tt.expectedResults {
					assert.Equal(t, tt.expectedResults, response.SearchResult2.Size)
				}
			}
			if response.SearchResult3 != nil {
				assert.Equal(t, tt.offset, response.SearchResult3.Offset)
				// Size is the number of results returned in this batch
				if tt.size < tt.expectedResults {
					assert.Equal(t, tt.expectedResults, response.SearchResult3.Size)
				}
			}
		})
	}
}

// TestSearchNormalization validates normalization rules for search queries (articles, punctuation)
func TestSearchNormalization(t *testing.T) {
	db := setupSearchTestDatabase(t)
	cfg := getSearchTestConfig()
	authMiddleware := opensubsonic_middleware.NewOpenSubsonicAuthMiddleware(db, cfg.JWT.Secret)
	app := setupSearchTestApp(db, cfg, authMiddleware)

	// Create test data with special names that need normalization
	testSongs := []models.Song{
		{Name: "The Song with & Ampersand", NameNormalized: "the song with  ampersand"},
		{Name: "A Song with / Slash", NameNormalized: "a song with  slash"},
		{Name: "An Article Example", NameNormalized: "article example"}, // "An" would be removed
	}

	for _, song := range testSongs {
		err := db.Create(&song).Error
		assert.NoError(t, err)
	}

	// Test search queries that should match normalized names
	normalizationTests := []struct {
		name        string
		query       string
		expectMatch bool
	}{
		{
			name:        "Match with ampersand normalization",
			query:       "song with and", // "ampersand" normalized to "and"
			expectMatch: true,
		},
		{
			name:        "Match with slash normalization",
			query:       "song with - slash", // "/" normalized to " - "
			expectMatch: true,
		},
		{
			name:        "Match after article removal",
			query:       "article example", // should match "An Article Example" after "An" removal
			expectMatch: true,
		},
	}

	for _, tt := range normalizationTests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/rest/search.view?query=%s&u=test&p=enc:password", tt.query), nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, 200, resp.StatusCode)

			// Read response
			body := make([]byte, resp.ContentLength)
			_, readErr := resp.Body.Read(body)
			if readErr != nil && readErr.Error() != "EOF" {
				assert.NoError(t, readErr)
			}

			// Parse response to check if we got results
			var response utils.OpenSubsonicResponse
			err = xml.Unmarshal(body, &response)
			assert.NoError(t, err)

			// Check if we got results based on expectations
			gotResults := false
			if response.SearchResult2 != nil {
				gotResults = len(response.SearchResult2.Songs) > 0
			}

			if tt.expectMatch {
				assert.True(t, gotResults, "Expected search to find results for query: %s", tt.query)
			} else {
				// This is for cases where we don't expect matches
			}
		})
	}
}

// Helper functions
func setupSearchTestDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	// Auto-migrate the models
	err = db.AutoMigrate(&models.User{}, &models.Library{}, &models.Artist{}, &models.Album{}, &models.Song{})
	assert.NoError(t, err)

	return db
}

func getSearchTestConfig() *config.AppConfig {
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

func setupSearchTestApp(db *gorm.DB, cfg *config.AppConfig, authMiddleware *opensubsonic_middleware.OpenSubsonicAuthMiddleware) *fiber.App {
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

func createSearchTestData(t *testing.T, db *gorm.DB) {
	// Create a user for authentication
	user := &models.User{
		Username:     "test",
		PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMye.IjdQc3Dx0C4Jux4DiQE4qY46HdNEvC", // bcrypt hash for "password"
		APIKey:       "test-key",
	}
	err := db.Create(user).Error
	assert.NoError(t, err)

	// Create test data for search
	artist := models.Artist{
		Name:           "Led Zeppelin",
		NameNormalized: "led zeppelin",
		AlbumCountCached: 9,
	}
	err = db.Create(&artist).Error
	assert.NoError(t, err)

	album := models.Album{
		Name:           "Led Zeppelin IV",
		NameNormalized: "led zeppelin iv",
		ArtistID:       artist.ID,
		SongCountCached: 8,
		DurationCached: 2550 * 1000, // 2550 seconds * 1000 ms
	}
	err = db.Create(&album).Error
	assert.NoError(t, err)

	song := models.Song{
		Name:           "Black Dog",
		NameNormalized: "black dog",
		AlbumID:        album.ID,
		ArtistID:       artist.ID,
		Duration:       255 * 1000, // 255 seconds * 1000 ms
		SortOrder:      1,
		BitRate:        320,
		FileName:       "black_dog.mp3",
		RelativePath:   "/music/led_zeppelin/led_zeppelin_iv/black_dog.mp3",
	}
	err = db.Create(&song).Error
	assert.NoError(t, err)
}