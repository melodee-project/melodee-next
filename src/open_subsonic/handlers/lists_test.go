package handlers

import (
	"net/http/httptest"
	"testing"

	"melodee/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestApp(handler *BrowsingHandler) *fiber.App {
	app := fiber.New()

	// Mock user middleware
	app.Use(func(c *fiber.Ctx) error {
		testUser := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
			IsAdmin:  false,
		}
		c.Locals("user", testUser)
		return c.Next()
	})

	return app
}

func getListsTestDB() *gorm.DB {
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

	db.Exec(`CREATE TABLE user_tracks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		track_id INTEGER,
		played_count INTEGER DEFAULT 0,
		last_played_at DATETIME,
		is_starred BOOLEAN DEFAULT 0,
		is_hated BOOLEAN DEFAULT 0,
		starred_at DATETIME,
		rating INTEGER DEFAULT 0,
		created_at DATETIME,
		updated_at DATETIME
	)`)

	db.Exec(`CREATE TABLE user_albums (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		album_id INTEGER,
		is_starred BOOLEAN DEFAULT 0,
		starred_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME
	)`)

	db.Exec(`CREATE TABLE user_artists (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		artist_id INTEGER,
		is_starred BOOLEAN DEFAULT 0,
		starred_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME
	)`)

	return db
}

func TestBrowsingHandler_GetAlbumList(t *testing.T) {
	// Setup
	db := getListsTestDB()
	handler := NewBrowsingHandler(db)
	app := setupTestApp(handler)

	app.Get("/rest/getAlbumList", handler.GetAlbumList)

	// Test
	req := httptest.NewRequest("GET", "/rest/getAlbumList?type=random&size=10", nil)
	resp, err := app.Test(req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestBrowsingHandler_GetRandomSongs(t *testing.T) {
	db := getListsTestDB()
	handler := NewBrowsingHandler(db)
	app := setupTestApp(handler)

	app.Get("/rest/getRandomSongs", handler.GetRandomSongs)

	req := httptest.NewRequest("GET", "/rest/getRandomSongs?size=10", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestBrowsingHandler_GetSongsByGenre(t *testing.T) {
	db := getListsTestDB()
	handler := NewBrowsingHandler(db)
	app := setupTestApp(handler)

	app.Get("/rest/getSongsByGenre", handler.GetSongsByGenre)

	req := httptest.NewRequest("GET", "/rest/getSongsByGenre?genre=Rock", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestBrowsingHandler_GetNowPlaying(t *testing.T) {
	db := getListsTestDB()
	handler := NewBrowsingHandler(db)
	app := setupTestApp(handler)

	app.Get("/rest/getNowPlaying", handler.GetNowPlaying)

	req := httptest.NewRequest("GET", "/rest/getNowPlaying", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestBrowsingHandler_GetTopSongs(t *testing.T) {
	db := getListsTestDB()
	handler := NewBrowsingHandler(db)
	app := setupTestApp(handler)

	app.Get("/rest/getTopSongs", handler.GetTopSongs)

	req := httptest.NewRequest("GET", "/rest/getTopSongs?artist=TestArtist", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestBrowsingHandler_GetSimilarSongs(t *testing.T) {
	db := getListsTestDB()
	handler := NewBrowsingHandler(db)
	app := setupTestApp(handler)

	app.Get("/rest/getSimilarSongs", handler.GetSimilarSongs)

	req := httptest.NewRequest("GET", "/rest/getSimilarSongs?id=1&count=10", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
}
