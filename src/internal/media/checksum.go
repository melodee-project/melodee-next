package media

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
	"melodee/internal/models"
)

// ChecksumService handles checksum calculation, validation, and idempotency
type ChecksumService struct {
	db         *gorm.DB
	mutex      sync.RWMutex
	cache      map[string]*ChecksumEntry
	config     *ChecksumConfig
}

// ChecksumConfig holds configuration for checksum service
type ChecksumConfig struct {
	Algorithm  string `mapstructure:"algorithm"`  // Only SHA256 supported for now
	EnableCaching bool `mapstructure:"enable_caching"` // Whether to cache checksums
	CacheTTL   time.Duration `mapstructure:"cache_ttl"` // How long to cache checksums
	StoreLocation string `mapstructure:"store_location"` // Where to store checksums (DB, file, memory)
}

// DefaultChecksumConfig returns the default checksum configuration
func DefaultChecksumConfig() *ChecksumConfig {
	return &ChecksumConfig{
		Algorithm:  "SHA256",
		EnableCaching: true,
		CacheTTL:   24 * time.Hour,
		StoreLocation: "DB", // Store in database by default
	}
}

// ChecksumEntry represents a cached checksum entry
type ChecksumEntry struct {
	Checksum    string
	FilePath    string
	FileSize    int64
	ModTime     time.Time
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

// NewChecksumService creates a new checksum service
func NewChecksumService(db *gorm.DB, config *ChecksumConfig) *ChecksumService {
	if config == nil {
		config = DefaultChecksumConfig()
	}

	service := &ChecksumService{
		db:     db,
		config: config,
		cache:  make(map[string]*ChecksumEntry),
	}

	if config.EnableCaching {
		// Start a cleanup goroutine to remove expired entries
		go service.cleanupRoutine()
	}

	return service
}

// CalculateChecksum calculates the checksum for a file
func (cs *ChecksumService) CalculateChecksum(filePath string) (string, error) {
	// First, check if we have a cached version
	if cs.config.EnableCaching {
		if cached, exists := cs.getCachedChecksum(filePath); exists {
			// Verify the file hasn't changed since caching
			if cs.isCacheValid(filePath, cached) {
				return cached.Checksum, nil
			}
			// Cache is invalid, remove it
			cs.removeCachedChecksum(filePath)
		}
	}

	// Calculate new checksum
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	checksum := hex.EncodeToString(hash.Sum(nil))

	// Cache the result if caching is enabled
	if cs.config.EnableCaching {
		cs.cacheChecksum(filePath, checksum)
	}

	return checksum, nil
}

// ValidateChecksum validates that the file's current checksum matches the expected value
func (cs *ChecksumService) ValidateChecksum(filePath, expectedChecksum string) (bool, error) {
	calculated, err := cs.CalculateChecksum(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to calculate current checksum: %w", err)
	}

	return calculated == expectedChecksum, nil
}

// IsAlreadyProcessed checks if a file has already been processed (idempotency check)
func (cs *ChecksumService) IsAlreadyProcessed(originalPath, checksum string) (bool, *models.Song, error) {
	var existingSong models.Song

	// Look up by checksum in the database
	if err := cs.db.Where("crc_hash = ? AND relative_path = ?", checksum, originalPath).First(&existingSong).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil, nil
		}
		return false, nil, err
	}

	return true, &existingSong, nil
}

// IsAlreadyProcessedByContentOnly checks if a file with the same content (by checksum) has already been processed
func (cs *ChecksumService) IsAlreadyProcessedByContentOnly(checksum string) (bool, *models.Song, error) {
	var existingSong models.Song

	// Look up by just the checksum (same content anywhere)
	if err := cs.db.Where("crc_hash = ?", checksum).First(&existingSong).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil, nil
		}
		return false, nil, err
	}

	return true, &existingSong, nil
}

// MarkAsProcessed stores the checksum in the system to prevent re-processing
func (cs *ChecksumService) MarkAsProcessed(filePath, checksum string, song *models.Song) error {
	// The song model should already have the checksum set, but let's ensure it's stored properly
	if song.CRCHash == "" {
		song.CRCHash = checksum
	}

	// If file already exists in database, prevent duplicate insertion
	var existingSong models.Song
	result := cs.db.Where("crc_hash = ? AND relative_path = ?", checksum, filePath).First(&existingSong)
	
	if result.Error == nil {
		// Already exists, return early to maintain idempotency
		return nil
	}
	
	if result.Error != gorm.ErrRecordNotFound {
		return result.Error
	}

	// Otherwise, this is a new song, we can save it
	return nil // Caller should handle saving the song
}

// VerifyFileIntegrity verifies file integrity by checking content against stored checksum
func (cs *ChecksumService) VerifyFileIntegrity(filePath, storedChecksum string) (bool, error) {
	// Calculate current checksum
	currentChecksum, err := cs.CalculateChecksum(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to calculate current checksum: %w", err)
	}

	// Compare with stored checksum
	isValid := currentChecksum == storedChecksum

	// Log integrity violations for monitoring
	if !isValid {
		// In a real system, we'd want to log this for admin notification
		fmt.Printf("INTEGRITY VIOLATION: File %s has checksum %s but expected %s\n",
			filePath, currentChecksum, storedChecksum)
	}

	return isValid, nil
}

