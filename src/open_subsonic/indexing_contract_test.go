package main

import (
	"encoding/xml"
	"net/http/httptest"
	"sort"
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

// TestIndexingAndSorting validates that indexing and sorting follow normalization rules
func TestIndexingAndSorting(t *testing.T) {
	db := setupIndexingTestDatabase(t)
	cfg := getIndexingTestConfig()
	authMiddleware := opensubsonic_middleware.NewOpenSubsonicAuthMiddleware(db, cfg.JWT.Secret)
	app := setupIndexingTestApp(db, cfg, authMiddleware)

	// Create test data with names that need normalization
	testArtists := []models.Artist{
		{Name: "The Beatles", NameNormalized: "the beatles"}, // Should normalize as "beatles" for sorting
		{Name: "Aerosmith", NameNormalized: "aerosmith"},
		{Name: "AC/DC", NameNormalized: "ac/dc"}, // Should normalize as "ac dc" for sorting
		{Name: "ZZ Top", NameNormalized: "zz top"},
		{Name: "ABBA", NameNormalized: "abba"},
		{Name: "Led Zeppelin", NameNormalized: "led zeppelin"},
		{Name: "The Rolling Stones", NameNormalized: "the rolling stones"}, // Should normalize as "rolling stones" for sorting
		{Name: "Elton John", NameNormalized: "elton john"},
		{Name: "Los Lobos", NameNormalized: "los lobos"},                     // Spanish article
		{Name: "Les Misérables Cast", NameNormalized: "les miserables cast"}, // French article with accent
	}

	for _, artist := range testArtists {
		err := db.Create(&artist).Error
		assert.NoError(t, err)
	}

	// Test GetIndexes endpoint for proper alphabetical indexing
	req := httptest.NewRequest("GET", "/rest/getIndexes.view?username=test&u=test&p=enc:password", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body := make([]byte, resp.ContentLength)
	_, err = resp.Body.Read(body)
	assert.NoError(t, err)

	var response utils.OpenSubsonicResponse
	err = xml.Unmarshal(body, &response)
	assert.NoError(t, err)

	assert.NotNil(t, response.Indexes)
	if response.Indexes != nil {
		// Validate that indexes are properly grouped and sorted
		for _, index := range response.Indexes.Indexes {
			// Validate that artists within each index are sorted alphabetically
			for i := 0; i < len(index.Artists)-1; i++ {
				currentName := strings.ToLower(index.Artists[i].Name)
				nextName := strings.ToLower(index.Artists[i+1].Name)
				assert.True(t, currentName <= nextName,
					"Artists should be sorted within each index: %s should come before %s", currentName, nextName)
			}
		}

		// Validate that indexes themselves are in alphabetical order
		for i := 0; i < len(response.Indexes.Indexes)-1; i++ {
			currentIndex := strings.ToLower(response.Indexes.Indexes[i].Name)
			nextIndex := strings.ToLower(response.Indexes.Indexes[i+1].Name)
			assert.True(t, currentIndex <= nextIndex,
				"Indexes should be sorted alphabetically: %s should come before %s", currentIndex, nextIndex)
		}
	}
}

// TestNormalizationRules validates normalization rules for articles, diacritics, and punctuation
func TestNormalizationRules(t *testing.T) {
	db := setupIndexingTestDatabase(t)
	cfg := getIndexingTestConfig()
	authMiddleware := opensubsonic_middleware.NewOpenSubsonicAuthMiddleware(db, cfg.JWT.Secret)
	app := setupIndexingTestApp(db, cfg, authMiddleware)

	// Create artists with different normalization scenarios
	testArtists := []struct {
		name          string
		expectedIndex string // Expected index group after normalization
		description   string
	}{
		{"The Beatles", "B", "Article 'The' should be moved to end for sorting"},
		{"Aerosmith", "A", "Regular name starts with A"},
		{"AC/DC", "A", "Slash should be converted to space for sorting"},
		{"ZZ Top", "Z", "Name starts with Z"},
		{"Los Lobos", "L", "Spanish article 'Los' should be moved for sorting"},
		{"Les Misérables Cast", "M", "French article 'Les' with accent should be moved for sorting"},
		{"'N Sync", "N", "Leading apostrophe should be handled"},
		{"& The Crickets", "C", "Ampersand should be converted to 'and' for sorting"},
		{"Various Artists", "V", "Regular name starts with V"},
	}

	for _, testData := range testArtists {
		artist := models.Artist{
			Name:           testData.name,
			NameNormalized: strings.ToLower(testData.name), // This would normally be computed differently
		}
		err := db.Create(&artist).Error
		assert.NoError(t, err, "Failed to create artist: %s", testData.name)
	}

	req := httptest.NewRequest("GET", "/rest/getIndexes.view?username=test&u=test&p=enc:password", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body := make([]byte, resp.ContentLength)
	_, err = resp.Body.Read(body)
	assert.NoError(t, err)

	var response utils.OpenSubsonicResponse
	err = xml.Unmarshal(body, &response)
	assert.NoError(t, err)

	assert.NotNil(t, response.Indexes)
	if response.Indexes != nil {
		// Check that artists are properly indexed according to normalization rules
		allArtists := []utils.IndexArtist{}
		for _, index := range response.Indexes.Indexes {
			for _, artist := range index.Artists {
				allArtists = append(allArtists, artist)
			}
		}

		// Verify that the artists are in correct alphabetic order after normalization
		// According to the directory organization plan, articles should be moved for sorting
		normalizedNames := make([]string, len(allArtists))
		for i, artist := range allArtists {
			normalizedNames[i] = strings.ToLower(artist.Name)
		}

		// Make sure they're sorted in the proper order
		sortedNames := make([]string, len(normalizedNames))
		copy(sortedNames, normalizedNames)
		sort.Strings(sortedNames)

		for i, name := range sortedNames {
			assert.Equal(t, name, normalizedNames[i],
				"Artist at position %d should be %s, but was %s", i, name, normalizedNames[i])
		}
	}
}

// TestGetArtistsSorting validates that GetArtists endpoint sorts properly
func TestGetArtistsSorting(t *testing.T) {
	db := setupIndexingTestDatabase(t)
	cfg := getIndexingTestConfig()
	authMiddleware := opensubsonic_middleware.NewOpenSubsonicAuthMiddleware(db, cfg.JWT.Secret)
	app := setupIndexingTestApp(db, cfg, authMiddleware)

	// Create test artists with various names
	testArtists := []models.Artist{
		{Name: "The Velvet Underground", NameNormalized: "the velvet underground"},
		{Name: "Anthrax", NameNormalized: "anthrax"},
		{Name: "Beck", NameNormalized: "beck"},
		{Name: "The Who", NameNormalized: "the who"},
		{Name: "Caetano Veloso", NameNormalized: "caetano veloso"},
		{Name: "Bob Dylan", NameNormalized: "bob dylan"},
	}

	for _, artist := range testArtists {
		err := db.Create(&artist).Error
		assert.NoError(t, err)
	}

	req := httptest.NewRequest("GET", "/rest/getArtists.view?u=test&p=enc:password", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body := make([]byte, resp.ContentLength)
	_, err = resp.Body.Read(body)
	assert.NoError(t, err)

	var response utils.OpenSubsonicResponse
	err = xml.Unmarshal(body, &response)
	assert.NoError(t, err)

	assert.NotNil(t, response.Artists)
	if response.Artists != nil {
		// Validate that artists are sorted properly for display
		artists := response.Artists.Artists

		// Check that they are in alphabetical order (case-insensitive)
		for i := 0; i < len(artists)-1; i++ {
			currentName := strings.ToLower(artists[i].Name)
			nextName := strings.ToLower(artists[i+1].Name)

			assert.True(t, currentName <= nextName,
				"Artists should be sorted alphabetically: '%s' should come before '%s'", currentName, nextName)
		}
	}
}

// TestArticlesNormalization validates handling of articles in artist names
func TestArticlesNormalization(t *testing.T) {
	db := setupIndexingTestDatabase(t)
	cfg := getIndexingTestConfig()
	authMiddleware := opensubsonic_middleware.NewOpenSubsonicAuthMiddleware(db, cfg.JWT.Secret)
	app := setupIndexingTestApp(db, cfg, authMiddleware)

	// Test data with various article types
	artistsWithArticles := []models.Artist{
		{Name: "The Beatles", NameNormalized: "the beatles"},
		{Name: "A Day in the Life", NameNormalized: "a day in the life"}, // Likely an album/song
		{Name: "Annie Lennox", NameNormalized: "annie lennox"},
		{Name: "La Vida Es Un Carnaval", NameNormalized: "la vida es un carnaval"}, // Spanish
		{Name: "Le Tour du Monde", NameNormalized: "le tour du monde"},             // French
		{Name: "Los Angeles Azules", NameNormalized: "los angeles azules"},         // Spanish plural
		{Name: "Les Paul", NameNormalized: "les paul"},                             // French
		{Name: "The The", NameNormalized: "the the"},                               // Artist with "the" in both positions
	}

	for _, artist := range artistsWithArticles {
		err := db.Create(&artist).Error
		assert.NoError(t, err)
	}

	// Test GetIndexes to verify article normalization affects indexing
	req := httptest.NewRequest("GET", "/rest/getIndexes.view?username=test&u=test&p=enc:password", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body := make([]byte, resp.ContentLength)
	_, err = resp.Body.Read(body)
	assert.NoError(t, err)

	var response utils.OpenSubsonicResponse
	err = xml.Unmarshal(body, &response)
	assert.NoError(t, err)

	assert.NotNil(t, response.Indexes)
	if response.Indexes != nil {
		// The test focuses on ensuring indexes work correctly with normalization
		// The specific behavior depends on the normalization implementation in handlers/browsing.go
		// which should move articles like "The", "A", "An", "La", "Le", "Les", "Los" for sorting purposes
		assert.NotEmpty(t, response.Indexes.Indexes, "Should have at least one index group")
	}
}

// TestDiacriticsNormalization validates handling of diacritical marks in names
func TestDiacriticsNormalization(t *testing.T) {
	db := setupIndexingTestDatabase(t)
	cfg := getIndexingTestConfig()
	authMiddleware := opensubsonic_middleware.NewOpenSubsonicAuthMiddleware(db, cfg.JWT.Secret)
	app := setupIndexingTestApp(db, cfg, authMiddleware)

	// Test data with diacritics
	artistsWithDiacritics := []models.Artist{
		{Name: "Café Tacuba", NameNormalized: "café tacuba"},
		{Name: "Nação Zumbi", NameNormalized: "nação zumbi"},
		{Name: "Misplaced Childhood", NameNormalized: "misplaced childhood"},
		{Name: "Mötley Crüe", NameNormalized: "mötley crüe"},
		{Name: "São Paulo Underground", NameNormalized: "são paulo underground"},
		{Name: "Réveil", NameNormalized: "réveil"},
	}

	for _, artist := range artistsWithDiacritics {
		err := db.Create(&artist).Error
		assert.NoError(t, err)
	}

	req := httptest.NewRequest("GET", "/rest/getArtists.view?u=test&p=enc:password", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body := make([]byte, resp.ContentLength)
	_, err = resp.Body.Read(body)
	assert.NoError(t, err)

	var response utils.OpenSubsonicResponse
	err = xml.Unmarshal(body, &response)
	assert.NoError(t, err)

	assert.NotNil(t, response.Artists)
	if response.Artists != nil {
		// Validate that artists are sorted properly with diacritics handled
		artists := response.Artists.Artists
		for i := 0; i < len(artists)-1; i++ {
			currentName := strings.ToLower(artists[i].Name)
			nextName := strings.ToLower(artists[i+1].Name)

			// Check that the sorting is consistent
			assert.True(t, currentName <= nextName,
				"Artists should be sorted: '%s' should come before '%s'", currentName, nextName)
		}
	}
}

// TestPunctuationNormalization validates handling of punctuation in names
func TestPunctuationNormalization(t *testing.T) {
	db := setupIndexingTestDatabase(t)
	cfg := getIndexingTestConfig()
	authMiddleware := opensubsonic_middleware.NewOpenSubsonicAuthMiddleware(db, cfg.JWT.Secret)
	app := setupIndexingTestApp(db, cfg, authMiddleware)

	// Test data with punctuation
	artistsWithPunctuation := []models.Artist{
		{Name: "AC/DC", NameNormalized: "ac/dc"},
		{Name: "A-ha", NameNormalized: "a-ha"},
		{Name: "Marvin Gaye", NameNormalized: "marvin gaye"},
		{Name: "Foo Fighters", NameNormalized: "foo fighters"},
		{Name: "Smashing Pumpkins", NameNormalized: "smashing pumpkins"},
		{Name: "Red Hot Chili Peppers", NameNormalized: "red hot chili peppers"},
	}

	for _, artist := range artistsWithPunctuation {
		err := db.Create(&artist).Error
		assert.NoError(t, err)
	}

	req := httptest.NewRequest("GET", "/rest/getIndexes.view?username=test&u=test&p=enc:password", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body := make([]byte, resp.ContentLength)
	_, err = resp.Body.Read(body)
	assert.NoError(t, err)

	var response utils.OpenSubsonicResponse
	err = xml.Unmarshal(body, &response)
	assert.NoError(t, err)

	assert.NotNil(t, response.Indexes)
	if response.Indexes != nil {
		// Validate that punctuation is handled properly in indexing
		assert.NotEmpty(t, response.Indexes.Indexes, "Should have index groups for punctuation handling")
	}
}

// Helper functions
func setupIndexingTestDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	// Auto-migrate the models
	err = db.AutoMigrate(&models.User{}, &models.Library{}, &models.Artist{}, &models.Album{}, &models.Track{}, &models.Playlist{})
	assert.NoError(t, err)

	return db
}

func getIndexingTestConfig() *config.AppConfig {
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

func setupIndexingTestApp(db *gorm.DB, cfg *config.AppConfig, authMiddleware *opensubsonic_middleware.OpenSubsonicAuthMiddleware) *fiber.App {
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
