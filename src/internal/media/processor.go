package media

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
	"melodee/internal/directory"
	"melodee/internal/models"
)

// ProcessingStage represents the stage of media processing
type ProcessingStage string

const (
	InboundStage    ProcessingStage = "inbound"
	StagingStage    ProcessingStage = "staging"
	ProductionStage ProcessingStage = "production"
)

// ProcessingConfig holds configuration for the processing workflow
type ProcessingConfig struct {
	InboundDir           string `mapstructure:"inbound_dir"`
	StagingDir           string `mapstructure:"staging_dir"`
	ProductionDir        string `mapstructure:"production_dir"`
	ConcurrencyInbound   int    `mapstructure:"concurrency_inbound"`   // Workers for inbound validation
	ConcurrencyStaging   int    `mapstructure:"concurrency_staging"`   // Workers for staging promotion
	ConcurrencyTranscode int    `mapstructure:"concurrency_transcode"` // Workers for transcoding
	MaxRetries           int    `mapstructure:"max_retries"`           // Max retries per file
	RetryDelayBase       int    `mapstructure:"retry_delay_base"`      // Base delay in seconds for retries
	ChecksumStoragePath  string `mapstructure:"checksum_storage_path"` // Path to store checksums and mtime
}

// DefaultProcessingConfig returns the default processing configuration
func DefaultProcessingConfig() *ProcessingConfig {
	return &ProcessingConfig{
		InboundDir:           "/melodee/inbound",
		StagingDir:           "/melodee/staging",
		ProductionDir:        "/melodee/storage",
		ConcurrencyInbound:   4,
		ConcurrencyStaging:   2,
		ConcurrencyTranscode: 2,
		MaxRetries:           3,
		RetryDelayBase:       2, // 2s, 4s, 8s for exponential backoff
		ChecksumStoragePath:  "/melodee/data/checksums",
	}
}

// MediaProcessor handles the complete media processing workflow
type MediaProcessor struct {
	config            *ProcessingConfig
	db                *gorm.DB
	directoryService  *directory.PathTemplateResolver
	quarantineService *QuarantineService
	mediaValidator    *MediaFileValidator
	ffmpegProcessor   *FFmpegProcessor
	checksumService   *ChecksumService
}

// NewMediaProcessor creates a new media processor
func NewMediaProcessor(
	config *ProcessingConfig,
	db *gorm.DB,
	directoryService *directory.PathTemplateResolver,
	quarantineService *QuarantineService,
	mediaValidator *MediaFileValidator,
	ffmpegProcessor *FFmpegProcessor,
	checksumService *ChecksumService,
) *MediaProcessor {
	if config == nil {
		config = DefaultProcessingConfig()
	}

	return &MediaProcessor{
		config:            config,
		db:                db,
		directoryService:  directoryService,
		quarantineService: quarantineService,
		mediaValidator:    mediaValidator,
		ffmpegProcessor:   ffmpegProcessor,
		checksumService:   checksumService,
	}
}

// ProcessInbound processes files from the inbound directory
func (mp *MediaProcessor) ProcessInbound(force bool) error {
	// Get all media files in the inbound directory
	files, err := mp.getMediaFilesInDirectory(mp.config.InboundDir)
	if err != nil {
		return fmt.Errorf("failed to get files in inbound directory: %w", err)
	}

	for _, file := range files {
		if err := mp.processInboundFile(file, force); err != nil {
			// Log error but continue processing other files
			fmt.Printf("Failed to process inbound file %s: %v\n", file.Path, err)
			continue
		}
	}

	return nil
}

