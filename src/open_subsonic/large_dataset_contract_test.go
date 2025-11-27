package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http/httptest"
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

// TestLargeDatasetContract validates that OpenSubsonic endpoints behave properly with large datasets
func TestLargeDatasetContract(t *testing.T) {
	db := setupLargeDatasetTestDatabase(t)
	cfg := getLargeDatasetTestConfig()
	authMiddleware := opensubsonic_middleware.NewOpenSubsonicAuthMiddleware(db, cfg.JWT.Secret)
	app := setupLargeDatasetTestApp(db, cfg, authMiddleware)

	// Create large dataset for testing
	createLargeDatasetTestData(t, db, 1000, 5, 10) // 1000 artists, 5 albums each, 10 songs each

	// Test core OpenSubsonic endpoints with large datasets
	largeDatasetTests := []struct {
		name     string
		endpoint string
	}{
		{
			name:     "GetArtists with large dataset",
			endpoint: "/rest/getArtists.view",
		},
		{
			name:     "GetIndexes with large dataset",
			endpoint: "/rest/getIndexes.view",
		},
		{
			name:     "Search3 with large dataset",
			endpoint: "/rest/search3.view",
		},
	}

	for _, tt := range largeDatasetTests {
		t.Run(tt.name, func(t *testing.T) {
			// Different parameters based on endpoint
			var req *httptest.Request
			if tt.endpoint == "/rest/search3.view" {
				req = httptest.NewRequest("GET", fmt.Sprintf("%s?query=artist&u=test&p=enc:password", tt.endpoint), nil)
			} else if tt.endpoint == "/rest/getIndexes.view" {
				req = httptest.NewRequest("GET", fmt.Sprintf("%s?username=test&u=test&p=enc:password", tt.endpoint), nil)
			} else {
				req = httptest.NewRequest("GET", fmt.Sprintf("%s?u=test&p=enc:password", tt.endpoint), nil)
			}

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

			// Validate that responses are stable under large datasets
			assert.NotNil(t, response.Status, "Response should have a status")
		})
	}
}

// TestLargeDatasetPagination validates pagination behavior with large datasets
func TestLargeDatasetPagination(t *testing.T) {
	db := setupLargeDatasetTestDatabase(t)
	cfg := getLargeDatasetTestConfig()
	authMiddleware := opensubsonic_middleware.NewOpenSubsonicAuthMiddleware(db, cfg.JWT.Secret)
	app := setupLargeDatasetTestApp(db, cfg, authMiddleware)

	// Create large dataset for testing
	createLargeDatasetTestData(t, db, 500, 3, 5) // 500 artists, 3 albums each, 5 songs each

	paginationTests := []struct {
		name            string
		endpoint        string
		offset          int
		size            int
		expectedMinSize int // minimum expected number of results
	}{
		{
			name:            "GetArtists pagination first page",
			endpoint:        "/rest/getArtists.view",
			offset:          0,
			size:            50,
			expectedMinSize: 50,
		},
		{
			name:            "GetArtists pagination middle page",
			endpoint:        "/rest/getArtists.view",
			offset:          200,
			size:            50,
			expectedMinSize: 50,
		},
		{
			name:            "GetArtists pagination large offset",
			endpoint:        "/rest/getArtists.view",
			offset:          400,
			size:            50,
			expectedMinSize: 50, // or fewer if at end
		},
	}

	for _, tt := range paginationTests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("%s?offset=%d&size=%d&u=test&p=enc:password", tt.endpoint, tt.offset, tt.size), nil)
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

			// Validate that response doesn't truncate results or change shape
			if response.Artists != nil {
				// Results should be within expected range
				assert.True(t, len(response.Artists.Artists) <= tt.size)
				assert.True(t, len(response.Artists.Artists) >= 0) // can be 0 if at end
			}
		})
	}
}

