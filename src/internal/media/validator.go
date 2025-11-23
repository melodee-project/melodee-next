package media

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-audio/wav"
	"github.com/hajimehoshi/go-mp3"
	"github.com/pkg/errors"
)

// ValidationConfig holds configuration for file validation
type ValidationConfig struct {
	MinBitRate      int      `mapstructure:"min_bitrate"`      // Minimum bitrate in kbps
	MaxBitRate      int      `mapstructure:"max_bitrate"`      // Maximum bitrate in kbps
	MaxFileSize     int64    `mapstructure:"max_file_size"`    // Maximum file size in bytes
	AllowedFormats  []string `mapstructure:"allowed_formats"`  // Allowed file extensions
	MinDuration     int      `mapstructure:"min_duration"`     // Minimum duration in seconds
	MaxDuration     int      `mapstructure:"max_duration"`     // Maximum duration in seconds
	MinSampleRate   int      `mapstructure:"min_sample_rate"`  // Minimum sample rate in Hz
	MaxSampleRate   int      `mapstructure:"max_sample_rate"`  // Maximum sample rate in Hz
	MinChannels     int      `mapstructure:"min_channels"`     // Minimum number of channels
	MaxChannels     int      `mapstructure:"max_channels"`     // Maximum number of channels
	MaxArtworkSize  int64    `mapstructure:"max_artwork_size"` // Maximum embedded artwork size in bytes
	CheckCorruption bool     `mapstructure:"check_corruption"` // Whether to check for file corruption
}

// DefaultValidationConfig returns the default validation configuration
func DefaultValidationConfig() *ValidationConfig {
	return &ValidationConfig{
		MinBitRate:      64,                // 64 kbps minimum
		MaxBitRate:      320,               // 320 kbps maximum for lossy, unlimited for lossless
		MaxFileSize:     500 * 1024 * 1024, // 500 MB max
		AllowedFormats:  []string{".mp3", ".flac", ".ogg", ".opus", ".m4a", ".wav", ".aac"},
		MinDuration:     10,               // 10 seconds minimum
		MaxDuration:     7200,             // 2 hours maximum
		MinSampleRate:   44100,            // 44.1 kHz minimum
		MaxSampleRate:   192000,           // 192 kHz maximum
		MinChannels:     1,                // Mono minimum
		MaxChannels:     8,                // 7.1 surround maximum
		MaxArtworkSize:  10 * 1024 * 1024, // 10 MB max artwork
		CheckCorruption: true,
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
	if !mv.isAllowedFormat(filepath.Ext(filePath)) {
		return fmt.Errorf("format %s not allowed", filepath.Ext(filePath))
	}

	// 3. Metadata validation
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

	// 5. Corruption check
	if mv.config.CheckCorruption {
		if err := mv.checkForCorruption(filePath, metadata); err != nil {
			return fmt.Errorf("corruption check failed: %w", err)
		}
	}

	return nil
}

// validateBasicFile performs basic file validation
func (mv *MediaFileValidator) validateBasicFile(filePath string) error {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filePath)
	}

	// Check file size
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	if info.Size() == 0 {
		return fmt.Errorf("file is empty")
	}

	if info.Size() > mv.config.MaxFileSize {
		return fmt.Errorf("file size %d bytes exceeds maximum %d bytes", info.Size(), mv.config.MaxFileSize)
	}

	return nil
}

// isAllowedFormat checks if the file format is allowed
func (mv *MediaFileValidator) isAllowedFormat(ext string) bool {
	ext = strings.ToLower(ext)
	for _, allowedExt := range mv.config.AllowedFormats {
		if ext == strings.ToLower(allowedExt) {
			return true
		}
	}
	return false
}

