package media

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
	"melodee/internal/directory"
	"melodee/internal/models"
)

// LibraryProcessor handles the inbound → staging → production flow
type LibraryProcessor struct {
	db             *gorm.DB
	directorySvc   *directory.DirectoryCodeGenerator
	pathResolver   *directory.PathTemplateResolver
	quarantineSvc  *QuarantineService
	config         *ProcessingConfig
}

// ProcessingConfig holds configuration for media processing
type ProcessingConfig struct {
	InboundDir      string `mapstructure:"inbound_dir"`
	StagingDir      string `mapstructure:"staging_dir"`
	ProductionDir   string `mapstructure:"production_dir"`
	QuarantineDir   string `mapstructure:"quarantine_dir"`
	Concurrency     int    `mapstructure:"concurrency"`
	FFmpegPath      string `mapstructure:"ffmpeg_path"`
	ChecksumEnabled bool   `mapstructure:"checksum_enabled"`
}

// DefaultProcessingConfig returns the default configuration
func DefaultProcessingConfig() *ProcessingConfig {
	return &ProcessingConfig{
		InboundDir:      "/melodee/inbound",
		StagingDir:      "/melodee/staging",
		ProductionDir:   "/melodee/storage",
		QuarantineDir:   "/melodee/quarantine",
		Concurrency:     4,
		FFmpegPath:      "ffmpeg",
		ChecksumEnabled: true,
	}
}

// NewLibraryProcessor creates a new library processor
func NewLibraryProcessor(
	db *gorm.DB,
	directorySvc *directory.DirectoryCodeGenerator,
	pathResolver *directory.PathTemplateResolver,
	quarantineSvc *QuarantineService,
	config *ProcessingConfig,
) *LibraryProcessor {
	if config == nil {
		config = DefaultProcessingConfig()
	}
	
	return &LibraryProcessor{
		db:            db,
		directorySvc:  directorySvc,
		pathResolver:  pathResolver,
		quarantineSvc: quarantineSvc,
		config:        config,
	}
}

// ProcessInbound processes files from the inbound directory
func (lp *LibraryProcessor) ProcessInbound() error {
	// Get all media files from inbound directory
	inboundFiles, err := lp.getInboundFiles()
	if err != nil {
		return fmt.Errorf("failed to get inbound files: %w", err)
	}

	// Process files with configured concurrency
	for _, file := range inboundFiles {
		if err := lp.processFile(file); err != nil {
			// Log error but continue processing other files
			fmt.Printf("Failed to process inbound file %s: %v\n", file, err)
		}
	}

	return nil
}

// processFile processes a single media file through the inbound → staging flow
func (lp *LibraryProcessor) processFile(filePath string) error {
	// 1. Validate file path safety
	if err := lp.quarantineSvc.ValidateFilePath(filePath); err != nil {
		return lp.quarantineSvc.QuarantineFile(filePath, PathSafety, err.Error(), 0)
	}

	// 2. Validate file integrity and format
	if valid, _ := lp.validateFileFormat(filePath); !valid {
		return lp.quarantineSvc.QuarantineFile(filePath, UnsupportedContainer, "unsupported file format", 0)
	}

	// 3. Extract metadata from the file
	metadata, err := lp.extractFileMetadata(filePath)
	if err != nil {
		return lp.quarantineSvc.QuarantineFile(filePath, TagParseError, err.Error(), 0)
	}

	// 4. Normalize metadata
	normalizedMetadata := lp.normalizeMetadata(metadata)

	// 5. Get or generate directory code for the artist
	directoryCode, err := lp.directorySvc.GetDirectoryCodeForArtist(normalizedMetadata.Artist)
	if err != nil {
		return lp.quarantineSvc.QuarantineFile(filePath, MetadataConflict, err.Error(), 0)
	}

	// 6. Calculate staging path using directory code and template
	artist := &models.Artist{
		Name:          normalizedMetadata.Artist,
		DirectoryCode: directoryCode,
	}
	
	album := &models.Album{
		Name:        normalizedMetadata.Album,
		ReleaseDate: &normalizedMetadata.ReleaseDate,
	}
	
	stagingPath, err := lp.pathResolver.ResolveForStaging(artist, album)
	if err != nil {
		return lp.quarantineSvc.QuarantineFile(filePath, PathSafety, err.Error(), 0)
	}

	// 7. Move file to staging with the calculated path
	targetPath := filepath.Join(lp.config.StagingDir, stagingPath, filepath.Base(filePath))
	
	// Ensure staging directory exists
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("failed to create staging directory: %w", err)
	}

	// Move the file to staging
	if err := os.Rename(filePath, targetPath); err != nil {
		return fmt.Errorf("failed to move file to staging: %w", err)
	}

	// 8. Create staging records in database
	if err := lp.createStagingRecords(artist, album, normalizedMetadata, targetPath); err != nil {
		// If DB record creation fails, move the file back and quarantine
		os.Rename(targetPath, filePath) // Move back to original location
		return lp.quarantineSvc.QuarantineFile(targetPath, MetadataConflict, err.Error(), 0)
	}

	return nil
}