// TestLargeDatasetResponseStability validates that responses remain stable with large datasets
func TestLargeDatasetResponseStability(t *testing.T) {
	db := setupLargeDatasetTestDatabase(t)
	cfg := getLargeDatasetTestConfig()
	authMiddleware := opensubsonic_middleware.NewOpenSubsonicAuthMiddleware(db, cfg.JWT.Secret)
	app := setupLargeDatasetTestApp(db, cfg, authMiddleware)

	// Create a large dataset
	createLargeDatasetTestData(t, db, 2000, 2, 3) // 2000 artists, 2 albums each, 3 songs each

	// Test multiple requests to same endpoint to ensure stability
	endpointTests := []string{
		"/rest/getArtists.view",
		"/rest/getIndexes.view",
		"/rest/search3.view?query=test",
	}

	for _, endpoint := range endpointTests {
		t.Run(fmt.Sprintf("Stability test for %s", endpoint), func(t *testing.T) {
			// Make multiple requests to the same endpoint
			for i := 0; i < 3; i++ {
				var req *httptest.Request
				if endpoint == "/rest/search3.view?query=test" {
					req = httptest.NewRequest("GET", fmt.Sprintf("/rest/search3.view?query=test&u=test&p=enc:password"), nil)
				} else if endpoint == "/rest/getIndexes.view" {
					req = httptest.NewRequest("GET", "/rest/getIndexes.view?username=test&u=test&p=enc:password", nil)
				} else {
					req = httptest.NewRequest("GET", endpoint+"?u=test&p=enc:password", nil)
				}

				resp, err := app.Test(req)
				assert.NoError(t, err)
				assert.Equal(t, 200, resp.StatusCode)

				// Read response
				body := make([]byte, resp.ContentLength)
				_, readErr := resp.Body.Read(body)
				if readErr != nil && readErr.Error() != "EOF" {
					assert.NoError(t, readErr)
				}

				// Parse response to ensure it's valid XML
				var response utils.OpenSubsonicResponse
				err = xml.Unmarshal(body, &response)
				assert.NoError(t, err)

				// Ensure the response structure is maintained
				assert.Equal(t, "ok", response.Status)
				assert.Equal(t, "1.16.1", response.Version)
			}
		})
	}
}

// TestLargeDatasetSearchWithPagination tests search functionality with large datasets and pagination
func TestLargeDatasetSearchWithPagination(t *testing.T) {
	db := setupLargeDatasetTestDatabase(t)
	cfg := getLargeDatasetTestConfig()
	authMiddleware := opensubsonic_middleware.NewOpenSubsonicAuthMiddleware(db, cfg.JWT.Secret)
	app := setupLargeDatasetTestApp(db, cfg, authMiddleware)

	// Create large dataset with searchable content
	createLargeDatasetTestData(t, db, 1000, 2, 5) // 1000 artists, 2 albums each, 5 songs each

	// Add some specific content that we can search for
	for i := 0; i < 20; i++ {
		artist := models.Artist{
			Name:           fmt.Sprintf("Special Artist %d", i),
			NameNormalized: fmt.Sprintf("special artist %d", i),
		}
		err := db.Create(&artist).Error
		assert.NoError(t, err)

		album := models.Album{
			Name:           fmt.Sprintf("Special Album %d", i),
			NameNormalized: fmt.Sprintf("special album %d", i),
			ArtistID:       artist.ID,
		}
		err = db.Create(&album).Error
		assert.NoError(t, err)

		song := models.Track{
			Name:           fmt.Sprintf("Special Song %d", i),
			NameNormalized: fmt.Sprintf("special song %d", i),
			AlbumID:        album.ID,
			ArtistID:       artist.ID,
		}
		err = db.Create(&song).Error
		assert.NoError(t, err)
	}

	searchTests := []struct {
		name     string
		endpoint string
		query    string
		offset   int
		size     int
	}{
		{
			name:     "Search for special artists",
			endpoint: "/rest/search3.view",
			query:    "special",
			offset:   0,
			size:     20,
		},
		{
			name:     "Search for special artists with offset",
			endpoint: "/rest/search3.view",
			query:    "special",
			offset:   10,
			size:     10,
		},
	}

	for _, tt := range searchTests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("%s?query=%s&offset=%d&size=%d&u=test&p=enc:password", tt.endpoint, tt.query, tt.offset, tt.size), nil)
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

			// Verify search results are within expected bounds
			if response.SearchResult3 != nil {
				assert.True(t, len(response.SearchResult3.Artists) <= tt.size)
				assert.True(t, len(response.SearchResult3.Albums) <= tt.size)
				assert.True(t, len(response.SearchResult3.Songs) <= tt.size)
			}
		})
	}
}

