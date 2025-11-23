package media

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TranscodeCache manages cached transcoded files
type TranscodeCache struct {
	cacheDir      string
	maxSize       int64 // Maximum cache size in bytes
	currentSize   int64
	cachedFiles   map[string]*CachedFile
	mutex         sync.RWMutex
	cleanupTicker *time.Ticker
}

// CachedFile represents a cached transcoded file
type CachedFile struct {
	Path       string
	SourceHash string
	Profile    string
	Format     string
	MaxBitRate int
	Size       int64
	AccessTime time.Time
	CreatedAt  time.Time
}

// TranscodeService handles media transcoding with caching
type TranscodeService struct {
	processor *FFmpegProcessor
	cache     *TranscodeCache
}

// NewTranscodeService creates a new transcoding service with caching
func NewTranscodeService(processor *FFmpegProcessor, cacheDir string, maxSize int64) *TranscodeService {
	cache := &TranscodeCache{
		cacheDir:    cacheDir,
		maxSize:     maxSize,
		cachedFiles: make(map[string]*CachedFile),
	}

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		fmt.Printf("Warning: failed to create cache directory: %v\n", err)
	}

	// Start cleanup goroutine
	cache.cleanupTicker = time.NewTicker(1 * time.Hour) // Clean up every hour
	go cache.cleanupRoutine()

	return &TranscodeService{
		processor: processor,
		cache:     cache,
	}
}

// TranscodeWithCache transcodes a media file using caching
func (ts *TranscodeService) TranscodeWithCache(inputPath string, profileName string, maxBitRate int, format string) (string, error) {
	// Generate a unique cache key based on input file, profile, and parameters
	cacheKey, err := ts.generateCacheKey(inputPath, profileName, maxBitRate, format)
	if err != nil {
		return "", fmt.Errorf("failed to generate cache key: %w", err)
	}

	// Check if we have a cached version
	if cachedPath, exists := ts.cache.Get(cacheKey); exists {
		// Update access time and return cached file
		ts.cache.UpdateAccessTime(cacheKey)
		return cachedPath, nil
	}

	// Create output path for transcoded file
	outputPath := filepath.Join(ts.cache.cacheDir, cacheKey+".tmp."+format)

	// Perform transcoding
	if err := ts.processor.TranscodeFile(inputPath, outputPath, profileName); err != nil {
		// Clean up temp file if transcoding failed
		os.Remove(outputPath)
		return "", fmt.Errorf("transcoding failed: %w", err)
	}

	// Get file info for cache tracking
	fileInfo, err := os.Stat(outputPath)
	if err != nil {
		// Clean up temp file if we can't get its info
		os.Remove(outputPath)
		return "", fmt.Errorf("failed to get output file info: %w", err)
	}

	// Check if adding this file would exceed cache size
	if !ts.cache.WouldFit(fileInfo.Size()) {
		// Try to make space by evicting old files
		ts.cache.EvictOldest(fileInfo.Size())
	}

	// Add to cache
	ts.cache.Add(cacheKey, inputPath, profileName, format, maxBitRate, outputPath)

	// Rename temp file to final name to make it visible
	finalPath := strings.TrimSuffix(outputPath, ".tmp."+format) + "." + format
	if err := os.Rename(outputPath, finalPath); err != nil {
		// If rename fails, still return the temp file
		return outputPath, nil
	}

	return finalPath, nil
}

// generateCacheKey generates a unique key for caching based on input parameters
func (ts *TranscodeService) generateCacheKey(inputPath string, profileName string, maxBitRate int, format string) (string, error) {
	// Get file info to include in the key for cache invalidation
	fileInfo, err := os.Stat(inputPath)
	if err != nil {
		return "", err
	}

	// Create source hash from file path, modification time, and size
	sourceInfo := fmt.Sprintf("%s-%d-%d", inputPath, fileInfo.ModTime().Unix(), fileInfo.Size())
	sourceHash := fmt.Sprintf("%x", sha256.Sum256([]byte(sourceInfo)))

	// Create cache key combining source hash, profile, and parameters
	cacheKey := fmt.Sprintf("%s_%s_%d_%s_%s",
		sourceHash[:16], // First 16 chars of source hash
		profileName,
		maxBitRate,
		format,
		uuid.New().String()[:8], // Add random component to prevent collisions
	)

	// Sanitize the key to be filesystem-safe
	cacheKey = sanitizeCacheKey(cacheKey)

	return cacheKey, nil
}

// sanitizeCacheKey removes characters that are unsafe for filesystem paths
func sanitizeCacheKey(key string) string {
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", "%"}
	sanitized := key

	for _, char := range invalidChars {
		sanitized = strings.ReplaceAll(sanitized, char, "_")
	}

	return sanitized
}

