package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"melodee/internal/config"
	"melodee/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func getMediaTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		return &gorm.DB{}
	}

	// Manually create tables for SQLite to avoid Postgres-specific syntax issues in AutoMigrate
	db.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT,
		email TEXT,
		password_hash TEXT,
		is_admin BOOLEAN,
		failed_login_attempts INTEGER DEFAULT 0,
		locked_until DATETIME,
		password_reset_token TEXT,
		password_reset_expiry DATETIME,
		created_at DATETIME,
		last_login_at DATETIME,
		api_key TEXT
	)`)

	db.Exec(`CREATE TABLE artists (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		name_normalized TEXT,
		album_count_cached INTEGER DEFAULT 0,
		is_locked BOOLEAN DEFAULT 0,
		directory_code TEXT,
		sort_name TEXT,
		alternate_names TEXT,
		track_count_cached INTEGER DEFAULT 0,
		duration_cached INTEGER DEFAULT 0,
		created_at DATETIME,
		last_scanned_at DATETIME,
		tags TEXT,
		music_brainz_id TEXT,
		spotify_id TEXT,
		last_fm_id TEXT,
		discogs_id TEXT,
		i_tunes_id TEXT,
		amg_id TEXT,
		wikidata_id TEXT,
		sort_order INTEGER DEFAULT 0,
		api_key TEXT
	)`)

	db.Exec(`CREATE TABLE albums (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		artist_id INTEGER,
		track_count_cached INTEGER DEFAULT 0,
		duration_cached INTEGER DEFAULT 0,
		created_at DATETIME,
		release_date DATETIME,
		original_release_date DATETIME,
		is_locked BOOLEAN DEFAULT 0,
		name_normalized TEXT,
		alternate_names TEXT,
		tags TEXT,
		album_type TEXT,
		directory TEXT,
		sort_name TEXT,
		sort_order INTEGER DEFAULT 0,
		image_count INTEGER DEFAULT 0,
		comment TEXT,
		description TEXT,
		genres TEXT,
		moods TEXT,
		notes TEXT,
		deezer_id TEXT,
		music_brainz_id TEXT,
		spotify_id TEXT,
		last_fm_id TEXT,
		discogs_id TEXT,
		i_tunes_id TEXT,
		amg_id TEXT,
		wikidata_id TEXT,
		is_compilation BOOLEAN DEFAULT 0,
		api_key TEXT
	)`)

	db.Exec(`CREATE TABLE tracks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		album_id INTEGER,
		artist_id INTEGER,
		duration INTEGER,
		bit_rate INTEGER,
		sort_order INTEGER,
		created_at DATETIME,
		tags TEXT,
		file_name TEXT,
		relative_path TEXT,
		name_normalized TEXT,
		sort_name TEXT,
		bit_depth INTEGER,
		sample_rate INTEGER,
		channels INTEGER,
		directory TEXT,
		crc_hash TEXT,
		api_key TEXT
	)`)

	return db
}

func TestMediaHandler_Stream(t *testing.T) {
	// Create a minimal setup for testing
	db := getMediaTestDB()
	cfg := &config.AppConfig{}

	mediaHandler := NewMediaHandler(db, cfg, nil)

	// Create Fiber app for testing
	app := fiber.New()
	app.Get("/rest/stream", func(c *fiber.Ctx) error {
		// Simulate authenticated user context
		testUser := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
			IsAdmin:  false,
		}
		c.Locals("user", testUser)
		return mediaHandler.Stream(c)
	})

	// Seed data
	artist := models.Artist{Name: "Test Artist"}
	db.Create(&artist)
	album := models.Album{Name: "Test Album", ArtistID: int64(artist.ID)}
	db.Create(&album)
	track := models.Track{
		Name:         "Test Song",
		AlbumID:      int64(album.ID),
		ArtistID:     int64(artist.ID),
		RelativePath: "test/song.mp3",
	}
	db.Create(&track)

	// Test basic request with id parameter
	req := httptest.NewRequest("GET", "/rest/stream?id=1", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Should return 200 OK or 404 (not found) or similar (not authentication error)
	// Since file doesn't exist on disk, it might return error, but not panic
	assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
	assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
}

func TestMediaHandler_GetCoverArt(t *testing.T) {
	// Create a minimal setup for testing
	db := getMediaTestDB()
	cfg := &config.AppConfig{}
	mediaHandler := NewMediaHandler(db, cfg, nil)

	// Create Fiber app for testing
	app := fiber.New()
	app.Get("/rest/getCoverArt", func(c *fiber.Ctx) error {
		// Simulate authenticated user context
		testUser := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
			IsAdmin:  false,
		}
		c.Locals("user", testUser)
		return mediaHandler.GetCoverArt(c)
	})

	// Seed data
	artist := models.Artist{Name: "Test Artist"}
	db.Create(&artist)
	album := models.Album{Name: "Test Album", ArtistID: int64(artist.ID)}
	db.Create(&album)

	// Test basic request with id parameter
	req := httptest.NewRequest("GET", "/rest/getCoverArt?id=al-1", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Should return 200 OK or 404 (not found) or similar (not authentication error)
	assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
	assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
}

func TestMediaHandler_GetAvatar(t *testing.T) {
	// Create a minimal setup for testing
	db := getMediaTestDB()
	cfg := &config.AppConfig{}
	mediaHandler := NewMediaHandler(db, cfg, nil)

	// Create Fiber app for testing
	app := fiber.New()
	app.Get("/rest/getAvatar", func(c *fiber.Ctx) error {
		// Simulate authenticated user context
		testUser := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
			IsAdmin:  false,
		}
		c.Locals("user", testUser)
		return mediaHandler.GetAvatar(c)
	})

	// Test basic request
	req := httptest.NewRequest("GET", "/rest/getAvatar?username=testuser", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Should return 200 OK or 404 (not found) or similar (not authentication error)
	assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
	assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
}

func TestMediaHandler_Download(t *testing.T) {
// Create a minimal setup for testing
db := getMediaTestDB()
cfg := &config.AppConfig{}

mediaHandler := NewMediaHandler(db, cfg, nil)

// Create Fiber app for testing
app := fiber.New()
app.Get("/rest/download", func(c *fiber.Ctx) error {
// Simulate authenticated user context
testUser := &models.User{
ID:       1,
Username: "testuser",
Email:    "test@example.com",
IsAdmin:  false,
}
c.Locals("user", testUser)
return mediaHandler.Download(c)
})

// Seed data
artist := models.Artist{Name: "Test Artist"}
db.Create(&artist)
album := models.Album{Name: "Test Album", ArtistID: int64(artist.ID)}
db.Create(&album)
track := models.Track{
Name:         "Test Song",
AlbumID:      int64(album.ID),
ArtistID:     int64(artist.ID),
RelativePath: "test/song.mp3",
}
db.Create(&track)

// Test basic request with id parameter
req := httptest.NewRequest("GET", "/rest/download?id=1", nil)
resp, err := app.Test(req)
assert.NoError(t, err)

// Should return 200 OK or 404 (not found) or similar (not authentication error)
assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
}
