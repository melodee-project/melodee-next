package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"melodee/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBrowsingHandler_GetArtists(t *testing.T) {
	// Create a minimal setup for testing
	db := getTestDB()

	browsingHandler := NewBrowsingHandler(db)

	// Create Fiber app for testing
	app := fiber.New()
	app.Get("/rest/getArtists", func(c *fiber.Ctx) error {
		// Simulate authenticated user context
		testUser := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
			IsAdmin:  false,
		}
		c.Locals("user", testUser)
		return browsingHandler.GetArtists(c)
	})

	// Test basic request
	req := httptest.NewRequest("GET", "/rest/getArtists", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Should return 200 OK or similar (not authentication error)
	assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
	assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
}

func TestBrowsingHandler_GetMusicFolders(t *testing.T) {
	// Create a minimal setup for testing
	db := getTestDB()

	browsingHandler := NewBrowsingHandler(db)

	// Create Fiber app for testing
	app := fiber.New()
	app.Get("/rest/getMusicFolders", func(c *fiber.Ctx) error {
		// Simulate authenticated user context
		testUser := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
			IsAdmin:  false,
		}
		c.Locals("user", testUser)
		return browsingHandler.GetMusicFolders(c)
	})

	// Test basic request
	req := httptest.NewRequest("GET", "/rest/getMusicFolders", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Should return 200 OK or similar (not authentication error)
	assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
	assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
}

func TestBrowsingHandler_GetArtist(t *testing.T) {
	// Create a minimal setup for testing
	db := getTestDB()

	browsingHandler := NewBrowsingHandler(db)

	// Create Fiber app for testing
	app := fiber.New()
	app.Get("/rest/getArtist", func(c *fiber.Ctx) error {
		// Simulate authenticated user context
		testUser := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
			IsAdmin:  false,
		}
		c.Locals("user", testUser)
		return browsingHandler.GetArtist(c)
	})

	// Test basic request with artist parameter
	req := httptest.NewRequest("GET", "/rest/getArtist?id=1", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Should return 200 OK or 404 (not found) or similar (not authentication error)
	assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
	assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
}

func getTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		return &gorm.DB{}
	}

	// Manually create tables for SQLite
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

	db.Exec(`CREATE TABLE libraries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		path TEXT,
		type TEXT,
		is_locked BOOLEAN DEFAULT 0,
		created_at DATETIME,
		track_count INTEGER DEFAULT 0,
		album_count INTEGER DEFAULT 0,
		duration INTEGER DEFAULT 0
	)`)

	return db
}
