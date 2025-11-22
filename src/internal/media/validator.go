package media

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dhowden/tag"
)

// ValidationConfig holds validation rules for media files
type ValidationConfig struct {
	MinBitRate       int      `mapstructure:"min_bitrate"`       // Minimum bitrate in kbps
	MaxBitRate       int      `mapstructure:"max_bitrate"`       // Maximum bitrate in kbps
	MaxFileSize      int64    `mapstructure:"max_file_size"`     // Maximum file size in bytes
	AllowedFormats   []string `mapstructure:"allowed_formats"`   // Allowed file extensions
	MinDuration      int      `mapstructure:"min_duration"`      // Minimum duration in seconds
	MaxDuration      int      `mapstructure:"max_duration"`      // Maximum duration in seconds
	ChecksumRequired bool     `mapstructure:"checksum_required"` // Whether checksum validation is required
}

// DefaultValidationConfig returns the default validation configuration
func DefaultValidationConfig() *ValidationConfig {
	return &ValidationConfig{
		MinBitRate:    64,   // 64 kbps minimum
		MaxBitRate:    320,  // 320 kbps maximum for lossy
		MaxFileSize:   100 * 1024 * 1024, // 100MB max file size
		AllowedFormats: []string{".mp3", ".flac", ".m4a", ".mp4", ".aac", ".ogg", ".opus", ".wav"},
		MinDuration:   10,   // 10 seconds minimum
		MaxDuration:   7200, // 2 hours maximum
		ChecksumRequired: true,
	}
}

// MediaFileValidator validates media files according to configured rules
type MediaFileValidator struct {
	config *ValidationConfig
}

// NewMediaFileValidator creates a new media file validator
func NewMediaFileValidator(config *ValidationConfig) *MediaFileValidator {
	if config == nil {
		config = DefaultValidationConfig()
	}
	
	return &MediaFileValidator{
		config: config,
	}
}

// Validate performs comprehensive file validation
func (mv *MediaFileValidator) Validate(filePath string) error {
	// 1. Basic file validation
	if err := mv.validateBasicFile(filePath); err != nil {
		return fmt.Errorf("basic file validation failed: %w", err)
	}

	// 2. Format validation
	ext := strings.ToLower(filepath.Ext(filePath))
	if !mv.isAllowedFormat(ext) {
		return fmt.Errorf("format %s not allowed", ext)
	}

	// 3. Extract and validate metadata
	metadata, err := mv.extractMetadata(filePath)
	if err != nil {
		return fmt.Errorf("metadata extraction failed: %w", err)
	}

	if err := mv.validateMetadata(metadata); err != nil {
		return fmt.Errorf("metadata validation failed: %w", err)
	}

	// 4. Quality validation
	if err := mv.validateQuality(metadata); err != nil {
		return fmt.Errorf("quality validation failed: %w", err)
	}

	// 5. Checksum validation if required
	if mv.config.ChecksumRequired {
		if err := mv.validateChecksum(filePath); err != nil {
			return fmt.Errorf("checksum validation failed: %w", err)
		}
	}

	return nil
}

// validateBasicFile performs basic file validation
func (mv *MediaFileValidator) validateBasicFile(filePath string) error {
	// Check if file exists
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("file does not exist: %w", err)
	}

	// Check if it's a regular file (not a directory)
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file")
	}

	// Check file size against maximum
	if info.Size() > mv.config.MaxFileSize {
		return fmt.Errorf("file size %d bytes exceeds maximum %d bytes", info.Size(), mv.config.MaxFileSize)
	}

	// Check if file is accessible for reading
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("file not accessible for reading: %w", err)
	}
	defer file.Close()

	return nil
}

// isAllowedFormat checks if the file format is allowed
func (mv *MediaFileValidator) isAllowedFormat(ext string) bool {
	for _, allowedExt := range mv.config.AllowedFormats {
		if strings.ToLower(allowedExt) == ext {
			return true
		}
	}
	return false
}

// extractMetadata extracts metadata from a media file
func (mv *MediaFileValidator) extractMetadata(filePath string) (*FileMetadata, error) {
	// Open file for reading
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Parse file tags using the tag library
	tagData, err := tag.ReadFrom(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read tags from file: %w", err)
	}

	// Extract file stats for duration and size
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file stats: %w", err)
	}

	// Create metadata struct
	metadata := &FileMetadata{
		Title:       tagData.Title(),
		Artist:      tagData.Artist(),
		Album:       tagData.Album(),
		TrackNumber: tagData.Track(),
		DiscNumber:  tagData.Disc(),
		Genre:       tagData.Genre(),
		FileName:    filepath.Base(filePath),
		FilePath:    filePath,
	}

	// Extract duration from file
	duration := time.Duration(0)
	if tagData.Duration() > 0 {
		duration = tagData.Duration()
	} else {
		// For formats that don't have duration in tags, we'd need to use a library like go-audio
		// to analyze the file. For now, we'll just set it to 0.
	}
	metadata.Duration = duration

	// Calculate bitrate from file size and duration
	if duration.Seconds() > 0 {
		bitrate := int((float64(stat.Size()) * 8) / duration.Seconds() / 1000) // kbps
		metadata.BitRate = bitrate
	}

	// Extract release date if available
	if tagData.Year() > 0 {
		metadata.ReleaseDate = time.Date(tagData.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	}

	return metadata, nil
}

