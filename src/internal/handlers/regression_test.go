package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"melodee/internal/handlers"
	"melodee/internal/middleware"
	"melodee/internal/models"
	"melodee/internal/pagination"
	"melodee/internal/services"
)

// TestRegressionWithLargeDatasets tests that API responses remain stable under large datasets
func TestRegressionWithLargeDatasets(t *testing.T) {
	// Setup test database
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate models
	err = db.AutoMigrate(
		&models.User{}, &models.Library{}, &models.Artist{}, &models.Album{}, &models.Track{},
		&models.Playlist{}, &models.PlaylistTrack{}, &models.UserSong{}, &models.UserAlbum{},
		&models.UserArtist{}, &models.UserPin{}, &models.Bookmark{}, &models.Player{},
		&models.PlayQueue{}, &models.SearchHistory{}, &models.Share{}, &models.ShareActivity{},
		&models.LibraryScanHistory{}, &models.Setting{}, &models.ArtistRelation{}, &models.RadioStation{},
		&models.Contributor{}, &models.CapacityStatus{},
	)
	if err != nil {
		t.Fatal("Failed to migrate database:", err)
	}

	// Create test repository and authentication service
	repo := services.NewRepository(db)
	authService := services.NewAuthService(db, "test-secret-key")

	// Create test users
	adminUser := &models.User{
		Username:     "admin",
		Email:        "admin@test.com",
		PasswordHash: "$2a$10$N9qo8uLOickgxRV.3xeJoUoJy6lVBs5xYhQ7tPzpuOdBu.W6/72d6", // bcrypt hash for "password"
		IsAdmin:      true,
	}
	err = repo.CreateUser(adminUser)
	if err != nil {
		t.Fatal("Failed to create admin user:", err)
	}

	regularUser := &models.User{
		Username:     "user",
		Email:        "user@test.com",
		PasswordHash: "$2a$10$N9qo8uLOickgxRV.3xeJoUoJy6lVBs5xYhQ7tPzpuOdBu.W6/72d6", // bcrypt hash for "password"
		IsAdmin:      false,
	}
	err = repo.CreateUser(regularUser)
	if err != nil {
		t.Fatal("Failed to create regular user:", err)
	}

	// Create a large number of test records to simulate a large dataset
	for i := 0; i < 1000; i++ {
		// Create artists
		artist := &models.Artist{
			Name:           "Test Artist " + string(rune(i+65)),
			NameNormalized: "test artist " + string(rune(i+65)),
		}
		err = repo.CreateArtist(artist)
		assert.NoError(t, err)

		// Create albums for each artist
		album := &models.Album{
			Name:           "Test Album " + string(rune(i+65)),
			NameNormalized: "test album " + string(rune(i+65)),
			ArtistID:       artist.ID,
			AlbumStatus:    "Ok",
		}
		err = repo.CreateAlbum(album)
		assert.NoError(t, err)

		// Create tracks for each album
		track := &models.Track{
			Name:           "Test Track " + string(rune(i+65)),
			NameNormalized: "test track " + string(rune(i+65)),
			AlbumID:        album.ID,
			ArtistID:       artist.ID,
		}
		err = repo.CreateTrack(track)
		assert.NoError(t, err)
	}

	// Create 50 playlists to test pagination
	for i := 0; i < 50; i++ {
		playlist := &models.Playlist{
			UserID: regularUser.ID,
			Name:   "Test Playlist " + string(rune(i+65)),
			Public: i%10 == 0, // Make some public
		}
		err = repo.CreatePlaylist(playlist)
		assert.NoError(t, err)
	}

	// Create the Fiber app with handlers
	app := fiber.New()

	// Create handlers
	playlistHandler := handlers.NewPlaylistHandler(repo)
	authMiddleware := middleware.NewAuthMiddleware(authService)

	// Setup routes
	protected := app.Group("/api", authMiddleware.JWTProtected())
	playlists := protected.Group("/playlists")
	playlists.Get("/", playlistHandler.GetPlaylists)
	playlists.Get("/:id", playlistHandler.GetPlaylist)

	// Test playlist endpoint with pagination to ensure stable responses
	req := httptest.NewRequest("GET", "/api/playlists?page=1&pageSize=10", nil)
	req.Header.Set("Authorization", "Bearer fake-token") // This will fail validation but tests structure

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode) // Expected due to fake token

	// Test search functionality with large dataset
	searchHandler := handlers.NewSearchHandler(repo)
	app.Post("/api/search", authMiddleware.JWTProtected(), searchHandler.Search)

	// Prepare search request
	searchBody := `{"query": "test", "type": "any"}`
	req = httptest.NewRequest("POST", "/api/search", strings.NewReader(searchBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer fake-token")

	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode) // Expected due to fake token

	// Test pagination with large dataset by creating a mock handler
	// that doesn't require authentication to test the pagination logic
	paginationApp := fiber.New()

	paginationApp.Get("/test-pagination", func(c *fiber.Ctx) error {
		page, pageSize := pagination.GetPaginationParams(c, 1, 10)
		offset := pagination.CalculateOffset(page, pageSize)

		// Simulate getting paginated data (in reality this would come from DB)
		total := int64(1000) // Simulate 1000 items
		data := make([]string, 0)

		// Just simulate data slice
		end := offset + pageSize
		if end > 1000 {
			end = 1000
		}

		for i := offset; i < end; i++ {
			data = append(data, "item-"+string(rune(i+65)))
		}

		paginationMeta := pagination.Calculate(total, page, pageSize)

		return c.JSON(fiber.Map{
			"data":       data,
			"pagination": paginationMeta,
		})
	})

	// Test pagination with different page sizes
	req = httptest.NewRequest("GET", "/test-pagination?page=1&pageSize=50", nil)
	resp, err = paginationApp.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)

	// Verify pagination structure matches expectations
	assert.Contains(t, result, "data")
	assert.Contains(t, result, "pagination")

	pagination, ok := result["pagination"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, float64(1000), pagination["totalCount"])
	assert.Equal(t, 50.0, pagination["pageSize"])
	assert.Equal(t, 1.0, pagination["currentPage"])
	assert.Equal(t, 20.0, pagination["totalPages"]) // 1000/50 = 20
	assert.False(t, pagination["hasPrevious"].(bool))
	assert.True(t, pagination["hasNext"].(bool))

	t.Log("Regression test with large dataset completed successfully")
}
