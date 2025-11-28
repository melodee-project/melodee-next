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

func setupPhase4TestDB(t *testing.T) *gorm.DB {
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

	// Podcast Channels
	db.Exec(`CREATE TABLE podcast_channels (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		api_key TEXT,
		url TEXT,
		title TEXT,
		description TEXT,
		image_url TEXT,
		status TEXT,
		error_message TEXT,
		created_at DATETIME,
		updated_at DATETIME
	)`)

	// Podcast Episodes
	db.Exec(`CREATE TABLE podcast_episodes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		api_key TEXT,
		channel_id INTEGER,
		title TEXT,
		description TEXT,
		publish_date DATETIME,
		duration INTEGER,
		status TEXT,
		error_message TEXT,
		file_name TEXT,
		file_size INTEGER,
		content_type TEXT,
		created_at DATETIME,
		updated_at DATETIME
	)`)

	// Radio Stations
	db.Exec(`CREATE TABLE radio_stations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		api_key TEXT,
		name TEXT,
		stream_url TEXT,
		home_page_url TEXT,
		created_by_user_id INTEGER,
		track_count INTEGER,
		is_enabled BOOLEAN DEFAULT 1,
		created_at DATETIME
	)`)

	return db
}

func TestPodcastHandler_CreateAndGet(t *testing.T) {
	db := setupPhase4TestDB(t)
	app := fiber.New()

	// Setup data
	user := models.User{Username: "testuser", PasswordHash: "hash"}
	db.Create(&user)

	// Setup handler
	handler := NewPodcastHandler(db)

	// Mock auth middleware
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user", &user)
		return c.Next()
	})

	app.Get("/createPodcastChannel", handler.CreatePodcastChannel)
	app.Get("/getPodcasts", handler.GetPodcasts)
	app.Get("/deletePodcastChannel", handler.DeletePodcastChannel)

	// Test Create
	req := httptest.NewRequest("GET", "/createPodcastChannel?url=http://example.com/feed.xml", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Test Get
	req = httptest.NewRequest("GET", "/getPodcasts?f=json", nil)
	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)

	subResp := response["subsonic-response"].(map[string]interface{})
	podcasts := subResp["podcasts"].(map[string]interface{})
	channels := podcasts["channel"].([]interface{})

	assert.Len(t, channels, 1)
	channel := channels[0].(map[string]interface{})
	assert.Equal(t, "http://example.com/feed.xml", channel["url"])
	assert.Equal(t, "new", channel["status"])

	// Test Delete
	// Get ID first
	id := channel["id"].(string)
	req = httptest.NewRequest("GET", "/deletePodcastChannel?id="+id, nil)
	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Verify deletion
	req = httptest.NewRequest("GET", "/getPodcasts?f=json", nil)
	resp, err = app.Test(req)

	json.NewDecoder(resp.Body).Decode(&response)
	subResp = response["subsonic-response"].(map[string]interface{})

	// Check if podcasts key exists and is empty or null
	if p, ok := subResp["podcasts"]; ok {
		if pMap, ok := p.(map[string]interface{}); ok {
			if list, ok := pMap["channel"]; ok {
				assert.Len(t, list, 0)
			}
		}
	}
}

func TestInternetRadioHandler_CreateAndGet(t *testing.T) {
	db := setupPhase4TestDB(t)
	app := fiber.New()

	// Setup data
	user := models.User{Username: "testuser", PasswordHash: "hash"}
	db.Create(&user)

	// Setup handler
	handler := NewInternetRadioHandler(db)

	// Mock auth middleware
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user", &user)
		return c.Next()
	})

	app.Get("/createInternetRadioStation", handler.CreateInternetRadioStation)
	app.Get("/getInternetRadioStations", handler.GetInternetRadioStations)
	app.Get("/deleteInternetRadioStation", handler.DeleteInternetRadioStation)

	// Test Create
	req := httptest.NewRequest("GET", "/createInternetRadioStation?streamUrl=http://stream.com&name=TestStation", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Test Get
	req = httptest.NewRequest("GET", "/getInternetRadioStations?f=json", nil)
	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)

	subResp := response["subsonic-response"].(map[string]interface{})
	stations := subResp["internetRadioStations"].(map[string]interface{})
	stationList := stations["internetRadioStation"].([]interface{})

	assert.Len(t, stationList, 1)
	station := stationList[0].(map[string]interface{})
	assert.Equal(t, "http://stream.com", station["streamUrl"])
	assert.Equal(t, "TestStation", station["name"])

	// Test Delete
	id := station["id"].(string)
	req = httptest.NewRequest("GET", "/deleteInternetRadioStation?id="+id, nil)
	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Verify deletion
	req = httptest.NewRequest("GET", "/getInternetRadioStations?f=json", nil)
	resp, err = app.Test(req)

	json.NewDecoder(resp.Body).Decode(&response)
	subResp = response["subsonic-response"].(map[string]interface{})

	if s, ok := subResp["internetRadioStations"]; ok {
		if sMap, ok := s.(map[string]interface{}); ok {
			if list, ok := sMap["internetRadioStation"]; ok {
				assert.Len(t, list, 0)
			}
		}
	}
}