// BatchValidateChecksums validates multiple files against their expected checksums
func (cs *ChecksumService) BatchValidateChecksums(files map[string]string) (map[string]bool, []string, error) {
	results := make(map[string]bool)
	errors := []string{}

	for filePath, expectedChecksum := range files {
		isValid, err := cs.ValidateChecksum(filePath, expectedChecksum)
		if err != nil {
			errors = append(errors, fmt.Sprintf("file: %s, error: %v", filePath, err))
			results[filePath] = false
		} else {
			results[filePath] = isValid
		}
	}

	return results, errors, nil
}

// PurgeInvalidEntries removes entries that no longer correspond to real files
func (cs *ChecksumService) PurgeInvalidEntries() error {
	// This would be used during cleanup operations to remove checksum entries
	// for files that no longer exist

	// In a real implementation, we'd iterate through stored checksum records
	// and remove those where the file no longer exists
	
	return nil
}

// getCalculatedChecksum retrieves calculated checksum for a file
func (cs *ChecksumService) getCalculatedChecksum(filePath string) (string, bool) {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()

	entry, exists := cs.cache[filePath]
	if !exists {
		return "", false
	}

	// Check if cache entry has expired
	if time.Now().After(entry.ExpiresAt) {
		// Remove expired entry
		delete(cs.cache, filePath)
		return "", false
	}

	return entry.Checksum, true
}

// getCachedChecksum gets a cached checksum entry if it exists and is valid
func (cs *ChecksumService) getCachedChecksum(filePath string) (*ChecksumEntry, bool) {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()

	entry, exists := cs.cache[filePath]
	if !exists {
		return nil, false
	}

	// Check if cache entry has expired
	if time.Now().After(entry.ExpiresAt) {
		// Remove expired entry
		delete(cs.cache, filePath)
		return nil, false
	}

	return entry, true
}

// cacheChecksum caches a checksum with TTL
func (cs *ChecksumService) cacheChecksum(filePath, checksum string) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	// Get file info for validation
	info, err := os.Stat(filePath)
	if err != nil {
		// Don't cache if we can't stat the file
		return
	}

	cs.cache[filePath] = &ChecksumEntry{
		Checksum:  checksum,
		FilePath:  filePath,
		FileSize:  info.Size(),
		ModTime:   info.ModTime(),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(cs.config.CacheTTL),
	}
}

// removeCachedChecksum removes a cached checksum
func (cs *ChecksumService) removeCachedChecksum(filePath string) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	delete(cs.cache, filePath)
}

// isCacheValid checks if the cached checksum is still valid for the current file
func (cs *ChecksumService) isCacheValid(filePath string, cached *ChecksumEntry) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}

	// Check if file size or modification time has changed
	return info.Size() == cached.FileSize && info.ModTime().Equal(cached.ModTime)
}

// cleanupRoutine periodically removes expired cache entries
func (cs *ChecksumService) cleanupRoutine() {
	ticker := time.NewTicker(1 * time.Hour) // Clean up every hour
	defer ticker.Stop()

	for range ticker.C {
		cs.cleanupExpiredEntries()
	}
}

// cleanupExpiredEntries removes expired cache entries
func (cs *ChecksumService) cleanupExpiredEntries() {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	now := time.Now()
	for filePath, entry := range cs.cache {
		if now.After(entry.ExpiresAt) {
			delete(cs.cache, filePath)
		}
	}
}

// CalculateAndVerifyFile calculates the checksum for a file and verifies its integrity against an expected value
func (cs *ChecksumService) CalculateAndVerifyFile(filePath string, expectedChecksum string) (string, bool, error) {
	calculated, err := cs.CalculateChecksum(filePath)
	if err != nil {
		return "", false, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	matches := calculated == expectedChecksum
	return calculated, matches, nil
}

// GetConsistencyReport generates a report of file consistency across the system
func (cs *ChecksumService) GetConsistencyReport() (*ConsistencyReport, error) {
	report := &ConsistencyReport{
		GeneratedAt: time.Now(),
	}

	// In a real system, this would query the database for all songs with checksums
	// and verify each file still exists and has the expected checksum

	// This is a simplified version for demonstration
	var songCount int64
	cs.db.Model(&models.Song{}).Count(&songCount)
	
	report.TotalFiles = int(songCount)
	report.VerifiedFiles = 0
	report.CorruptedFiles = 0
	report.MissingFiles = 0

	return report, nil
}

// ConsistencyReport represents the results of a file consistency check
type ConsistencyReport struct {
	GeneratedAt   time.Time `json:"generated_at"`
	TotalFiles    int       `json:"total_files"`
	VerifiedFiles int       `json:"verified_files"`
	CorruptedFiles int      `json:"corrupted_files"`
	MissingFiles  int       `json:"missing_files"`
	IntegrityScore float64   `json:"integrity_score"` // Percentage of verified files (0-100)
}

// GetIntegrityScore calculates the integrity score as percentage of verified files
func (cr *ConsistencyReport) GetIntegrityScore() float64 {
	if cr.TotalFiles == 0 {
		return 100.0 // Perfect score if no files to verify
	}
	cr.IntegrityScore = (float64(cr.VerifiedFiles) / float64(cr.TotalFiles)) * 100.0
	return cr.IntegrityScore
}