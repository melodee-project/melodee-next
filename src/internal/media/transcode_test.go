package media

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTranscodeService(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a mock FFmpeg processor
	ffmpegConfig := DefaultFFmpegConfig()
	processor := NewFFmpegProcessor(ffmpegConfig)

	// Create transcode service with cache
	cacheDir := filepath.Join(tempDir, "cache")
	service := NewTranscodeService(processor, cacheDir, 100*1024*1024) // 100MB max cache

	// Test that cache directory was created
	_, err := os.Stat(cacheDir)
	assert.NoError(t, err)

	// Test cache stats
	stats := service.cache.GetCacheStats()
	assert.Equal(t, int64(0), stats["current_size"])
	assert.Equal(t, int64(100*1024*1024), stats["max_size"])
	assert.Equal(t, 0, stats["file_count"])
}

func TestSanitizeCacheKey(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"Normal key", "test_key_123", "test_key_123"},
		{"Key with slashes", "test/key/123", "test_key_123"},
		{"Key with colons", "test:key:123", "test_key_123"},
		{"Key with asterisks", "test*key*123", "test_key_123"},
		{"Key with question marks", "test?key?123", "test_key_123"},
		{"Key with quotes", `test"key"123`, "test_key_123"},
		{"Key with brackets", "test<key>123", "test_key_123"},
		{"Key with pipes", "test|key|123", "test_key_123"},
		{"Key with percent", "test%key%123", "test_key_123"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeCacheKey(tc.input)
			assert.Equal(t, tc.expected, result)
			// Ensure no invalid characters remain
			for _, invalidChar := range []string{"/", ":", "*", "?", "\"", "<", ">", "|", "%"} {
				assert.NotContains(t, result, invalidChar)
			}
		})
	}
}

func TestTranscodeCacheOperations(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a mock FFmpeg processor
	ffmpegConfig := DefaultFFmpegConfig()
	processor := NewFFmpegProcessor(ffmpegConfig)

	// Create transcode service with cache
	cacheDir := filepath.Join(tempDir, "cache")
	service := NewTranscodeService(processor, cacheDir, 100*1024*1024) // 100MB max cache

	// Create a test file (empty file for testing)
	testInputPath := filepath.Join(tempDir, "test_input.mp3")
	err := os.WriteFile(testInputPath, []byte("fake mp3 content"), 0644)
	assert.NoError(t, err)

	// Test adding and getting from cache
	cacheKey := "test_key_12345"
	outputPath := filepath.Join(cacheDir, "test_output.mp3")

	// Create output file as well to simulate transcoding
	err = os.WriteFile(outputPath, []byte("fake transcoded content"), 0644)
	assert.NoError(t, err)

	// Add to cache
	service.cache.Add(cacheKey, testInputPath, "test_profile", "mp3", 192, outputPath)

	// Get from cache
	retrievedPath, exists := service.cache.Get(cacheKey)
	assert.True(t, exists)
	assert.Equal(t, outputPath, retrievedPath)

	// Update access time
	service.cache.UpdateAccessTime(cacheKey)

	// Check stats
	stats := service.cache.GetCacheStats()
	assert.Greater(t, stats["file_count"], 0)
}

func TestTranscodeCacheEviction(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a mock FFmpeg processor
	ffmpegConfig := DefaultFFmpegConfig()
	processor := NewFFmpegProcessor(ffmpegConfig)

	// Create transcode service with a very small cache (1KB)
	cacheDir := filepath.Join(tempDir, "cache")
	service := NewTranscodeService(processor, cacheDir, 1024) // 1KB max cache

	// Simulate adding files that exceed cache size
	cacheKey1 := "test_key_1"
	outputPath1 := filepath.Join(cacheDir, "output1.mp3")
	err := os.WriteFile(outputPath1, make([]byte, 800), 0644) // 800 bytes
	assert.NoError(t, err)

	service.cache.Add(cacheKey1, "input1.mp3", "profile1", "mp3", 128, outputPath1)

	// Add another file that exceeds cache size
	cacheKey2 := "test_key_2"
	outputPath2 := filepath.Join(cacheDir, "output2.mp3")
	err = os.WriteFile(outputPath2, make([]byte, 500), 0644) // 500 bytes - total would be 1300 bytes
	assert.NoError(t, err)

	service.cache.Add(cacheKey2, "input2.mp3", "profile2", "mp3", 128, outputPath2)

	// The eviction logic should be triggered when trying to add a file that exceeds capacity
	// Check that the cache size is within limits
	stats := service.cache.GetCacheStats()
	assert.LessOrEqual(t, stats["current_size"], 1024.0)
}