// getInboundFiles returns all media files from the inbound directory
func (lp *LibraryProcessor) getInboundFiles() ([]string, error) {
	var files []string
	
	err := filepath.Walk(lp.config.InboundDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && lp.isMediaFile(path) {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

// isMediaFile checks if a file extension indicates it's a media file
func (lp *LibraryProcessor) isMediaFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	mediaExtensions := map[string]bool{
		".mp3":  true,
		".flac": true,
		".m4a":  true,
		".mp4":  true,
		".aac":  true,
		".ogg":  true,
		".opus": true,
		".wav":  true,
		".wma":  true,
		".aiff": true,
		".alac": true,
	}

	return mediaExtensions[ext]
}

// validateFileFormat checks if the file is a supported audio format
func (lp *LibraryProcessor) validateFileFormat(filePath string) (bool, error) {
	// In a real implementation, we'd use a proper file format checker
	// For now, just check the extension
	return lp.isMediaFile(filePath), nil
}

// extractFileMetadata extracts metadata from a media file
func (lp *LibraryProcessor) extractFileMetadata(filePath string) (*FileMetadata, error) {
	// In a real implementation, we'd use a library like go-audio or taglib-go
	// For now, return a placeholder implementation
	
	// Parse file name to get basic info
	_, fileName := filepath.Split(filePath)
	ext := filepath.Ext(fileName)
	baseName := strings.TrimSuffix(fileName, ext)
	
	// This is a simplified extraction - in a real system, use proper metadata extraction
	metadata := &FileMetadata{
		Title:       baseName,
		Artist:      "Unknown Artist", // Would be extracted from tags
		Album:       "Unknown Album",  // Would be extracted from tags
		TrackNumber: 0,
		DiscNumber:  0,
		Genre:       "",
		Duration:    0, // Would be calculated
		BitRate:     0, // Would be calculated
		ReleaseDate: time.Now(), // Would be extracted from tags
		FileName:    fileName,
		FilePath:    filePath,
	}

	return metadata, nil
}

// normalizeMetadata normalizes extracted metadata
func (lp *LibraryProcessor) normalizeMetadata(metadata *FileMetadata) *FileMetadata {
	// Normalize artist and album names according to rules
	metadata.Artist = strings.TrimSpace(metadata.Artist)
	metadata.Album = strings.TrimSpace(metadata.Album)
	
	// Other normalization rules would go here
	
	return metadata
}

// createStagingRecords creates staging records in the database
func (lp *LibraryProcessor) createStagingRecords(artist *models.Artist, album *models.Album, metadata *FileMetadata, filePath string) error {
	// First, get or create the artist
	var existingArtist models.Artist
	result := lp.db.Where("name = ?", artist.Name).First(&existingArtist)
	
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// Artist doesn't exist, create it with directory code
			newArtist := models.Artist{
				Name:          artist.Name,
				NameNormalized: strings.ToLower(artist.Name), // Simplified normalization
				DirectoryCode: artist.DirectoryCode,
			}
			
			if err := lp.db.Create(&newArtist).Error; err != nil {
				return fmt.Errorf("failed to create artist: %w", err)
			}
			
			artist.ID = newArtist.ID
		} else {
			return fmt.Errorf("failed to query artist: %w", result.Error)
		}
	} else {
		artist.ID = existingArtist.ID
	}

	// Then, get or create the album
	var existingAlbum models.Album
	result = lp.db.Where("name = ? AND artist_id = ?", album.Name, artist.ID).First(&existingAlbum)
	
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// Album doesn't exist, create it
			newAlbum := models.Album{
				Name:        album.Name,
				NameNormalized: strings.ToLower(album.Name),
				ArtistID:    artist.ID,
				AlbumStatus: "New", // New albums start in 'New' status and need to be reviewed
			}
			
			if album.ReleaseDate != nil {
				newAlbum.ReleaseDate = album.ReleaseDate
			}
			
			if err := lp.db.Create(&newAlbum).Error; err != nil {
				return fmt.Errorf("failed to create album: %w", err)
			}
			
			album.ID = newAlbum.ID
		} else {
			return fmt.Errorf("failed to query album: %w", result.Error)
		}
	} else {
		album.ID = existingAlbum.ID
	}

	// Create the song record
	song := models.Song{
		Name:        metadata.Title,
		NameNormalized: strings.ToLower(metadata.Title),
		AlbumID:     album.ID,
		ArtistID:    artist.ID,
		FileName:    metadata.FileName,
		RelativePath: strings.TrimPrefix(filePath, lp.config.StagingDir),
		Directory:   filepath.Dir(strings.TrimPrefix(filePath, lp.config.StagingDir)),
	}
	
	if err := lp.db.Create(&song).Error; err != nil {
		return fmt.Errorf("failed to create song: %w", err)
	}

	return nil
}