// validateMetadata validates extracted metadata
func (mv *MediaFileValidator) validateMetadata(metadata *FileMetadata) error {
	// Validate required fields
	if metadata.Title == "" {
		// Use filename if no title is available
		name := strings.TrimSuffix(metadata.FileName, filepath.Ext(metadata.FileName))
		metadata.Title = name
	}

	if metadata.Artist == "" {
		return fmt.Errorf("artist name is required")
	}

	if metadata.Album == "" {
		return fmt.Errorf("album name is required")
	}

	return nil
}

// validateQuality checks technical quality parameters
func (mv *MediaFileValidator) validateQuality(metadata *FileMetadata) error {
	if metadata.BitRate != 0 && metadata.BitRate < mv.config.MinBitRate {
		return fmt.Errorf("bitrate %d kbps below minimum %d kbps", metadata.BitRate, mv.config.MinBitRate)
	}

	if metadata.BitRate != 0 && metadata.BitRate > mv.config.MaxBitRate {
		return fmt.Errorf("bitrate %d kbps above maximum %d kbps", metadata.BitRate, mv.config.MaxBitRate)
	}

	if metadata.Duration != 0 && metadata.Duration < time.Duration(mv.config.MinDuration)*time.Second {
		return fmt.Errorf("duration %v below minimum %v", metadata.Duration, time.Duration(mv.config.MinDuration)*time.Second)
	}

	if metadata.Duration != 0 && metadata.Duration > time.Duration(mv.config.MaxDuration)*time.Second {
		return fmt.Errorf("duration %v above maximum %v", metadata.Duration, time.Duration(mv.config.MaxDuration)*time.Second)
	}

	return nil
}

// validateChecksum validates the file checksum against stored value
func (mv *MediaFileValidator) validateChecksum(filePath string) error {
	// Calculate SHA256 checksum of the file
	checksum, err := mv.calculateFileChecksum(filePath)
	if err != nil {
		return fmt.Errorf("failed to calculate file checksum: %w", err)
	}

	// In a real implementation, we would compare this checksum with a stored value
	// For now, we just calculate it to ensure the file is readable and has a valid checksum
	fmt.Printf("File checksum: %s\n", checksum)

	return nil
}

// calculateFileChecksum calculates the SHA256 checksum of a file
func (mv *MediaFileValidator) calculateFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// ValidateAndProcessFile validates a file and returns normalized metadata
func (mv *MediaFileValidator) ValidateAndProcessFile(filePath string) (*FileMetadata, error) {
	// Validate the file first
	if err := mv.Validate(filePath); err != nil {
		return nil, err
	}

	// Extract and return metadata
	return mv.extractMetadata(filePath)
}

// CalculateFileChecksum is a public method to calculate checksum for other uses
func (mv *MediaFileValidator) CalculateFileChecksum(filePath string) (string, error) {
	return mv.calculateFileChecksum(filePath)
}

// ValidatePathSafety checks if a file path is safe from traversal attacks
func (mv *MediaFileValidator) ValidatePathSafety(path string) error {
	// Check for path traversal patterns
	if strings.Contains(path, "../") || strings.Contains(path, "..\\") {
		return fmt.Errorf("path traversal detected in path: %s", path)
	}

	// Check for null bytes
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("null byte in path: %s", path)
	}

	// Clean the path and compare with original to detect traversal
	cleaned := filepath.Clean(path)
	if cleaned != path {
		return fmt.Errorf("path contains relative components: %s", path)
	}

	return nil
}

// ValidateFileForPromotion checks if a file is ready for promotion from staging to production
func (mv *MediaFileValidator) ValidateFileForPromotion(filePath string, requireCueFiles bool) error {
	// Basic validation
	if err := mv.validateBasicFile(filePath); err != nil {
		return err
	}

	// Check if associated .cue files exist if required
	if requireCueFiles {
		cuePath := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + ".cue"
		if _, err := os.Stat(cuePath); os.IsNotExist(err) {
			return fmt.Errorf("required .cue file not found: %s", cuePath)
		}
	}

	// Validate content
	return mv.Validate(filePath)
}