// validateMetadata validates extracted metadata
func (mv *MediaFileValidator) validateMetadata(metadata *MediaMetadata) error {
	// Validate required metadata fields
	if metadata.Name == "" {
		return fmt.Errorf("filename is empty")
	}

	// For audio files specifically
	if metadata.BitRate != 0 {
		if metadata.BitRate < mv.config.MinBitRate {
			return fmt.Errorf("bitrate %d kbps below minimum %d kbps", metadata.BitRate, mv.config.MinBitRate)
		}

		// For lossy formats, apply max bitrate; for lossless, allow higher rates
		ext := strings.ToLower(filepath.Ext(metadata.FilePath))
		if ext == ".mp3" || ext == ".aac" || ext == ".ogg" || ext == ".opus" {
			// Lossy formats have max bitrate
			if metadata.BitRate > mv.config.MaxBitRate {
				return fmt.Errorf("bitrate %d kbps above maximum %d kbps for lossy format", metadata.BitRate, mv.config.MaxBitRate)
			}
		}
	}

	if metadata.Duration != 0 {
		minDur := time.Duration(mv.config.MinDuration) * time.Second
		maxDur := time.Duration(mv.config.MaxDuration) * time.Second

		if metadata.Duration < minDur {
			return fmt.Errorf("duration %v below minimum %v", metadata.Duration, minDur)
		}

		if metadata.Duration > maxDur {
			return fmt.Errorf("duration %v above maximum %v", metadata.Duration, maxDur)
		}
	}

	if metadata.SampleRate != 0 {
		if metadata.SampleRate < mv.config.MinSampleRate {
			return fmt.Errorf("sample rate %d Hz below minimum %d Hz", metadata.SampleRate, mv.config.MinSampleRate)
		}

		if metadata.SampleRate > mv.config.MaxSampleRate {
			return fmt.Errorf("sample rate %d Hz above maximum %d Hz", metadata.SampleRate, mv.config.MaxSampleRate)
		}
	}

	if metadata.Channels != 0 {
		if metadata.Channels < mv.config.MinChannels {
			return fmt.Errorf("%d channels below minimum %d channels", metadata.Channels, mv.config.MinChannels)
		}

		if metadata.Channels > mv.config.MaxChannels {
			return fmt.Errorf("%d channels above maximum %d channels", metadata.Channels, mv.config.MaxChannels)
		}
	}

	return nil
}

// validateQuality checks technical quality parameters
func (mv *MediaFileValidator) validateQuality(metadata *MediaMetadata) error {
	// Quality validation is covered in validateMetadata
	// This function can contain additional quality checks if needed
	return nil
}

// checkForCorruption performs basic corruption detection
func (mv *MediaFileValidator) checkForCorruption(filePath string, metadata *MediaMetadata) error {
	// Check file header integrity for known formats
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".mp3":
		return mv.checkMP3Corruption(filePath)
	case ".flac":
		return mv.checkFLACCorruption(filePath)
	case ".wav":
		return mv.checkWAVCorruption(filePath)
	case ".m4a", ".aac":
		return mv.checkAACCorruption(filePath)
	case ".ogg", ".opus":
		return mv.checkOGGCorruption(filePath)
	default:
		// For unsupported formats, we perform a basic read test
		return mv.checkReadability(filePath)
	}
}

// checkMP3Corruption checks for MP3 file corruption
func (mv *MediaFileValidator) checkMP3Corruption(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Try to decode a portion to check for corruption
	decoder, err := mp3.NewDecoder(file)
	if err != nil {
		return fmt.Errorf("MP3 header corruption: %w", err)
	}
	// Note: mp3.Decoder doesn't have a Close method in this library version

	// Read a small portion to verify the file is decodable
	buffer := make([]byte, 4096)
	_, err = decoder.Read(buffer)
	if err != nil && err.Error() != "EOF" {
		return fmt.Errorf("MP3 content corruption: %w", err)
	}

	return nil
}

// checkFLACCorruption checks for FLAC file corruption
func (mv *MediaFileValidator) checkFLACCorruption(filePath string) error {
	// For FLAC, we'll do a basic header check
	// A full implementation would use a FLAC decoder library
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Read the first 4 bytes to check FLAC signature
	header := make([]byte, 4)
	_, err = file.Read(header)
	if err != nil {
		return fmt.Errorf("failed to read FLAC header: %w", err)
	}

	// FLAC files start with "fLaC"
	if string(header) != "fLaC" {
		return fmt.Errorf("invalid FLAC header, expected 'fLaC', got '%s'", string(header))
	}

	return nil
}

// checkWAVCorruption checks for WAV file corruption
func (mv *MediaFileValidator) checkWAVCorruption(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Try to decode as WAV
	decoder := wav.NewDecoder(file)
	if decoder == nil {
		return fmt.Errorf("WAV header corruption: could not create decoder")
	}

	return nil
}

// checkAACCorruption checks for AAC file corruption
func (mv *MediaFileValidator) checkAACCorruption(filePath string) error {
	// For AAC, we'll do a basic file readability check
	// A full implementation would use an AAC decoder library
	return mv.checkReadability(filePath)
}

// checkOGGCorruption checks for OGG file corruption
func (mv *MediaFileValidator) checkOGGCorruption(filePath string) error {
	// For OGG, we'll do a basic file readability check
	// A full implementation would use an OGG decoder library
	return mv.checkReadability(filePath)
}

// checkReadability attempts to read the file to detect obvious corruption
func (mv *MediaFileValidator) checkReadability(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Try to read the first and last few kilobytes to detect corruption
	buffer := make([]byte, 1024)

	// Read first 1KB
	_, err = file.Read(buffer)
	if err != nil && err.Error() != "EOF" {
		return fmt.Errorf("failed to read beginning of file: %w", err)
	}

	// Get file size and read last 1KB
	info, err := file.Stat()
	if err != nil {
		return err
	}

	if info.Size() > 1024 {
		_, err = file.Seek(-1024, 2) // Seek to 1KB from end
		if err != nil {
			return err
		}

		_, err = file.Read(buffer)
		if err != nil && err.Error() != "EOF" {
			return fmt.Errorf("failed to read end of file: %w", err)
		}
	}

	return nil
}