// processInboundFile validates and processes a single inbound file
func (mp *MediaProcessor) processInboundFile(file MediaFile, force bool) error {
	// 1. Validate file integrity and format
	if err := mp.mediaValidator.Validate(file.Path); err != nil {
		return mp.moveToQuarantine(file.Path, ValidationBounds, fmt.Sprintf("File validation failed: %v", err), 0)
	}

	// 2. Calculate checksum using the checksum service
	checksum, err := mp.checksumService.CalculateChecksum(file.Path)
	if err != nil {
		return mp.moveToQuarantine(file.Path, ChecksumMismatch, fmt.Sprintf("Checksum calculation failed: %v", err), 0)
	}

	// 3. Extract metadata using embedded libraries
	metadata, err := mp.extractMetadata(file.Path)
	if err != nil {
		return mp.moveToQuarantine(file.Path, TagParseError, fmt.Sprintf("Metadata extraction failed: %v", err), 0)
	}
	// Update metadata with the calculated checksum
	metadata.Checksum = checksum

	// 4. Normalize metadata according to system rules
	normalizedMetadata := mp.normalizeMetadata(metadata)

	// 5. Check if file has been processed before (idempotency check) using checksum service
	if !force {
		isProcessed, _, err := mp.checksumService.IsAlreadyProcessedByContentOnly(checksum)
		if err != nil {
			return fmt.Errorf("failed to check if file with same content is already processed: %w", err)
		}
		if isProcessed {
			fmt.Printf("File with same content as %s already processed (checksum: %s), skipping\n", file.Path, checksum)
			return nil
		}
	}

	// 6. Determine destination in staging area using directory code
	stagingPath, err := mp.calculateStagingPath(normalizedMetadata)
	if err != nil {
		return mp.moveToQuarantine(file.Path, PathSafety, fmt.Sprintf("Failed to calculate staging path: %v", err), 0)
	}

	// Ensure staging directory exists
	if err := os.MkdirAll(filepath.Dir(stagingPath), 0755); err != nil {
		return fmt.Errorf("failed to create staging directory: %w", err)
	}

	// 7. Move file to staging with normalized structure
	if err := mp.moveFile(file.Path, stagingPath); err != nil {
		return fmt.Errorf("failed to move file to staging: %w", err)
	}

	// 8. Create database records in staging state
	if err := mp.createStagingRecords(normalizedMetadata, stagingPath); err != nil {
		// Move file back to inbound if DB record creation fails
		if rollbackErr := mp.moveFile(stagingPath, file.Path); rollbackErr != nil {
			fmt.Printf("Failed to rollback file move: %v (original error: %v)\n", rollbackErr, err)
		}
		return fmt.Errorf("failed to create staging records: %w", err)
	}

	// 9. Mark file as processed using checksum service
	if err := mp.checksumService.MarkAsProcessed(stagingPath, checksum, nil); err != nil {
		fmt.Printf("Warning: failed to mark file as processed: %v\n", err)
	}

	return nil
}

// getMediaFilesInDirectory returns all media files in the specified directory
func (mp *MediaProcessor) getMediaFilesInDirectory(dir string) ([]MediaFile, error) {
	var files []MediaFile

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check if it's a media file based on extension
		if mp.isMediaFile(path) {
			files = append(files, MediaFile{
				Path:    path,
				Name:    info.Name(),
				Size:    info.Size(),
				ModTime: info.ModTime(),
				Ext:     strings.ToLower(filepath.Ext(path)),
			})
		}

		return nil
	})

	return files, err
}

// isMediaFile checks if a file is a media file based on extension
func (mp *MediaProcessor) isMediaFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	mediaExts := []string{".mp3", ".flac", ".ogg", ".opus", ".m4a", ".mp4", ".aac", ".wma", ".wav", ".aiff", ".ape", ".wv", ".dsf", ".cda"}

	for _, mediaExt := range mediaExts {
		if ext == mediaExt {
			return true
		}
	}

	return false
}

// MediaFile represents a media file to be processed
type MediaFile struct {
	Path    string
	Name    string
	Size    int64
	ModTime time.Time
	Ext     string
}

// extractMetadata extracts metadata from the file
func (mp *MediaProcessor) extractMetadata(filePath string) (*MediaMetadata, error) {
	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	// Calculate checksum
	checksum, err := mp.calculateChecksum(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// In a real implementation, we would call an actual metadata extractor
	// For now, return a basic metadata struct
	return &MediaMetadata{
		FilePath: filePath,
		Size:     info.Size(),
		ModTime:  info.ModTime(),
		Checksum: checksum,
		Name:     filepath.Base(filePath),
		// Other metadata fields would be extracted here
	}, nil
}

// calculateChecksum calculates the SHA256 checksum of a file
func (mp *MediaProcessor) calculateChecksum(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// normalizeMetadata normalizes metadata according to system rules
func (mp *MediaProcessor) normalizeMetadata(metadata *MediaMetadata) *MediaMetadata {
	// This would implement normalization rules from the documentation
	// For now, return the metadata as is
	return metadata
}

// calculateStagingPath calculates the staging path for a file
func (mp *MediaProcessor) calculateStagingPath(metadata *MediaMetadata) (string, error) {
	// For now, use a simple path structure
	// In a real implementation, this would involve artist/album metadata
	artistCode := "UNKNOWN" // Would come from metadata
	albumName := "UNKNOWN"  // Would come from metadata

	stagingPath := filepath.Join(mp.config.StagingDir, artistCode, albumName, filepath.Base(metadata.FilePath))
	return stagingPath, nil
}

// moveFile moves a file from source to destination
func (mp *MediaProcessor) moveFile(src, dst string) error {
	// First try to rename (move) the file
	if err := os.Rename(src, dst); err != nil {
		// If rename fails (e.g., different filesystems), copy then delete
		if err := mp.copyFile(src, dst); err != nil {
			return fmt.Errorf("failed to copy file: %w", err)
		}
		// Remove the original file after copying
		if err := os.Remove(src); err != nil {
			return fmt.Errorf("failed to remove original file: %w", err)
		}
	}
	return nil
}

// copyFile copies a file from source to destination
func (mp *MediaProcessor) copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, input, 0644)
}

