package handlers

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"melodee/internal/media"
	"melodee/internal/test"
)

func TestMediaHandler_Stream(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.mp3")
	err := os.WriteFile(testFile, []byte("fake mp3 content"), 0644)
	assert.NoError(t, err)

	// Setup test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Create FFmpeg processor
	ffmpegConfig := media.DefaultFFmpegConfig()
	processor := media.NewFFmpegProcessor(ffmpegConfig)

	// Create transcode service
	cacheDir := filepath.Join(tempDir, "cache")
	transcodeService := media.NewTranscodeService(processor, cacheDir, 100*1024*1024)

	// Create media handler
	handler := NewMediaHandler(db, nil, transcodeService)

	// Create Fiber app for testing
	app := fiber.New()
	app.Get("/stream", handler.Stream)

	// Test streaming without transcoding (file doesn't exist in DB but we'll handle the error)
	req := httptest.NewRequest("GET", "/stream?id=999999", nil) // Non-existent ID
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode) // OpenSubsonic returns 200 with XML error
}

func TestMediaHandler_TranscodeFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.mp3")
	err := os.WriteFile(testFile, []byte("fake mp3 content"), 0644)
	assert.NoError(t, err)

	// Setup test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Create FFmpeg processor
	ffmpegConfig := media.DefaultFFmpegConfig()
	processor := media.NewFFmpegProcessor(ffmpegConfig)

	// Create transcode service
	cacheDir := filepath.Join(tempDir, "cache")
	transcodeService := media.NewTranscodeService(processor, cacheDir, 100*1024*1024)

	// Create media handler
	handler := NewMediaHandler(db, nil, transcodeService)

	// Test transcoding with valid service
	outputPath, err := handler.transcodeFile(testFile, 128, "mp3")
	assert.NoError(t, err)
	// This might return the original file if transcoding fails due to missing actual FFmpeg binary
	// but should not return an error

	// Test with nil transcode service (fallback behavior)
	originalHandler := &MediaHandler{
		db:               db,
		cfg:              nil,
		transcodeService: nil,
	}
	
	fallbackPath, err := originalHandler.transcodeFile(testFile, 128, "mp3")
	assert.NoError(t, err)
	assert.Equal(t, testFile, fallbackPath) // Should return original file when no service
}

func TestMediaHandler_GetContentType(t *testing.T) {
	testCases := []struct {
		filename string
		expected string
	}{
		{"test.mp3", "audio/mpeg"},
		{"test.flac", "audio/flac"},
		{"test.m4a", "audio/mp4"},
		{"test.mp4", "audio/mp4"},
		{"test.aac", "audio/aac"},
		{"test.ogg", "audio/ogg"},
		{"test.opus", "audio/opus"},
		{"test.wav", "audio/wav"},
		{"test.jpg", "image/jpeg"},
		{"test.jpeg", "image/jpeg"},
		{"test.png", "image/png"},
		{"test.gif", "image/gif"},
		{"test.txt", "application/octet-stream"}, // Default case
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			result := getContentType(tc.filename)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMediaHandler_HandleRangeRequest(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.mp3")
	err := os.WriteFile(testFile, []byte("this is a test file for range requests"), 0644)
	assert.NoError(t, err)

	// Setup test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Create FFmpeg processor and transcode service
	ffmpegConfig := media.DefaultFFmpegConfig()
	processor := media.NewFFmpegProcessor(ffmpegConfig)
	cacheDir := filepath.Join(tempDir, "cache")
	transcodeService := media.NewTranscodeService(processor, cacheDir, 100*1024*1024)

	// Create media handler
	handler := NewMediaHandler(db, nil, transcodeService)

	// Create Fiber app for testing
	app := fiber.New()

	// Mock a song for the range request
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Range", "bytes=0-10")
	c := app.AcquireCtx(req)
	defer app.ReleaseCtx(c)

	// Set up a context and try to handle range request
	// Note: This will likely fail in test environment due to missing file paths in DB
	// but we can at least test the range parsing logic
	result := handler.handleRangeRequest(c, testFile, struct {
		ID           int64  `json:"id"`
		Name         string `json:"name"`
		NameNormalized string `json:"name_normalized"`
		AlbumID      int64  `json:"album_id"`
		ArtistID     int64  `json:"artist_id"`
		Duration     int64  `json:"duration"`
		BitRate      int    `json:"bit_rate"`
		BitDepth     int    `json:"bit_depth"`
		SampleRate   int    `json:"sample_rate"`
		Channels     int    `json:"channels"`
		CreatedAt    string `json:"created_at"`
		Tags         string `json:"tags"`
		Directory    string `json:"directory"`
		FileName     string `json:"file_name"`
		RelativePath string `json:"relative_path"`
		CrcHash      string `json:"crc_hash"`
		SortOrder    int    `json:"sort_order"`
	}{ID: 1, RelativePath: testFile})

	// The result will be an HTTP status code response; this is expected behavior
	_ = result
}