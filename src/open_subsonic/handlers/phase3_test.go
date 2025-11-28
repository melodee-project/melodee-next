package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"melodee/internal/models"
)

func setupPhase3TestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Manual SQLite Schema Setup
	// Users
	db.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT,
		password_hash TEXT,
		email TEXT,
		is_admin BOOLEAN DEFAULT 0,
		api_key TEXT,
		created_at DATETIME,
		last_login_at DATETIME,
		failed_login_attempts INTEGER DEFAULT 0,
		locked_until DATETIME,
		password_reset_token TEXT,
		password_reset_expiry DATETIME
	)`)

	// Artists
	db.Exec(`CREATE TABLE artists (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		name_normalized TEXT,
		api_key TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		album_count_cached INTEGER DEFAULT 0,
		track_count_cached INTEGER DEFAULT 0,
		is_locked BOOLEAN DEFAULT 0,
		directory_code TEXT,
		sort_name TEXT,
		alternate_names TEXT,
		duration_cached INTEGER DEFAULT 0,
		last_scanned_at DATETIME,
		tags TEXT,
		music_brainz_id TEXT,
		spotify_id TEXT,
		last_fm_id TEXT,
		discogs_id TEXT,
		i_tunes_id TEXT,
		amg_id TEXT,
		wikidata_id TEXT,
		sort_order INTEGER DEFAULT 0
	)`)

	// Albums
	db.Exec(`CREATE TABLE albums (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		name_normalized TEXT,
		artist_id INTEGER,
		created_at DATETIME,
		updated_at DATETIME,
		track_count_cached INTEGER DEFAULT 0,
		duration_cached INTEGER DEFAULT 0,
		genre TEXT,
		year INTEGER,
		cover_art_path TEXT,
		api_key TEXT,
		release_date DATETIME,
		genres TEXT,
		is_locked BOOLEAN DEFAULT 0,
		alternate_names TEXT,
		original_release_date DATETIME,
		album_type TEXT,
		directory TEXT,
		sort_name TEXT,
		sort_order INTEGER DEFAULT 0,
		image_count INTEGER DEFAULT 0,
		comment TEXT,
		description TEXT,
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
		tags TEXT
	)`)

	// Tracks
	db.Exec(`CREATE TABLE tracks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		name_normalized TEXT,
		album_id INTEGER,
		artist_id INTEGER,
		track_number INTEGER,
		disc_number INTEGER,
		duration INTEGER,
		bit_rate INTEGER,
		file_path TEXT,
		file_size INTEGER,
		created_at DATETIME,
		updated_at DATETIME,
		content_type TEXT,
		suffix TEXT,
		api_key TEXT,
		sort_order INTEGER DEFAULT 0,
		relative_path TEXT,
		directory TEXT,
		sort_name TEXT,
		bit_depth INTEGER,
		sample_rate INTEGER,
		channels INTEGER,
		tags TEXT,
		file_name TEXT,
		crc_hash TEXT
	)`)

	// Bookmarks
	db.Exec(`CREATE TABLE bookmarks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		track_id INTEGER,
		position INTEGER,
		comment TEXT,
		created_at DATETIME,
		updated_at DATETIME
	)`)

	// PlayQueue
	db.Exec(`CREATE TABLE play_queues (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		track_id INTEGER,
		track_api_key TEXT,
		is_current_track BOOLEAN,
		changed_by TEXT,
		position REAL,
		play_queue_id INTEGER,
		created_at DATETIME,
		updated_at DATETIME
	)`)

	return db
}

func TestBookmarkHandler_CreateAndGet(t *testing.T) {
	db := setupPhase3TestDB(t)
	app := fiber.New()

	// Setup data
	user := models.User{Username: "testuser", PasswordHash: "hash"}
	db.Create(&user)

	artist := models.Artist{Name: "Test Artist"}
	db.Create(&artist)

	album := models.Album{Name: "Test Album", ArtistID: artist.ID}
	db.Create(&album)

	track := models.Track{Name: "Test Track", AlbumID: album.ID, ArtistID: artist.ID, Duration: 300000}
	db.Create(&track)

	// Setup handler
	handler := NewBookmarkHandler(db)

	// Mock auth middleware
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user", &user)
		return c.Next()
	})

	app.Get("/createBookmark", handler.CreateBookmark)
	app.Get("/getBookmarks", handler.GetBookmarks)
	app.Get("/deleteBookmark", handler.DeleteBookmark)

	// Test Create
	req := httptest.NewRequest("GET", "/createBookmark?id=1&position=15000&comment=test", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Test Get
	req = httptest.NewRequest("GET", "/getBookmarks?f=json", nil)
	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)

	subResp := response["subsonic-response"].(map[string]interface{})
	bookmarks := subResp["bookmarks"].(map[string]interface{})
	bookmarkList := bookmarks["bookmark"].([]interface{})

	assert.Len(t, bookmarkList, 1)
	bookmark := bookmarkList[0].(map[string]interface{})
	assert.Equal(t, float64(15000), bookmark["position"])
	assert.Equal(t, "test", bookmark["comment"])

	// Test Delete
	req = httptest.NewRequest("GET", "/deleteBookmark?id=1", nil)
	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Verify deletion
	req = httptest.NewRequest("GET", "/getBookmarks?f=json", nil)
	resp, err = app.Test(req)

	json.NewDecoder(resp.Body).Decode(&response)
	subResp = response["subsonic-response"].(map[string]interface{})

	// Check if bookmarks key exists and is empty or null
	if b, ok := subResp["bookmarks"]; ok {
		if bMap, ok := b.(map[string]interface{}); ok {
			if list, ok := bMap["bookmark"]; ok {
				assert.Len(t, list, 0)
			}
		}
	}
}

func TestPlayQueueHandler_SaveAndGet(t *testing.T) {
	db := setupPhase3TestDB(t)
	app := fiber.New()

	// Setup data
	user := models.User{Username: "testuser", PasswordHash: "hash"}
	db.Create(&user)

	artist := models.Artist{Name: "Test Artist"}
	db.Create(&artist)

	album := models.Album{Name: "Test Album", ArtistID: artist.ID}
	db.Create(&album)

	track1 := models.Track{Name: "Track 1", AlbumID: album.ID, ArtistID: artist.ID}
	db.Create(&track1)
	track2 := models.Track{Name: "Track 2", AlbumID: album.ID, ArtistID: artist.ID}
	db.Create(&track2)

	// Setup handler
	handler := NewPlayQueueHandler(db)

	// Mock auth middleware
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user", &user)
		return c.Next()
	})

	app.Get("/savePlayQueue", handler.SavePlayQueue)
	app.Get("/getPlayQueue", handler.GetPlayQueue)

	// Test Save
	// ?id=1&id=2&current=1&position=5000
	req := httptest.NewRequest("GET", "/savePlayQueue?id=1&id=2&current=1&position=5000", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Test Get
	req = httptest.NewRequest("GET", "/getPlayQueue?f=json", nil)
	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)

	subResp := response["subsonic-response"].(map[string]interface{})
	playQueue := subResp["playQueue"].(map[string]interface{})
	entries := playQueue["entry"].([]interface{})

	assert.Len(t, entries, 2)
	assert.Equal(t, float64(1), playQueue["current"])
	assert.Equal(t, float64(5000), playQueue["position"])
}