// createStagingRecords creates database records for staging files
func (mp *MediaProcessor) createStagingRecords(metadata *MediaMetadata, stagingPath string) error {
	// This would create actual staging records in the database
	// For now, we'll create placeholder logic

	// In a real system, we would create staging records that track the file
	// until it's promoted to production
	stagingRecord := &models.Track{
		Name: metadata.Name,
		// Other fields would be populated from metadata
		Directory:    filepath.Dir(stagingPath),
		FileName:     filepath.Base(stagingPath),
		RelativePath: strings.TrimPrefix(stagingPath, mp.config.StagingDir),
		CRCHash:      metadata.Checksum,
		CreatedAt:    time.Now(),
	}

	// Create the staging record in the database
	if err := mp.db.Create(stagingRecord).Error; err != nil {
		return fmt.Errorf("failed to create staging record: %w", err)
	}

	return nil
}

// isAlreadyProcessed checks if a file has already been processed
func (mp *MediaProcessor) isAlreadyProcessed(filePath, checksum string) (bool, error) {
	// In a real system, this would check against a processed files table
	// For now, return false to indicate the file hasn't been processed
	return false, nil
}

// markAsProcessed marks a file as processed in the system
func (mp *MediaProcessor) markAsProcessed(filePath, checksum string) error {
	// In a real system, this would update a processed files table
	// For now, this is a no-op
	return nil
}

// moveToQuarantine moves a problematic file to quarantine
func (mp *MediaProcessor) moveToQuarantine(filePath string, reason QuarantineReason, message string, libraryID int32) error {
	return mp.quarantineService.QuarantineFile(filePath, reason, message, libraryID)
}

// MediaMetadata holds metadata about a media file
type MediaMetadata struct {
	FilePath           string
	Size               int64
	ModTime            time.Time
	Checksum           string
	Name               string
	Artist             string
	Album              string
	Title              string
	TrackNumber        int
	DiscNumber         int
	Genre              string
	Year               int
	Duration           time.Duration
	BitRate            int
	SampleRate         int
	Channels           int
	Format             string
	ArtworkPath        string
	HasEmbeddedArtwork bool
	CueSheetPath       string
	HasCueSheet        bool
}

// ProcessStaging promotes approved staging content to production
func (mp *MediaProcessor) ProcessStaging() error {
	// Get all items in staging that are ready for production
	var stagingItems []models.Track // This would be a proper staging model
	if err := mp.db.Where("relative_path LIKE ?", "staging/%").Find(&stagingItems).Error; err != nil {
		return fmt.Errorf("failed to retrieve staging content: %w", err)
	}

	for _, item := range stagingItems {
		if err := mp.promoteStagingItem(item); err != nil {
			// Log error but continue with other items
			fmt.Printf("Failed to promote staging item %s: %v\n", item.RelativePath, err)
			continue
		}
	}

	return nil
}

