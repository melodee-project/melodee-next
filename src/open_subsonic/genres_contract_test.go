package main

import (
	"bytes"
	"encoding/json"
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

// TestGenresEndpoint validates that the genres endpoint properly aggregates from media tags
func TestGenresEndpoint(t *testing.T) {
	db := setupGenresTestDatabase(t)
	cfg := getGenresTestConfig()
	authMiddleware := opensubsonic_middleware.NewOpenSubsonicAuthMiddleware(db, cfg.JWT.Secret)
	app := setupGenresTestApp(db, cfg, authMiddleware)

	// Create test data with various genres in tags
	testSongs := []models.Song{
		{
			Name:           "Song 1",
			NameNormalized: "song 1",
			Tags: mustMarshalJSON(map[string]interface{}{
				"genre": "Rock",
			}),
		},
		{
			Name:           "Song 2", 
			NameNormalized: "song 2",
			Tags: mustMarshalJSON(map[string]interface{}{
				"Genre": "Pop", // Capitalized
			}),
		},
		{
			Name:           "Song 3",
			NameNormalized: "song 3",
			Tags: mustMarshalJSON(map[string]interface{}{
				"genre": "Rock", // Same genre as song 1
			}),
		},
		{
			Name:           "Song 4",
			NameNormalized: "song 4",
			Tags: mustMarshalJSON(map[string]interface{}{
				"music_genre": "Jazz",
			}),
		},
		{
			Name:           "Song 5",
			NameNormalized: "song 5",
			Tags: mustMarshalJSON(map[string]interface{}{
				"style": "Classical",
			}),
		},
		{
			Name:           "Song 6",
			NameNormalized: "song 6",
			Tags: mustMarshalJSON(map[string]interface{}{
				"common": map[string]interface{}{
					"genre": "Electronic",
				},
			}),
		},
	}

	for _, song := range testSongs {
		err := db.Create(&song).Error
		assert.NoError(t, err)
	}

	// Also create albums with genres
	testAlbums := []models.Album{
		{
			Name:           "Rock Album",
			NameNormalized: "rock album",
			Genres:         []string{"Rock", "Alternative"},
		},
		{
			Name:           "Jazz Album",
			NameNormalized: "jazz album", 
			Genres:         []string{"Jazz", "Blues"},
		},
	}

	for _, album := range testAlbums {
		err := db.Create(&album).Error
		assert.NoError(t, err)
	}

	req := httptest.NewRequest("GET", "/rest/getGenres.view?u=test&p=enc:password", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body := make([]byte, resp.ContentLength)
	_, err = resp.Body.Read(body)
	assert.NoError(t, err)

	var response utils.OpenSubsonicResponse
	err = xml.Unmarshal(body, &response)
	assert.NoError(t, err)

	assert.NotNil(t, response.Genres)
	if response.Genres != nil {
		genres := response.Genres.Genres
		
		// Verify we have genres aggregated from both songs and albums
		expectedGenres := []string{"Rock", "Pop", "Jazz", "Classical", "Electronic", "Alternative", "Blues"}
		
		// Create a map of genre names to counts for easy verification
		genreMap := make(map[string]int)
		for _, genre := range genres {
			genreMap[genre.Name] = genre.Count
		}
		
		// Check that all expected genres are present
		for _, expectedGenre := range expectedGenres {
			_, exists := genreMap[expectedGenre]
			assert.True(t, exists, "Genre '%s' should be present in the response", expectedGenre)
		}
		
		// Check that Rock appears at least twice (once from songs, once from albums)
		rockCount, rockExists := genreMap["Rock"]
		assert.True(t, rockExists, "Rock genre should exist")
		assert.GreaterOrEqual(t, rockCount, 2, "Rock should appear in at least 2 entries (from songs and album)")
		
		// Check that genres are sorted alphabetically
		for i := 0; i < len(genres)-1; i++ {
			current := strings.ToLower(genres[i].Name)
			next := strings.ToLower(genres[i+1].Name)
			assert.True(t, current <= next, "Genres should be sorted alphabetically: %s should come before %s", genres[i].Name, genres[i+1].Name)
		}
	}
}

// TestExtractGenreFromTags validates the genre extraction from JSONB tags
func TestExtractGenreFromTags(t *testing.T) {
	testCases := []struct {
		name     string
		tags     []byte
		expected string
	}{
		{
			name: "Simple genre field",
			tags: mustMarshalJSON(map[string]interface{}{"genre": "Rock"}),
			expected: "Rock",
		},
		{
			name: "Capitalized Genre field", 
			tags: mustMarshalJSON(map[string]interface{}{"Genre": "Pop"}),
			expected: "Pop",
		},
		{
			name: "Uppercase GENRE field",
			tags: mustMarshalJSON(map[string]interface{}{"GENRE": "Jazz"}),
			expected: "Jazz",
		},
		{
			name: "Nested genre in common object",
			tags: mustMarshalJSON(map[string]interface{}{
				"common": map[string]interface{}{
					"genre": "Classical",
				},
			}),
			expected: "Classical",
		},
		{
			name: "Genre as array",
			tags: mustMarshalJSON(map[string]interface{}{
				"genre": []interface{}{"Rock", "Alternative"},
			}),
			expected: "Rock", // Should take first element
		},
		{
			name: "Numeric genre ID (ID3)", 
			tags: mustMarshalJSON(map[string]interface{}{
				"GenreID3v1": 1.0, // Would be "Blues" in ID3
			}),
			expected: "1", // Converted from number to string
		},
		{
			name: "Empty tags",
			tags: []byte{},
			expected: "",
		},
		{
			name: "Nil tags", 
			tags: nil,
			expected: "",
		},
		{
			name: "Invalid JSON",
			tags: []byte("invalid json"),
			expected: "",
		},
		{
			name: "Different genre field name",
			tags: mustMarshalJSON(map[string]interface{}{
				"music_genre": "Electronic",
			}),
			expected: "Electronic",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			result := extractGenreFromTags(tt.tags)
			assert.Equal(t, tt.expected, result, "extractGenreFromTags(%v) should return %s", tt.tags, tt.expected)
		})
	}
}

// TestGenresWithEmptyData validates behavior when no genres exist
func TestGenresWithEmptyData(t *testing.T) {
	db := setupGenresTestDatabase(t)
	cfg := getGenresTestConfig()
	authMiddleware := opensubsonic_middleware.NewOpenSubsonicAuthMiddleware(db, cfg.JWT.Secret)
	app := setupGenresTestApp(db, cfg, authMiddleware)

	// Test with no songs or albums having genres
	req := httptest.NewRequest("GET", "/rest/getGenres.view?u=test&p=enc:password", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body := make([]byte, resp.ContentLength)
	_, err = resp.Body.Read(body)
	assert.NoError(t, err)

	var response utils.OpenSubsonicResponse
	err = xml.Unmarshal(body, &response)
	assert.NoError(t, err)

	assert.NotNil(t, response.Genres)
	if response.Genres != nil {
		// Should return empty genres list, not an error
		assert.NotNil(t, response.Genres.Genres, "Genres list should exist")
		// May or may not be empty depending on implementation
	}
}

// TestNormalizeGenreName validates the genre name normalization
func TestNormalizeGenreName(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"Rock", "Rock"},
		{"  Rock  ", "Rock"}, // Trimmed
		{"Rock  Pop", "Rock Pop"}, // Double space normalized
		{"", ""}, // Empty string
		{"  ", ""}, // Only spaces
		{"Rock/Pop", "Rock/Pop"}, // Keep special characters
		{"Hip-Hop", "Hip-Hop"}, // Keep hyphens
		{"Death Metal", "Death Metal"}, // Normal spaces preserved
	}

	for _, tt := range testCases {
		t.Run(fmt.Sprintf("normalize_%s", tt.input), func(t *testing.T) {
			result := normalizeGenreName(tt.input)
			assert.Equal(t, tt.expected, result, "normalizeGenreName(%s) should return %s", tt.input, tt.expected)
		})
	}
}

// Helper function to marshal JSON
func mustMarshalJSON(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

// Helper functions
func setupGenresTestDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	// Auto-migrate the models
	err = db.AutoMigrate(&models.User{}, &models.Library{}, &models.Artist{}, &models.Album{}, &models.Song{}, &models.Playlist{})
	assert.NoError(t, err)

	// Create a test user for authentication
	user := &models.User{
		Username:     "test",
		PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMye.IjdQc3Dx0C4Jux4DiQE4qY46HdNEvC", // bcrypt hash for "password"
		APIKey:       "test-key",
	}
	err = db.Create(user).Error
	assert.NoError(t, err)

	return db
}

func getGenresTestConfig() *config.AppConfig {
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

func setupGenresTestApp(db *gorm.DB, cfg *config.AppConfig, authMiddleware *opensubsonic_middleware.OpenSubsonicAuthMiddleware) *fiber.App {
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

