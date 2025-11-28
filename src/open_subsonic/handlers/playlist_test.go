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

func getPlaylistTestDB() *gorm.DB {
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

	db.Exec(`CREATE TABLE playlists (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		api_key TEXT,
		user_id INTEGER,
		name TEXT,
		comment TEXT,
		public BOOLEAN DEFAULT 0,
		created_at DATETIME,
		changed_at DATETIME,
		duration INTEGER DEFAULT 0,
		track_count INTEGER DEFAULT 0,
		cover_art_id INTEGER
	)`)

	db.Exec(`CREATE TABLE playlist_tracks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		playlist_id INTEGER,
		track_id INTEGER,
		position INTEGER,
		created_at DATETIME
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

func TestPlaylistHandler_GetPlaylists(t *testing.T) {
	db := getPlaylistTestDB()
	playlistHandler := NewPlaylistHandler(db)

	app := fiber.New()
	app.Get("/rest/getPlaylists", func(c *fiber.Ctx) error {
		testUser := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
			IsAdmin:  false,
		}
		c.Locals("user", testUser)
		return playlistHandler.GetPlaylists(c)
	})

	// Seed data
	user := models.User{Username: "testuser"}
	db.Create(&user)
	playlist := models.Playlist{Name: "Test Playlist", UserID: int64(user.ID)}
	db.Create(&playlist)

	req := httptest.NewRequest("GET", "/rest/getPlaylists", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestPlaylistHandler_GetPlaylist(t *testing.T) {
	db := getPlaylistTestDB()
	playlistHandler := NewPlaylistHandler(db)

	app := fiber.New()
	app.Get("/rest/getPlaylist", func(c *fiber.Ctx) error {
		testUser := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
			IsAdmin:  false,
		}
		c.Locals("user", testUser)
		return playlistHandler.GetPlaylist(c)
	})

	// Seed data
	user := models.User{Username: "testuser"}
	db.Create(&user)
	playlist := models.Playlist{Name: "Test Playlist", UserID: int64(user.ID)}
	db.Create(&playlist)

	req := httptest.NewRequest("GET", "/rest/getPlaylist?id=1", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestPlaylistHandler_CreatePlaylist(t *testing.T) {
	db := getPlaylistTestDB()
	playlistHandler := NewPlaylistHandler(db)

	app := fiber.New()
	app.Get("/rest/createPlaylist", func(c *fiber.Ctx) error {
		testUser := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
			IsAdmin:  false,
		}
		c.Locals("user", testUser)
		return playlistHandler.CreatePlaylist(c)
	})

	// Seed user for ownership check
	user := models.User{Username: "testuser"}
	db.Create(&user)

	req := httptest.NewRequest("GET", "/rest/createPlaylist?name=NewPlaylist", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestPlaylistHandler_UpdatePlaylist(t *testing.T) {
	db := getPlaylistTestDB()
	playlistHandler := NewPlaylistHandler(db)

	app := fiber.New()
	app.Get("/rest/updatePlaylist", func(c *fiber.Ctx) error {
		testUser := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
			IsAdmin:  false,
		}
		c.Locals("user", testUser)
		return playlistHandler.UpdatePlaylist(c)
	})

	// Seed data
	user := models.User{Username: "testuser"}
	db.Create(&user)
	playlist := models.Playlist{Name: "Test Playlist", UserID: int64(user.ID)}
	db.Create(&playlist)

	req := httptest.NewRequest("GET", "/rest/updatePlaylist?playlistId=1&name=UpdatedName", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestPlaylistHandler_DeletePlaylist(t *testing.T) {
	db := getPlaylistTestDB()
	playlistHandler := NewPlaylistHandler(db)

	app := fiber.New()
	app.Get("/rest/deletePlaylist", func(c *fiber.Ctx) error {
		testUser := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
			IsAdmin:  false,
		}
		c.Locals("user", testUser)
		return playlistHandler.DeletePlaylist(c)
	})

	// Seed data
	user := models.User{Username: "testuser"}
	db.Create(&user)
	playlist := models.Playlist{Name: "Test Playlist", UserID: int64(user.ID)}
	db.Create(&playlist)

	req := httptest.NewRequest("GET", "/rest/deletePlaylist?id=1", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