// promoteStagingItem promotes a single staging item to production
func (mp *MediaProcessor) promoteStagingItem(item models.Track) error {
	// Get the staging file path
	stagingPath := filepath.Join(mp.config.StagingDir, item.RelativePath)

	// Validate file integrity by re-checking checksum using the checksum service
	isValid, err := mp.checksumService.VerifyFileIntegrity(stagingPath, item.CRCHash)
	if err != nil {
		return mp.moveToQuarantine(stagingPath, ChecksumMismatch, fmt.Sprintf("Failed to verify file integrity: %v", err), 0)
	}

	if !isValid {
		currentChecksum, err := mp.checksumService.CalculateChecksum(stagingPath)
		if err != nil {
			return mp.moveToQuarantine(stagingPath, ChecksumMismatch, fmt.Sprintf("Failed to calculate current checksum: %v", err), 0)
		}

		return mp.moveToQuarantine(stagingPath, ChecksumMismatch, fmt.Sprintf("Checksum mismatch: expected %s, got %s", item.CRCHash, currentChecksum), 0)
	}

	// Determine appropriate production library based on artist directory code
	productionLibrary, err := mp.selectProductionLibrary(item)
	if err != nil {
		return fmt.Errorf("failed to select production library: %w", err)
	}

	// Calculate final production path using directory code and template
	productionPath, err := mp.calculateProductionPath(item, productionLibrary)
	if err != nil {
		return fmt.Errorf("failed to calculate production path: %w", err)
	}

	// Ensure production directory exists
	if err := os.MkdirAll(filepath.Dir(productionPath), 0755); err != nil {
		return fmt.Errorf("failed to create production directory: %w", err)
	}

	// Move file from staging to production
	if err := mp.moveFile(stagingPath, productionPath); err != nil {
		return fmt.Errorf("failed to move file to production: %w", err)
	}

	// Create or update production database records
	productionID, err := mp.createProductionRecords(item, productionLibrary.ID, productionPath)
	if err != nil {
		// If DB creation fails, move the file back to staging
		if rollbackErr := mp.moveFile(productionPath, stagingPath); rollbackErr != nil {
			fmt.Printf("Failed to rollback file move: %v (original error: %v)\n", rollbackErr, err)
		}
		return fmt.Errorf("failed to create production records: %w", err)
	}

	// Update staging record to show it's been promoted
	if err := mp.markStagingItemAsPromoted(item.ID, productionID); err != nil {
		return fmt.Errorf("failed to update staging record: %w", err)
	}

	return nil
}

// selectProductionLibrary selects the appropriate production library for an item
func (mp *MediaProcessor) selectProductionLibrary(item models.Track) (*models.Library, error) {
	// Get the artist associated with this item to determine directory code
	var song models.Track
	if err := mp.db.Preload("Album.Artist").First(&song, item.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to find song: %w", err)
	}

	if song.Album == nil || song.Album.Artist == nil {
		return nil, fmt.Errorf("song has no associated artist")
	}

	artist := song.Album.Artist

	// Select library based on directory code
	return mp.selectLibraryByDirectoryCode(artist.DirectoryCode)
}

// LibrarySelectionConfig defines rules for selecting production libraries
type LibrarySelectionConfig struct {
	LibraryStrategy string              `mapstructure:"strategy"` // "hash", "round_robin", "directory_code", "size_based"
	LoadBalancing   LoadBalancingConfig `mapstructure:"load_balancing"`
	DirectoryRules  map[string]string   `mapstructure:"directory_rules"` // Artist directory code to library mapping
	SizeThresholds  SizeThresholdConfig `mapstructure:"size_thresholds"`
	HashSalt        string              `mapstructure:"hash_salt"`
}

// LoadBalancingConfig holds load balancing configuration
type LoadBalancingConfig struct {
	Enabled                 bool `mapstructure:"enabled"`
	ThresholdPercentage     int  `mapstructure:"threshold_percentage"`      // Move to next library when current is this % full
	StopThresholdPercentage int  `mapstructure:"stop_threshold_percentage"` // Stop allocations at this %
}

// SizeThresholdConfig holds size threshold configuration
type SizeThresholdConfig struct {
	MaxSizePerLibrary int64 `mapstructure:"max_size_per_library"` // Max size per library in bytes
}

// DefaultLibrarySelectionConfig returns default library selection configuration
func DefaultLibrarySelectionConfig() *LibrarySelectionConfig {
	return &LibrarySelectionConfig{
		LibraryStrategy: "directory_code",
		LoadBalancing: LoadBalancingConfig{
			Enabled:                 true,
			ThresholdPercentage:     80,
			StopThresholdPercentage: 90,
		},
		DirectoryRules: make(map[string]string),
		SizeThresholds: SizeThresholdConfig{
			MaxSizePerLibrary: 1073741824000, // 1TB
		},
		HashSalt: "melodee",
	}
}