// Get gets a cached file path if it exists
func (c *TranscodeCache) Get(cacheKey string) (string, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	cachedFile, exists := c.cachedFiles[cacheKey]
	if !exists {
		return "", false
	}

	// Check if file still exists on disk
	if _, err := os.Stat(cachedFile.Path); os.IsNotExist(err) {
		// File was deleted externally, remove from cache
		delete(c.cachedFiles, cacheKey)
		c.currentSize -= cachedFile.Size
		return "", false
	}

	return cachedFile.Path, true
}

// Add adds a file to the cache
func (c *TranscodeCache) Add(cacheKey, sourcePath, profile, format string, maxBitRate int, outputPath string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Get file info
	fileInfo, err := os.Stat(outputPath)
	if err != nil {
		return // Skip if we can't get file info
	}

	// Create cached file entry
	cachedFile := &CachedFile{
		Path:       outputPath,
		SourceHash: cacheKey, // This should be the actual source hash
		Profile:    profile,
		Format:     format,
		MaxBitRate: maxBitRate,
		Size:       fileInfo.Size(),
		AccessTime: time.Now(),
		CreatedAt:  time.Now(),
	}

	// Add to cache
	c.cachedFiles[cacheKey] = cachedFile
	c.currentSize += fileInfo.Size()
}

// UpdateAccessTime updates the access time for a cached file
func (c *TranscodeCache) UpdateAccessTime(cacheKey string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if cachedFile, exists := c.cachedFiles[cacheKey]; exists {
		cachedFile.AccessTime = time.Now()
	}
}

// WouldFit checks if adding a file of the given size would fit in the cache
func (c *TranscodeCache) WouldFit(size int64) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.currentSize+size <= c.maxSize
}

// EvictOldest evicts the oldest files to make space for a new file of the given size
func (c *TranscodeCache) EvictOldest(size int64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Calculate how much space we need to free
	needToFree := (c.currentSize + size) - c.maxSize
	freed := int64(0)

	// Find oldest files to evict
	for freed < needToFree {
		oldestFileKey := ""
		oldestTime := time.Now()

		// Find the oldest accessed file
		for key, cachedFile := range c.cachedFiles {
			if cachedFile.AccessTime.Before(oldestTime) {
				oldestTime = cachedFile.AccessTime
				oldestFileKey = key
			}
		}

		// If no more files to evict, break
		if oldestFileKey == "" {
			break
		}

		// Remove the oldest file
		oldestFile := c.cachedFiles[oldestFileKey]
		delete(c.cachedFiles, oldestFileKey)
		c.currentSize -= oldestFile.Size
		freed += oldestFile.Size

		// Delete the file from disk
		if err := os.Remove(oldestFile.Path); err != nil {
			fmt.Printf("Warning: failed to delete cached file: %v\n", err)
		}
	}
}

// cleanupRoutine periodically cleans up the cache
func (c *TranscodeCache) cleanupRoutine() {
	for range c.cleanupTicker.C {
		c.cleanupStaleFiles()
	}
}

// cleanupStaleFiles removes stale or invalid cache entries
func (c *TranscodeCache) cleanupStaleFiles() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for key, cachedFile := range c.cachedFiles {
		// Check if file exists on disk
		if _, err := os.Stat(cachedFile.Path); os.IsNotExist(err) {
			// File was deleted externally, remove from cache
			delete(c.cachedFiles, key)
			c.currentSize -= cachedFile.Size
		}
	}
}

// Close stops the cleanup routine
func (c *TranscodeCache) Close() {
	if c.cleanupTicker != nil {
		c.cleanupTicker.Stop()
	}
}

// GetCacheStats returns cache statistics
func (c *TranscodeCache) GetCacheStats() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return map[string]interface{}{
		"current_size": c.currentSize,
		"max_size":     c.maxSize,
		"file_count":   len(c.cachedFiles),
		"used_percent": float64(c.currentSize) / float64(c.maxSize) * 100,
	}
}

// ClearCache clears all cached files
func (c *TranscodeCache) ClearCache() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for key, cachedFile := range c.cachedFiles {
		// Delete file from disk
		if err := os.Remove(cachedFile.Path); err != nil {
			fmt.Printf("Warning: failed to delete cached file: %v\n", err)
		}
		delete(c.cachedFiles, key)
	}

	c.currentSize = 0
}

// GetCacheDir returns the cache directory path
func (ts *TranscodeService) GetCacheDir() string {
	return ts.cache.cacheDir
}
