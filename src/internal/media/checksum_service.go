package media

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gorm.io/gorm"
	"melodee/internal/models"
	"melodee/internal/utils"
)

// ChecksumConfig holds configuration for checksum service
type ChecksumConfig struct {
	Algorithm     string        // "sha256" or "crc32"
	EnableCaching bool          // Whether to cache checksum calculations
	CacheTTL      time.Duration // How long to cache checksums
	StoreLocation string        // Where to store checksums (e.g. "DB", "File")
}

// DefaultChecksumConfig returns the default checksum configuration
func DefaultChecksumConfig() *ChecksumConfig {
	return &ChecksumConfig{
		Algorithm:     "sha256",  // Using SHA256 for better collision resistance
		EnableCaching: true,      // Enable caching by default
		CacheTTL:      24 * time.Hour, // Default cache TTL
		StoreLocation: "DB",      // Default storage location
	}
}

// CachedChecksum holds a checksum with its timestamp for cache management
type CachedChecksum struct {
	Value     string
	Timestamp time.Time
}

// ChecksumService provides checksum functionality for media files
type ChecksumService struct {
	db       *gorm.DB
	config   *ChecksumConfig
	cache    map[string]CachedChecksum
	cacheMu  sync.RWMutex
}

// NewChecksumService creates a new checksum service
func NewChecksumService(db *gorm.DB, config *ChecksumConfig) *ChecksumService {
	if config == nil {
		config = DefaultChecksumConfig()
	}

	return &ChecksumService{
		db:       db,
		config:   config,
		cache:    make(map[string]CachedChecksum),
	}
}

// CalculateFileChecksum calculates the checksum for a media file
func (cs *ChecksumService) CalculateFileChecksum(filePath string) (string, error) {
	if cs.config.EnableCaching {
		// Check if we have a cached checksum
		if cached, found := cs.getCachedChecksum(filePath); found {
			return cached, nil
		}
	}

	var checksum string
	var err error

	switch cs.config.Algorithm {
	case "sha256":
		checksum, err = utils.CalculateFileSHA256Checksum(filePath)
	case "crc32":
		checksum, err = utils.CalculateFileCRC32Checksum(filePath)
	default:
		// Default to SHA256 for better security
		checksum, err = utils.CalculateFileSHA256Checksum(filePath)
	}

	if err != nil {
		return "", err
	}

	if cs.config.EnableCaching {
		cs.setCachedChecksum(filePath, checksum)
	}

	return checksum, nil
}

// getCachedChecksum gets a checksum from the cache if it exists and is not expired
func (cs *ChecksumService) getCachedChecksum(filePath string) (string, bool) {
	if !cs.config.EnableCaching {
		return "", false
	}

	cs.cacheMu.RLock()
	defer cs.cacheMu.RUnlock()

	if cached, exists := cs.cache[filePath]; exists {
		// Check if cache entry has expired
		if time.Since(cached.Timestamp) < cs.config.CacheTTL {
			return cached.Value, true
		}
		// Entry expired, remove it
		delete(cs.cache, filePath)
	}

	return "", false
}

// setCachedChecksum sets a checksum in the cache
func (cs *ChecksumService) setCachedChecksum(filePath, checksum string) {
	if !cs.config.EnableCaching {
		return
	}

	cs.cacheMu.Lock()
	defer cs.cacheMu.Unlock()

	cs.cache[filePath] = CachedChecksum{
		Value:     checksum,
		Timestamp: time.Now(),
	}
}

// ClearCache clears the checksum cache
func (cs *ChecksumService) ClearCache() {
	if cs.config.EnableCaching {
		cs.cacheMu.Lock()
		defer cs.cacheMu.Unlock()
		cs.cache = make(map[string]CachedChecksum)
	}
}

// GetCacheSize returns the current size of the checksum cache
func (cs *ChecksumService) GetCacheSize() int {
	cs.cacheMu.RLock()
	defer cs.cacheMu.RUnlock()
	return len(cs.cache)
}

// SaveChecksumForSong saves the checksum for a song in the database
func (cs *ChecksumService) SaveChecksumForSong(songID int64, filePath string) error {
	// Calculate checksum for the file
	checksum, err := cs.CalculateFileChecksum(filePath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum for file %s: %w", filePath, err)
	}

	// Update the song's CRC hash in the database
	var song models.Song
	if err := cs.db.First(&song, songID).Error; err != nil {
		return fmt.Errorf("failed to find song with ID %d: %w", songID, err)
	}

	song.CrcHash = checksum
	if err := cs.db.Save(&song).Error; err != nil {
		return fmt.Errorf("failed to save checksum to database: %w", err)
	}

	return nil
}

