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

func setupPhase5TestDB(t *testing.T) *gorm.DB {
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

	// Shares
	db.Exec(`CREATE TABLE shares (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		name TEXT,
		description TEXT,
		expires_at DATETIME,
		max_streaming_minutes INTEGER,
		max_streaming_count INTEGER,
		allow_streaming BOOLEAN DEFAULT 1,
		allow_download BOOLEAN DEFAULT 0,
		created_at DATETIME,
		updated_at DATETIME
	)`)

	return db
}

func TestSharesHandler_CreateAndGet(t *testing.T) {
	db := setupPhase5TestDB(t)
	app := fiber.New()

	// Setup data
	user := models.User{Username: "testuser", PasswordHash: "hash"}
	db.Create(&user)

	// Setup handler
	handler := NewSharesHandler(db)

	// Mock auth middleware
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user", &user)
		return c.Next()
	})

	app.Get("/createShare", handler.CreateShare)
	app.Get("/getShares", handler.GetShares)
	app.Get("/deleteShare", handler.DeleteShare)

	// Test Create
	req := httptest.NewRequest("GET", "/createShare?id=1&description=TestShare", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Test Get
	req = httptest.NewRequest("GET", "/getShares?f=json", nil)
	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)

	subResp := response["subsonic-response"].(map[string]interface{})
	shares := subResp["shares"].(map[string]interface{})
	shareList := shares["share"].([]interface{})

	assert.Len(t, shareList, 1)
	share := shareList[0].(map[string]interface{})
	assert.Equal(t, "TestShare", share["description"])
	assert.Equal(t, "testuser", share["username"])

	// Test Delete
	id := share["id"].(string)
	req = httptest.NewRequest("GET", "/deleteShare?id="+id, nil)
	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Verify deletion
	req = httptest.NewRequest("GET", "/getShares?f=json", nil)
	resp, err = app.Test(req)

	json.NewDecoder(resp.Body).Decode(&response)
	subResp = response["subsonic-response"].(map[string]interface{})

	if s, ok := subResp["shares"]; ok {
		if sMap, ok := s.(map[string]interface{}); ok {
			if list, ok := sMap["share"]; ok {
				assert.Len(t, list, 0)
			}
		}
	}
}

func TestVideoHandler_GetVideos(t *testing.T) {
	db := setupPhase5TestDB(t)
	app := fiber.New()
	handler := NewVideoHandler(db)
	app.Get("/getVideos", handler.GetVideos)

	req := httptest.NewRequest("GET", "/getVideos?f=json", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Test GetCaptions
	app.Get("/getCaptions", handler.GetCaptions)
	req = httptest.NewRequest("GET", "/getCaptions?id=1", nil)
	resp, err = app.Test(req)
	assert.NoError(t, err)
	// Stub returns 404
	assert.Equal(t, 404, resp.StatusCode)
}

func TestChatHandler_AddAndGet(t *testing.T) {
	db := setupPhase5TestDB(t)
	app := fiber.New()

	// Setup data
	user := models.User{Username: "chatuser"}
	db.Create(&user)

	handler := NewChatHandler(db)

	// Mock auth middleware
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user", &user)
		return c.Next()
	})

	app.Get("/addChatMessage", handler.AddChatMessage)
	app.Get("/getChatMessages", handler.GetChatMessages)

	// Test Add
	req := httptest.NewRequest("GET", "/addChatMessage?message=Hello&f=json", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)
	subResp := response["subsonic-response"].(map[string]interface{})
	chatMessages := subResp["chatMessages"].(map[string]interface{})
	messages := chatMessages["chatMessage"].([]interface{})

	assert.Len(t, messages, 1)
	msg := messages[0].(map[string]interface{})
	assert.Equal(t, "Hello", msg["message"])
	assert.Equal(t, "chatuser", msg["username"])

	// Test Get
	req = httptest.NewRequest("GET", "/getChatMessages?f=json", nil)
	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestScanHandler_GetStatus(t *testing.T) {
	db := setupPhase5TestDB(t)
	app := fiber.New()
	handler := NewScanHandler(db)
	app.Get("/getScanStatus", handler.GetScanStatus)

	req := httptest.NewRequest("GET", "/getScanStatus?f=json", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestJukeboxHandler_Control(t *testing.T) {
	db := setupPhase5TestDB(t)
	app := fiber.New()
	handler := NewJukeboxHandler(db)
	app.Get("/jukeboxControl", handler.JukeboxControl)

	req := httptest.NewRequest("GET", "/jukeboxControl?action=status&f=json", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}