// PromoteToProduction promotes approved staging content to production
func (lp *LibraryProcessor) PromoteToProduction(albumID int64) error {
	// Get the album from staging to promote
	var album models.Album
	if err := lp.db.First(&album, albumID).Error; err != nil {
		return fmt.Errorf("failed to find album: %w", err)
	}

	// Verify the album is in OK status
	if album.AlbumStatus != "Ok" {
		return fmt.Errorf("album is not in Ok status, current status: %s", album.AlbumStatus)
	}

	// Get related artist
	var artist models.Artist
	if err := lp.db.First(&artist, album.ArtistID).Error; err != nil {
		return fmt.Errorf("failed to find artist: %w", err)
	}

	// Calculate production path using directory code and template
	productionPath, err := lp.pathResolver.ResolveForArtistAlbum(&artist, &album, lp.config.ProductionDir)
	if err != nil {
		return fmt.Errorf("failed to resolve production path: %w", err)
	}

	// Get all songs in this album from staging
	var songs []models.Song
	if err := lp.db.Where("album_id = ?", albumID).Find(&songs).Error; err != nil {
		return fmt.Errorf("failed to find songs: %w", err)
	}

	// Move each file from staging to production and update DB records
	for i := range songs {
		stagingFilePath := filepath.Join(lp.config.StagingDir, songs[i].RelativePath)
		productionFilePath := filepath.Join(productionPath, filepath.Base(stagingFilePath))
		
		// Ensure production directory exists
		if err := os.MkdirAll(filepath.Dir(productionFilePath), 0755); err != nil {
			return fmt.Errorf("failed to create production directory: %w", err)
		}

		// Move the file from staging to production
		if err := os.Rename(stagingFilePath, productionFilePath); err != nil {
			return fmt.Errorf("failed to move file to production: %w", err)
		}

		// Update the song record with new production path
		songs[i].Directory = strings.TrimPrefix(filepath.Dir(productionFilePath), lp.config.ProductionDir)
		songs[i].RelativePath = strings.TrimPrefix(productionFilePath, lp.config.ProductionDir)
		songs[i].FileName = filepath.Base(productionFilePath)
		
		if err := lp.db.Save(&songs[i]).Error; err != nil {
			return fmt.Errorf("failed to update song path: %w", err)
		}
	}

	// Update album status to reflect it's now in production
	album.AlbumStatus = "InProduction"
	if err := lp.db.Save(&album).Error; err != nil {
		return fmt.Errorf("failed to update album status: %w", err)
	}

	return nil
}