// SaveChecksumForSongByPath saves the checksum for a song by its file path
func (cs *ChecksumService) SaveChecksumForSongByPath(filePath string) error {
	// Calculate checksum for the file
	checksum, err := cs.CalculateFileChecksum(filePath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum for file %s: %w", filePath, err)
	}

	// Find the song by its relative path
	var song models.Song
	if err := cs.db.Where("relative_path = ?", filePath).First(&song).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("song with path %s not found in database", filePath)
		}
		return fmt.Errorf("failed to find song: %w", err)
	}

	// Update the CRC hash in the database
	song.CrcHash = checksum
	if err := cs.db.Save(&song).Error; err != nil {
		return fmt.Errorf("failed to save checksum to database: %w", err)
	}

	return nil
}

// VerifyFileChecksum verifies if a file's checksum matches the stored value
func (cs *ChecksumService) VerifyFileChecksum(filePath string) (bool, error) {
	// Calculate current checksum
	currentChecksum, err := cs.CalculateFileChecksum(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to calculate current checksum: %w", err)
	}

	// Find the corresponding song record to get stored checksum
	var song models.Song
	if err := cs.db.Where("relative_path = ?", filePath).First(&song).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, fmt.Errorf("song with path %s not found in database", filePath)
		}
		return false, fmt.Errorf("failed to find song record: %w", err)
	}

	// Compare checksums
	return song.CrcHash == currentChecksum, nil
}

// IsFileProcessed checks if a file has already been processed by comparing checksums
func (cs *ChecksumService) IsFileProcessed(filePath string) (bool, string, error) {
	// Calculate current checksum
	currentChecksum, err := cs.CalculateFileChecksum(filePath)
	if err != nil {
		return false, "", fmt.Errorf("failed to calculate file checksum: %w", err)
	}

	// Check if a song with this checksum already exists
	var existingSong models.Song
	if err := cs.db.Where("crc_hash = ?", currentChecksum).First(&existingSong).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// No song with this checksum exists, so file is not processed
			return false, currentChecksum, nil
		}
		return false, "", fmt.Errorf("failed to search for existing song: %w", err)
	}

	// Song with this checksum exists, so file has been processed
	return true, currentChecksum, nil
}

// ProcessFileWithIdempotency processes a file only if it hasn't been processed before
func (cs *ChecksumService) ProcessFileWithIdempotency(filePath string, processFn func() error) (bool, error) {
	// Check if file has already been processed
	alreadyProcessed, checksum, err := cs.IsFileProcessed(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to check if file is processed: %w", err)
	}

	if alreadyProcessed {
		// File already exists in the system with the same checksum
		return true, nil // Already processed, return success
	}

	// Process the file
	if err := processFn(); err != nil {
		return false, fmt.Errorf("failed to process file: %w", err)
	}

	// Save the checksum to mark the file as processed
	var song models.Song
	if err := cs.db.Where("relative_path = ?", filePath).First(&song).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Song record doesn't exist yet, this is normal for new files
			// We'll need to create or update after processing
		} else {
			return false, fmt.Errorf("failed to find song record: %w", err)
		}
	} else {
		// Update existing record with checksum
		song.CrcHash = checksum
		if err := cs.db.Save(&song).Error; err != nil {
			return false, fmt.Errorf("failed to save checksum: %w", err)
		}
	}

	return false, nil
}

// GetSongByChecksum retrieves a song by its checksum
func (cs *ChecksumService) GetSongByChecksum(checksum string) (*models.Song, error) {
	var song models.Song
	if err := cs.db.Where("crc_hash = ?", checksum).First(&song).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no song found with checksum %s", checksum)
		}
		return nil, fmt.Errorf("failed to find song by checksum: %w", err)
	}

	return &song, nil
}

// BatchCalculateChecksums calculates checksums for multiple files
func (cs *ChecksumService) BatchCalculateChecksums(filePaths []string) (map[string]string, error) {
	results := make(map[string]string)
	
	for _, filePath := range filePaths {
		checksum, err := cs.CalculateFileChecksum(filePath)
		if err != nil {
			// Log error but continue with other files
			fmt.Printf("Failed to calculate checksum for %s: %v\n", filePath, err)
			continue
		}
		results[filePath] = checksum
	}
	
	return results, nil
}

// ValidateChecksumIntegrity validates the integrity of all checksums in the database
func (cs *ChecksumService) ValidateChecksumIntegrity() ([]int64, error) {
	var songs []models.Song
	if err := cs.db.Where("crc_hash IS NOT NULL AND crc_hash != ''").Find(&songs).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve songs with checksums: %w", err)
	}

	var invalidSongIDs []int64
	for _, song := range songs {
		// Validate that the file still has the same checksum
		if song.RelativePath != "" {
			currentChecksum, err := cs.CalculateFileChecksum(song.RelativePath)
			if err != nil {
				// File might not exist anymore
				invalidSongIDs = append(invalidSongIDs, song.ID)
				continue
			}
			
			if currentChecksum != song.CrcHash {
				// Checksum mismatch
				invalidSongIDs = append(invalidSongIDs, song.ID)
			}
		}
	}
	
	return invalidSongIDs, nil
}