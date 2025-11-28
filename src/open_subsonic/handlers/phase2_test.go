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

func setupPhase2TestApp(browsingHandler *BrowsingHandler, userHandler *UserHandler) *fiber.App {
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

func getPhase2TestDB() *gorm.DB {
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

func TestUserHandler_StarUnstar(t *testing.T) {
	db := getPhase2TestDB()
	userHandler := NewUserHandler(db)
	app := setupPhase2TestApp(nil, userHandler)

	app.Post("/rest/star", userHandler.Star)
	app.Post("/rest/unstar", userHandler.Unstar)
	app.Get("/rest/getStarred", userHandler.GetStarred)

	// Test Star
	req := httptest.NewRequest("POST", "/rest/star?id=1", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Test GetStarred
	req = httptest.NewRequest("GET", "/rest/getStarred", nil)
	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Test Unstar
	req = httptest.NewRequest("POST", "/rest/unstar?id=1", nil)
	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestUserHandler_SetRating(t *testing.T) {
	db := getPhase2TestDB()
	userHandler := NewUserHandler(db)
	app := setupPhase2TestApp(nil, userHandler)

	app.Post("/rest/setRating", userHandler.SetRating)

	req := httptest.NewRequest("POST", "/rest/setRating?id=1&rating=5", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestUserHandler_Scrobble(t *testing.T) {
	db := getPhase2TestDB()
	userHandler := NewUserHandler(db)
	app := setupPhase2TestApp(nil, userHandler)

	app.Post("/rest/scrobble", userHandler.Scrobble)

	req := httptest.NewRequest("POST", "/rest/scrobble?id=1", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestBrowsingHandler_GetLyrics(t *testing.T) {
	db := getPhase2TestDB()
	browsingHandler := NewBrowsingHandler(db)
	app := setupPhase2TestApp(browsingHandler, nil)

	app.Get("/rest/getLyrics", browsingHandler.GetLyrics)

	// Seed data
	artist := models.Artist{Name: "Test Artist"}
	db.Create(&artist)
	track := models.Track{Name: "Test Song", ArtistID: int64(artist.ID)}
	db.Create(&track)

	req := httptest.NewRequest("GET", "/rest/getLyrics?artist=Test%20Artist&title=Test%20Song", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestBrowsingHandler_GetArtistInfo(t *testing.T) {
	db := getPhase2TestDB()
	browsingHandler := NewBrowsingHandler(db)
	app := setupPhase2TestApp(browsingHandler, nil)

	app.Get("/rest/getArtistInfo", browsingHandler.GetArtistInfo)
	app.Get("/rest/getArtistInfo2", browsingHandler.GetArtistInfo2)

	// Seed data
	artist := models.Artist{Name: "Test Artist"}
	db.Create(&artist)

	// Test V1
	req := httptest.NewRequest("GET", "/rest/getArtistInfo?id=1", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Test V2
	req = httptest.NewRequest("GET", "/rest/getArtistInfo2?id=1", nil)
	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestBrowsingHandler_GetAlbumInfo(t *testing.T) {
	db := getPhase2TestDB()
	browsingHandler := NewBrowsingHandler(db)
	app := setupPhase2TestApp(browsingHandler, nil)

	app.Get("/rest/getAlbumInfo", browsingHandler.GetAlbumInfo)
	app.Get("/rest/getAlbumInfo2", browsingHandler.GetAlbumInfo2)

	// Seed data
	artist := models.Artist{Name: "Test Artist"}
	db.Create(&artist)
	album := models.Album{Name: "Test Album", ArtistID: int64(artist.ID)}
	db.Create(&album)

	// Test V1
	req := httptest.NewRequest("GET", "/rest/getAlbumInfo?id=1", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Test V2
	req = httptest.NewRequest("GET", "/rest/getAlbumInfo2?id=1", nil)
	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestUserHandler_GetUser(t *testing.T) {
	db := getPhase2TestDB()
	userHandler := NewUserHandler(db)
	app := setupPhase2TestApp(nil, userHandler)

	app.Get("/rest/getUser", userHandler.GetUser)

	// Seed user
	user := models.User{Username: "testuser", Email: "test@example.com"}
	db.Create(&user)

	req := httptest.NewRequest("GET", "/rest/getUser?username=testuser", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestBrowsingHandler_GetLyricsBySongId(t *testing.T) {
	db := getPhase2TestDB()
	browsingHandler := NewBrowsingHandler(db)
	app := setupPhase2TestApp(browsingHandler, nil)

	app.Get("/rest/getLyricsBySongId", browsingHandler.GetLyricsBySongId)

	// Seed data
	artist := models.Artist{Name: "Test Artist"}
	db.Create(&artist)
	track := models.Track{Name: "Test Song", ArtistID: int64(artist.ID)}
	db.Create(&track)

	req := httptest.NewRequest("GET", "/rest/getLyricsBySongId?id=1", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}