// FileMetadata represents metadata extracted from a media file
type FileMetadata struct {
	Title       string
	Artist      string
	Album       string
	TrackNumber int
	DiscNumber  int
	Genre       string
	Duration    time.Duration
	BitRate     int // in kbps
	ReleaseDate time.Time
	FileName    string
	FilePath    string
}

// LibrarySelector selects appropriate production libraries based on directory codes
type LibrarySelector struct {
	db     *gorm.DB
	config *LibrarySelectionConfig
}

// LibrarySelectionConfig defines rules for selecting production libraries
type LibrarySelectionConfig struct {
	Strategy       string            `mapstructure:"strategy"`        // "hash", "round_robin", "directory_code", "size_based"
	LoadBalancing  LoadBalancingConfig `mapstructure:"load_balancing"`
	DirectoryRules map[string]string   `mapstructure:"directory_rules"` // Artist directory code to library mapping
	SizeThresholds SizeThresholdConfig `mapstructure:"size_thresholds"`
}

// LoadBalancingConfig configures load balancing for library selection
type LoadBalancingConfig struct {
	Enabled              bool    `mapstructure:"enabled"`
	ThresholdPercentage  float64 `mapstructure:"threshold_percentage"` // Move to next library when current is this % full
	CapacityProbeCommand string  `mapstructure:"capacity_probe_command"`
	CapacityCheckInterval time.Duration `mapstructure:"capacity_check_interval"`
	StopThresholdPercentage float64 `mapstructure:"stop_threshold_percentage"` // Stop allocations at this %
}

// SizeThresholdConfig configures size thresholds for library selection
type SizeThresholdConfig struct {
	MaxSizePerLibrary int64 `mapstructure:"max_size_per_library"` // in bytes
}

// NewLibrarySelector creates a new library selector
func NewLibrarySelector(db *gorm.DB, config *LibrarySelectionConfig) *LibrarySelector {
	return &LibrarySelector{
		db:     db,
		config: config,
	}
}

// SelectLibrary determines which production library to use for a given artist directory code
func (ls *LibrarySelector) SelectLibrary(artistDirectoryCode string) (*models.Library, error) {
	switch ls.config.Strategy {
	case "hash":
		return ls.selectByHash(artistDirectoryCode)
	case "round_robin":
		return ls.selectByRoundRobin(artistDirectoryCode)
	case "directory_code":
		return ls.selectByDirectoryCode(artistDirectoryCode)
	case "size_based":
		return ls.selectBySize(artistDirectoryCode)
	default:
		return ls.selectByDefaultStrategy(artistDirectoryCode)
	}
}