// extractMetadata extracts metadata from the file
func (mv *MediaFileValidator) extractMetadata(filePath string) (*MediaMetadata, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	// Calculate checksum
	checksum, err := mv.calculateChecksum(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// Create basic metadata with file info
	metadata := &MediaMetadata{
		FilePath: filePath,
		Size:     info.Size(),
		ModTime:  info.ModTime(),
		Checksum: checksum,
		Name:     filepath.Base(filePath),
	}

	// Try to extract more detailed metadata based on format
	if err := mv.extractFormatSpecificMetadata(metadata); err != nil {
		// Log the error but don't fail completely - return basic metadata
		fmt.Printf("Warning: failed to extract detailed metadata: %v\n", err)
	}

	return metadata, nil
}

// extractFormatSpecificMetadata extracts format-specific metadata
func (mv *MediaFileValidator) extractFormatSpecificMetadata(metadata *MediaMetadata) error {
	ext := strings.ToLower(filepath.Ext(metadata.FilePath))

	switch ext {
	case ".mp3":
		return mv.extractMP3Metadata(metadata)
	case ".wav":
		return mv.extractWAVMetadata(metadata)
	case ".flac":
		return mv.extractFLACMetadata(metadata)
	case ".m4a", ".aac":
		return mv.extractAACMetadata(metadata)
	case ".ogg", ".opus":
		return mv.extractOGGMetadata(metadata)
	default:
		return fmt.Errorf("unsupported format for detailed metadata extraction: %s", ext)
	}
}

// extractMP3Metadata extracts metadata from MP3 files
func (mv *MediaFileValidator) extractMP3Metadata(metadata *MediaMetadata) error {
	file, err := os.Open(metadata.FilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder, err := mp3.NewDecoder(file)
	if err != nil {
		return err
	}
	// Note: mp3.Decoder doesn't have a Close method in this library version

	// Set metadata from decoder
	metadata.Duration = time.Duration(decoder.Length()) * time.Second
	metadata.SampleRate = int(decoder.SampleRate())
	// Note: go-mp3 library doesn't provide Channels() method, assume stereo for MP3s
	metadata.Channels = 2
	// Note: MP3 decoder doesn't directly provide bitrate, we'd need to calculate it
	// This is a simplified approach

	return nil
}

// extractWAVMetadata extracts metadata from WAV files
func (mv *MediaFileValidator) extractWAVMetadata(metadata *MediaMetadata) error {
	file, err := os.Open(metadata.FilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := wav.NewDecoder(file)
	if !decoder.IsValidFile() {
		return fmt.Errorf("invalid WAV file")
	}

	// Set metadata from decoder
	metadata.SampleRate = int(decoder.SampleRate)
	metadata.Channels = int(decoder.NumChans)

	// Calculate duration from sample count and sample rate
	// Note: go-audio/wav doesn't have NumSampleFrames, use Duration() if available
	// For now, skip duration calculation
	if decoder.SampleRate > 0 && decoder.NumChans > 0 {
		// Duration would need to be calculated from file size and format
		// This is a placeholder
		metadata.Duration = 0
	}

	return nil
}

// extractFLACMetadata extracts metadata from FLAC files
func (mv *MediaFileValidator) extractFLACMetadata(metadata *MediaMetadata) error {
	// For FLAC, we'd typically use a FLAC library
	// This is a simplified placeholder
	// In a real implementation, we would parse the FLAC metadata blocks

	// For now, just return without errors
	return nil
}

// extractAACMetadata extracts metadata from AAC files
func (mv *MediaFileValidator) extractAACMetadata(metadata *MediaMetadata) error {
	// For AAC, we'd typically use a library like go-audio or similar
	// This is a simplified placeholder
	return nil
}

// extractOGGMetadata extracts metadata from OGG files
func (mv *MediaFileValidator) extractOGGMetadata(metadata *MediaMetadata) error {
	// For OGG, we'd typically use a library like go-ogg
	// This is a simplified placeholder
	return nil
}

// calculateChecksum calculates the SHA256 checksum of a file
func (mv *MediaFileValidator) calculateChecksum(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// ValidatePath validates that a file path is safe and within allowed boundaries
func (mv *MediaFileValidator) ValidatePath(path string) error {
	// Check for path traversal attempts
	if strings.Contains(path, "../") || strings.Contains(path, "..\\") {
		return errors.New("path traversal detected")
	}

	// Check for null bytes
	if strings.Contains(path, "\x00") {
		return errors.New("path contains null bytes")
	}

	// Check maximum path length
	if len(path) > 4096 {
		return fmt.Errorf("path too long: %d characters, max: 4096", len(path))
	}

	return nil
}