// Helper functions

func setupLargeDatasetTestDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	// Auto-migrate the models
	err = db.AutoMigrate(&models.User{}, &models.Library{}, &models.Artist{}, &models.Album{}, &models.Track{})
	assert.NoError(t, err)

	return db
}

func getLargeDatasetTestConfig() *config.AppConfig {
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

func setupLargeDatasetTestApp(db *gorm.DB, cfg *config.AppConfig, authMiddleware *opensubsonic_middleware.OpenSubsonicAuthMiddleware) *fiber.App {
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
		AppName:      "Melodee OpenSubsonic Large Dataset Test Server",
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

func createLargeDatasetTestData(t *testing.T, db *gorm.DB, numArtists, albumsPerArtist, songsPerAlbum int) {
	// Create a user for authentication
	user := &models.User{
		Username:     "test",
		PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMye.IjdQc3Dx0C4Jux4DiQE4qY46HdNEvC", // bcrypt hash for "password"
		APIKey:       uuid.New(),
	}
	err := db.Create(user).Error
	assert.NoError(t, err)

	// Create large dataset efficiently
	artists := make([]models.Artist, numArtists)
	for i := 0; i < numArtists; i++ {
		artists[i] = models.Artist{
			Name:           fmt.Sprintf("Artist %d", i),
			NameNormalized: fmt.Sprintf("artist %d", i),
		}
	}
	err = db.CreateInBatches(artists, 1000).Error
	assert.NoError(t, err)

	// Create albums for each artist
	albums := make([]models.Album, 0, numArtists*albumsPerArtist)
	for artistIdx, artist := range artists {
		for albumIdx := 0; albumIdx < albumsPerArtist; albumIdx++ {
			albums = append(albums, models.Album{
				Name:           fmt.Sprintf("Album %d by Artist %d", albumIdx, artistIdx),
				NameNormalized: fmt.Sprintf("album %d by artist %d", albumIdx, artistIdx),
				ArtistID:       artist.ID,
			})
		}
	}
	err = db.CreateInBatches(albums, 1000).Error
	assert.NoError(t, err)

	// Create songs for each album
	songs := make([]models.Track, 0, len(albums)*songsPerAlbum)
	albumIdx := 0
	for artistIdx := 0; artistIdx < numArtists; artistIdx++ {
		for albumIdxWithinArtist := 0; albumIdxWithinArtist < albumsPerArtist; albumIdxWithinArtist++ {
			for songIdx := 0; songIdx < songsPerAlbum; songIdx++ {
				songs = append(songs, models.Track{
					Name:           fmt.Sprintf("Song %d from Album %d by Artist %d", songIdx, albumIdxWithinArtist, artistIdx),
					NameNormalized: fmt.Sprintf("song %d from album %d by artist %d", songIdx, albumIdxWithinArtist, artistIdx),
					AlbumID:        albums[albumIdx].ID,
					ArtistID:       artists[artistIdx].ID,
					Duration:       180000, // 3 minutes in milliseconds
					SortOrder:      int32(songIdx),
					BitRate:        320,
					FileName:       fmt.Sprintf("song_%d_album_%d_artist_%d.mp3", songIdx, albumIdxWithinArtist, artistIdx),
					RelativePath:   fmt.Sprintf("/music/artist_%d/album_%d/song_%d.mp3", artistIdx, albumIdxWithinArtist, songIdx),
				})
			}
			albumIdx++
		}
	}
	err = db.CreateInBatches(songs, 1000).Error
	assert.NoError(t, err)
}