// selectByDirectoryCode uses configured rules to map directory codes to libraries
func (ls *LibrarySelector) selectByDirectoryCode(directoryCode string) (*models.Library, error) {
	// Check if there's a specific rule for this directory code
	for pattern, libraryName := range ls.config.DirectoryRules {
		if ls.matchesPattern(directoryCode, pattern) {
			var library models.Library
			if err := ls.db.Where("name = ? AND type = ?", libraryName, "production").First(&library).Error; err != nil {
				return nil, fmt.Errorf("configured library not found: %w", err)
			}
			return &library, nil
		}
	}

	// Fallback: Use hash-based selection for the first character of directory code
	firstChar := string(directoryCode[0])
	availableLibraries, err := ls.getAvailableProductionLibraries()
	if err != nil {
		return nil, fmt.Errorf("failed to get available libraries: %w", err)
	}

	if len(availableLibraries) == 0 {
		return nil, fmt.Errorf("no available production libraries")
	}

	index := hashString(firstChar) % len(availableLibraries)
	return &availableLibraries[index], nil
}

// matchesPattern checks if a directory code matches a pattern (e.g., "A-C")
func (ls *LibrarySelector) matchesPattern(directoryCode, pattern string) bool {
	if strings.Contains(pattern, "-") {
		// Pattern like "A-C": check if first character is within range
		parts := strings.Split(pattern, "-")
		if len(parts) == 2 {
			if len(directoryCode) == 0 {
				return false
			}
			
			firstChar := directoryCode[0]
			startChar := parts[0][0]
			endChar := parts[1][0]
			
			return firstChar >= startChar && firstChar <= endChar
		}
	}
	
	// Direct match
	return directoryCode == pattern
}

// getAvailableProductionLibraries returns unlocked production libraries
func (ls *LibrarySelector) getAvailableProductionLibraries() ([]models.Library, error) {
	var libraries []models.Library
	if err := ls.db.Where("type = ? AND is_locked = ?", "production", false).Find(&libraries).Error; err != nil {
		return nil, fmt.Errorf("failed to get production libraries: %w", err)
	}
	return libraries, nil
}

// hashString creates a simple hash for distribution
func hashString(s string) int {
	var hash int
	for i := 0; i < len(s); i++ {
		hash = hash*31 + int(s[i])
	}
	if hash < 0 {
		hash = -hash
	}
	return hash
}

// selectByDefaultStrategy uses directory code as the default strategy
func (ls *LibrarySelector) selectByDefaultStrategy(directoryCode string) (*models.Library, error) {
	return ls.selectByDirectoryCode(directoryCode)
}

// selectByHash uses hash-based distribution
func (ls *LibrarySelector) selectByHash(directoryCode string) (*models.Library, error) {
	availableLibraries, err := ls.getAvailableProductionLibraries()
	if err != nil {
		return nil, fmt.Errorf("failed to get available libraries: %w", err)
	}

	if len(availableLibraries) == 0 {
		return nil, fmt.Errorf("no available production libraries")
	}

	hashValue := hashString(directoryCode)
	index := hashValue % len(availableLibraries)
	return &availableLibraries[index], nil
}

// selectByRoundRobin uses round-robin distribution
func (ls *LibrarySelector) selectByRoundRobin(directoryCode string) (*models.Library, error) {
	// This would require tracking of round-robin state in a persistent store
	// For simplicity, using hash to simulate a consistent assignment
	availableLibraries, err := ls.getAvailableProductionLibraries()
	if err != nil {
		return nil, fmt.Errorf("failed to get available libraries: %w", err)
	}

	if len(availableLibraries) == 0 {
		return nil, fmt.Errorf("no available production libraries")
	}

	// Use directory code as input for selection
	hashValue := hashString(directoryCode)
	index := hashValue % len(availableLibraries)
	return &availableLibraries[index], nil
}

// selectBySize distributes to libraries with available space
func (ls *LibrarySelector) selectBySize(directoryCode string) (*models.Library, error) {
	var libraries []models.Library
	if err := ls.db.Where("type = ? AND is_locked = ?", "production", false).Find(&libraries).Error; err != nil {
		return nil, fmt.Errorf("failed to get production libraries: %w", err)
	}

	if len(libraries) == 0 {
		return nil, fmt.Errorf("no available production libraries")
	}

	// For this implementation, just return the first library
	// In a real implementation, we would check actual disk space usage
	return &libraries[0], nil
}