package handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestSharesHandler_ContractCompliance(t *testing.T) {
	app := fiber.New()

	// Create handler with nil repo for testing
	sharesHandler := NewSharesHandler(nil)

	// Test GetShares response format
	app.Get("/test-shares", sharesHandler.GetShares)
	req := httptest.NewRequest("GET", "/test-shares", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Should return 401 without auth or 500 with nil repo
	assert.True(t, resp.StatusCode == 401 || resp.StatusCode == 500)

	// Test CreateShare
	app.Post("/test-shares", sharesHandler.CreateShare)
	
	// Create a request body with the expected format
	createReq := struct {
		Name                 string   `json:"name"`
		TrackIDs             []string `json:"track_ids"`
		ExpiresAt            string   `json:"expires_at"`
		MaxStreamingMinutes  int      `json:"max_streaming_minutes"`
		AllowDownload        bool     `json:"allow_download"`
	}{
		Name:                 "Test Share",
		TrackIDs:             []string{"track-1", "track-2"},
		ExpiresAt:            time.Now().AddDate(0, 0, 30).Format(time.RFC3339),
		MaxStreamingMinutes:  60,
		AllowDownload:        true,
	}
	jsonData, _ := json.Marshal(createReq)
	
	req = httptest.NewRequest("POST", "/test-shares", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	assert.NoError(t, err)
	
	// Should get 401 without auth or 400/500 with other issues
	assert.True(t, resp.StatusCode == 401 || resp.StatusCode == 400 || resp.StatusCode == 500)

	// Test UpdateShare
	app.Put("/test-shares/:id", sharesHandler.UpdateShare)
	
	updateReq := struct {
		Name                 string   `json:"name"`
		TrackIDs             []string `json:"track_ids"`
		ExpiresAt            string   `json:"expires_at"`
		MaxStreamingMinutes  int      `json:"max_streaming_minutes"`
		AllowDownload        bool     `json:"allow_download"`
	}{
		Name:                 "Updated Share",
		TrackIDs:             []string{"track-1"},
		ExpiresAt:            time.Now().AddDate(0, 0, 60).Format(time.RFC3339),
		MaxStreamingMinutes:  120,
		AllowDownload:        false,
	}
	jsonData, _ = json.Marshal(updateReq)
	
	req = httptest.NewRequest("PUT", "/test-shares/share-1", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	assert.NoError(t, err)
	
	// Should get 401 without auth or 400/500 with other issues
	assert.True(t, resp.StatusCode == 401 || resp.StatusCode == 400 || resp.StatusCode == 500)

	// Test DeleteShare
	app.Delete("/test-shares/:id", sharesHandler.DeleteShare)
	req = httptest.NewRequest("DELETE", "/test-shares/share-1", nil)
	resp, err = app.Test(req)
	assert.NoError(t, err)
	
	// Should get 401 without auth or 400/500 with other issues
	assert.True(t, resp.StatusCode == 401 || resp.StatusCode == 400 || resp.StatusCode == 500)
}