// selectLibraryByDirectoryCode uses configured rules to map directory codes to libraries
func (mp *MediaProcessor) selectLibraryByDirectoryCode(directoryCode string) (*models.Library, error) {
	config := DefaultLibrarySelectionConfig()

	// Check if there's a specific rule for this directory code
	if libraryName, exists := config.DirectoryRules[directoryCode]; exists {
		var library models.Library
		if err := mp.db.Where("name = ? AND type = ?", libraryName, "production").First(&library).Error; err != nil {
			return nil, fmt.Errorf("configured library not found: %w", err)
		}
		return &library, nil
	}

	// Check for range rules (e.g., "A-C" -> library)
	for rangeRule, libraryName := range config.DirectoryRules {
		if mp.matchesRange(directoryCode, rangeRule) {
			var library models.Library
			if err := mp.db.Where("name = ? AND type = ?", libraryName, "production").First(&library).Error; err != nil {
				return nil, fmt.Errorf("configured library not found: %w", err)
			}
			return &library, nil
		}
	}

	// Fallback: Use strategy-based selection
	switch config.LibraryStrategy {
	case "hash":
		return mp.selectByHash(directoryCode)
	case "round_robin":
		return mp.selectByRoundRobin()
	case "directory_code":
		return mp.selectByFirstChar(directoryCode)
	case "size_based":
		return mp.selectBySize()
	default:
		return mp.selectByDefaultStrategy()
	}
}

// matchesRange checks if a directory code matches a range rule (e.g., "A-C")
func (mp *MediaProcessor) matchesRange(directoryCode, rangeRule string) bool {
	if len(rangeRule) < 3 || rangeRule[1] != '-' {
		return false
	}

	startChar := rune(rangeRule[0])
	endChar := rune(rangeRule[2])

	if len(directoryCode) == 0 {
		return false
	}

	firstChar := rune(directoryCode[0])

	return firstChar >= startChar && firstChar <= endChar
}

// selectByHash uses consistent hashing to distribute artists across libraries
func (mp *MediaProcessor) selectByHash(directoryCode string) (*models.Library, error) {
	availableLibraries, err := mp.getAvailableProductionLibraries()
	if err != nil {
		return nil, fmt.Errorf("failed to get available libraries: %w", err)
	}

	if len(availableLibraries) == 0 {
		return nil, fmt.Errorf("no available production libraries")
	}

	// Use directory code + salt for consistent hashing
	hashInput := directoryCode + DefaultLibrarySelectionConfig().HashSalt
	hashValue := mp.simpleHash(hashInput)
	index := int(hashValue) % len(availableLibraries)
	return &availableLibraries[index], nil
}

// selectByFirstChar selects library based on the first character of directory code
func (mp *MediaProcessor) selectByFirstChar(directoryCode string) (*models.Library, error) {
	availableLibraries, err := mp.getAvailableProductionLibraries()
	if err != nil {
		return nil, fmt.Errorf("failed to get available libraries: %w", err)
	}

	if len(availableLibraries) == 0 {
		return nil, fmt.Errorf("no available production libraries")
	}

	if len(directoryCode) == 0 {
		return &availableLibraries[0], nil
	}

	// Distribute based on first character
	firstChar := directoryCode[0]
	index := int(firstChar) % len(availableLibraries)
	return &availableLibraries[index], nil
}

// selectByRoundRobin selects libraries in a round-robin fashion
func (mp *MediaProcessor) selectByRoundRobin() (*models.Library, error) {
	// For round-robin, we'd need to track the current position
	// For now, we'll just get the first available library
	availableLibraries, err := mp.getAvailableProductionLibraries()
	if err != nil {
		return nil, fmt.Errorf("failed to get available libraries: %w", err)
	}

	if len(availableLibraries) == 0 {
		return nil, fmt.Errorf("no available production libraries")
	}

	// In a real system, we'd track the last used index
	return &availableLibraries[0], nil
}

// selectBySize selects the library with the most available space
func (mp *MediaProcessor) selectBySize() (*models.Library, error) {
	availableLibraries, err := mp.getAvailableProductionLibraries()
	if err != nil {
		return nil, fmt.Errorf("failed to get available libraries: %w", err)
	}

	if len(availableLibraries) == 0 {
		return nil, fmt.Errorf("no available production libraries")
	}

	var selectedLibrary *models.Library
	var minSize int64 = 1<<63 - 1 // Max int64

	for i := range availableLibraries {
		// Calculate current size of library (simplified)
		size, err := mp.calculateLibrarySize(&availableLibraries[i])
		if err != nil {
			fmt.Printf("Warning: failed to calculate size for library %s: %v\n", availableLibraries[i].Name, err)
			continue
		}

		if size < minSize {
			minSize = size
			selectedLibrary = &availableLibraries[i]
		}
	}

	if selectedLibrary == nil {
		return nil, fmt.Errorf("no suitable library found")
	}

	return selectedLibrary, nil
}