func TestTranscodeCacheCleanup(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a mock FFmpeg processor
	ffmpegConfig := DefaultFFmpegConfig()
	processor := NewFFmpegProcessor(ffmpegConfig)

	// Create transcode service with cache
	cacheDir := filepath.Join(tempDir, "cache")
	service := NewTranscodeService(processor, cacheDir, 100*1024*1024) // 100MB max cache

	// Create a test file
	testInputPath := filepath.Join(tempDir, "test_input.mp3")
	err := os.WriteFile(testInputPath, []byte("fake content"), 0644)
	assert.NoError(t, err)

	// Add an entry to cache
	cacheKey := "test_key_cleanup"
	outputPath := filepath.Join(cacheDir, "test_output.mp3")
	err = os.WriteFile(outputPath, []byte("fake transcoded content"), 0644)
	assert.NoError(t, err)

	service.cache.Add(cacheKey, testInputPath, "test_profile", "mp3", 192, outputPath)

	// Verify it exists
	_, exists := service.cache.Get(cacheKey)
	assert.True(t, exists)

	// Delete the file from disk (simulate external deletion)
	err = os.Remove(outputPath)
	assert.NoError(t, err)

	// Manually call cleanup to test the cleanup logic
	service.cache.cleanupStaleFiles()

	// Should no longer exist in cache
	_, exists = service.cache.Get(cacheKey)
	assert.False(t, exists)
}

func TestTranscodeCacheWouldFit(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a mock FFmpeg processor
	ffmpegConfig := DefaultFFmpegConfig()
	processor := NewFFmpegProcessor(ffmpegConfig)

	// Create transcode service with cache
	cacheDir := filepath.Join(tempDir, "cache")
	service := NewTranscodeService(processor, cacheDir, 1024) // 1KB max cache

	// Test with zero size - should always fit
	assert.True(t, service.cache.WouldFit(0))
	assert.True(t, service.cache.WouldFit(512)) // 512 bytes should fit in 1KB
	assert.False(t, service.cache.WouldFit(2048)) // 2KB won't fit in 1KB
}

func TestTranscodeCacheClear(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a mock FFmpeg processor
	ffmpegConfig := DefaultFFmpegConfig()
	processor := NewFFmpegProcessor(ffmpegConfig)

	// Create transcode service with cache
	cacheDir := filepath.Join(tempDir, "cache")
	service := NewTranscodeService(processor, cacheDir, 100*1024*1024) // 100MB max cache

	// Create and add test files to cache
	for i := 0; i < 3; i++ {
		cacheKey := "test_key_" + string(rune('0'+i))
		outputPath := filepath.Join(cacheDir, "output"+string(rune('0'+i))+".mp3")
		err := os.WriteFile(outputPath, []byte("fake transcoded content "+string(rune('0'+i))), 0644)
		assert.NoError(t, err)
		service.cache.Add(cacheKey, "input"+string(rune('0'+i))+".mp3", "profile", "mp3", 192, outputPath)
	}

	// Verify files exist
	stats := service.cache.GetCacheStats()
	assert.Equal(t, 3, stats["file_count"])

	// Clear cache
	service.cache.ClearCache()

	// Verify cache is empty
	stats = service.cache.GetCacheStats()
	assert.Equal(t, 0, stats["file_count"])
	assert.Equal(t, int64(0), stats["current_size"])
}

func TestTranscodeServiceClose(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a mock FFmpeg processor
	ffmpegConfig := DefaultFFmpegConfig()
	processor := NewFFmpegProcessor(ffmpegConfig)

	// Create transcode service with cache
	cacheDir := filepath.Join(tempDir, "cache")
	service := NewTranscodeService(processor, cacheDir, 100*1024*1024) // 100MB max cache

	// Close the service
	service.cache.Close()

	// Just verify that the function can be called without panicking
	// The ticker closure should happen without errors
}