// selectByDefaultStrategy selects using the default strategy (directory code-based)
func (mp *MediaProcessor) selectByDefaultStrategy() (*models.Library, error) {
	// Use directory code as default
	// This is a simplified version of selectByFirstChar
	availableLibraries, err := mp.getAvailableProductionLibraries()
	if err != nil {
		return nil, fmt.Errorf("failed to get available libraries: %w", err)
	}

	if len(availableLibraries) == 0 {
		return nil, fmt.Errorf("no available production libraries")
	}

	return &availableLibraries[0], nil
}

// getAvailableProductionLibraries returns unlocked production libraries
func (mp *MediaProcessor) getAvailableProductionLibraries() ([]models.Library, error) {
	var libraries []models.Library
	if err := mp.db.Where("type = ? AND is_locked = ?", "production", false).Find(&libraries).Error; err != nil {
		return nil, fmt.Errorf("failed to get production libraries: %w", err)
	}
	return libraries, nil
}

// calculateLibrarySize calculates the approximate size of a library (simplified version)
func (mp *MediaProcessor) calculateLibrarySize(library *models.Library) (int64, error) {
	// In a real implementation, this would calculate actual disk usage
	// For now, we'll return a placeholder value based on file count
	var count int64
	if err := mp.db.Model(&models.Track{}).Where("directory LIKE ?", library.Path+"%").Count(&count).Error; err != nil {
		return 0, err
	}

	// Assume average file size is 5MB
	return count * 5 * 1024 * 1024, nil
}

// simpleHash provides a simple string hashing function
func (mp *MediaProcessor) simpleHash(s string) uint32 {
	var hash uint32 = 5381
	for _, c := range s {
		hash = ((hash << 5) + hash) + uint32(c)
	}
	return hash
}

// calculateProductionPath calculates the production path for an item
func (mp *MediaProcessor) calculateProductionPath(item models.Track, library *models.Library) (string, error) {
	// For now, use a simple mapping
	// In a real system, this would use the directory service with artist directory codes
	productionPath := filepath.Join(library.Path, item.Directory, item.FileName)
	return productionPath, nil
}

// createProductionRecords creates production database records
func (mp *MediaProcessor) createProductionRecords(item models.Track, libraryID int32, productionPath string) (int64, error) {
	// Create a new production song record based on the staging item
	productionSong := &models.Track{
		Name:         item.Name,
		Directory:    filepath.Dir(productionPath),
		FileName:     filepath.Base(productionPath),
		RelativePath: strings.TrimPrefix(productionPath, mp.config.ProductionDir),
		CRCHash:      item.CRCHash,
		CreatedAt:    time.Now(),
		// Other fields would be copied from staging item
	}

	if err := mp.db.Create(productionSong).Error; err != nil {
		return 0, fmt.Errorf("failed to create production record: %w", err)
	}

	return productionSong.ID, nil
}

// markStagingItemAsPromoted marks a staging item as promoted
func (mp *MediaProcessor) markStagingItemAsPromoted(stagingID, productionID int64) error {
	// Update the staging item to mark it as promoted
	// In a real system, this might involve moving the record to a different table or updating status
	return mp.db.Model(&models.Track{}).Where("id = ?", stagingID).Update("relative_path", fmt.Sprintf("promoted/%d", stagingID)).Error
}

// ProcessWithRetry attempts processing with configurable retry logic
func (mp *MediaProcessor) ProcessWithRetry(filePath string, maxRetries int, processFunc func(string) error) error {
	var err error
	for attempt := 0; attempt < maxRetries; attempt++ {
		err = processFunc(filePath)
		if err == nil {
			return nil
		}

		fmt.Printf("Processing failed for %s (attempt %d): %v\n", filePath, attempt+1, err)

		// Exponential backoff (2s, 4s, 8s, etc.)
		delay := time.Duration(mp.config.RetryDelayBase) * time.Second * time.Duration(1<<uint(attempt))
		time.Sleep(delay)
	}

	return fmt.Errorf("processing failed after %d attempts: %w", maxRetries, err)
}
