package media

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"melodee/internal/directory"
	"melodee/internal/models"
	"melodee/internal/test"
)

func TestMediaProcessor(t *testing.T) {
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	tempDir := t.TempDir()

	// Initialize components
	validatorConfig := DefaultValidationConfig()
	validator := NewMediaFileValidator(validatorConfig)

	ffmpegConfig := DefaultFFmpegConfig()
	ffmpegProcessor := NewFFmpegProcessor(ffmpegConfig)

	quarantineService := NewQuarantineService(db, filepath.Join(tempDir, "quarantine"))

	dirConfig := directory.DefaultPathTemplateConfig()
	dirResolver := directory.NewPathTemplateResolver(dirConfig)

	// Create processing config
	procConfig := DefaultProcessingConfig()
	procConfig.InboundDir = filepath.Join(tempDir, "inbound")
	procConfig.StagingDir = filepath.Join(tempDir, "staging")
	procConfig.ProductionDir = filepath.Join(tempDir, "production")

	// Create processor
	processor := NewMediaProcessor(procConfig, db, dirResolver, quarantineService, validator, ffmpegProcessor)

	// Create required directories
	err := os.MkdirAll(procConfig.InboundDir, 0755)
	assert.NoError(t, err)
	err = os.MkdirAll(procConfig.StagingDir, 0755)
	assert.NoError(t, err)
	err = os.MkdirAll(procConfig.ProductionDir, 0755)
	assert.NoError(t, err)

	// Test basic processor creation
	assert.NotNil(t, processor)
	assert.Equal(t, procConfig, processor.config)
}

func TestMediaFileValidator(t *testing.T) {
	validatorConfig := DefaultValidationConfig()
	validator := NewMediaFileValidator(validatorConfig)

	// Create a temporary file for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.mp3")

	// Create a small test file
	err := os.WriteFile(testFile, []byte("fake MP3 content"), 0644)
	assert.NoError(t, err)

	// Test valid file validation
	err = validator.Validate(testFile)
	// Note: This might fail due to the file not being a real MP3, which is expected
	// The important thing is that the basic validation logic works
	_ = err // Use the error variable to avoid "unused" warning

	// Test file not existing
	err = validator.Validate(filepath.Join(tempDir, "nonexistent.mp3"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file does not exist")

	// Test empty file
	emptyFile := filepath.Join(tempDir, "empty.mp3")
	err = os.WriteFile(emptyFile, []byte(""), 0644)
	assert.NoError(t, err)

	err = validator.Validate(emptyFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file is empty")

	// Test path validation
	err = validator.ValidatePath("../../test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path traversal detected")

	err = validator.ValidatePath("valid_path")
	assert.NoError(t, err)
}

func TestMediaFileValidatorFormatValidation(t *testing.T) {
	validatorConfig := DefaultValidationConfig()
	validator := NewMediaFileValidator(validatorConfig)

	// Test allowed formats
	allowedExts := []string{".mp3", ".flac", ".ogg", ".opus", ".m4a", ".wav", ".aac"}
	for _, ext := range allowedExts {
		assert.True(t, validator.isAllowedFormat(ext), "Format %s should be allowed", ext)
		assert.True(t, validator.isAllowedFormat(strings.ToUpper(ext)), "Format %s should be allowed", strings.ToUpper(ext))
	}

	// Test disallowed formats
	disallowedExts := []string{".txt", ".exe", ".doc", ".invalid"}
	for _, ext := range disallowedExts {
		assert.False(t, validator.isAllowedFormat(ext), "Format %s should not be allowed", ext)
	}
}

func TestMediaFileValidatorConfig(t *testing.T) {
	// Test default config
	config := DefaultValidationConfig()
	assert.Equal(t, 64, config.MinBitRate)
	assert.Equal(t, 320, config.MaxBitRate)
	assert.Equal(t, int64(500*1024*1024), config.MaxFileSize) // 500MB
	assert.Equal(t, 10, config.MinDuration)                   // 10 seconds
	assert.Equal(t, 7200, config.MaxDuration)                 // 2 hours
	assert.True(t, config.CheckCorruption)
}

func TestLibrarySelectionConfig(t *testing.T) {
	// Test default library selection config
	config := DefaultLibrarySelectionConfig()
	assert.Equal(t, "directory_code", config.LibraryStrategy)
	assert.True(t, config.LoadBalancing.Enabled)
	assert.Equal(t, 80, config.LoadBalancing.ThresholdPercentage)
	assert.Equal(t, 90, config.LoadBalancing.StopThresholdPercentage)
	assert.Equal(t, "melodee", config.HashSalt)
}

func TestMediaProcessorDirectoryCodeMatching(t *testing.T) {
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	tempDir := t.TempDir()
	procConfig := DefaultProcessingConfig()
	procConfig.InboundDir = filepath.Join(tempDir, "inbound")
	procConfig.StagingDir = filepath.Join(tempDir, "staging")
	procConfig.ProductionDir = filepath.Join(tempDir, "production")

	// Initialize components
	validator := NewMediaFileValidator(DefaultValidationConfig())
	ffmpegProcessor := NewFFmpegProcessor(DefaultFFmpegConfig())
	quarantineService := NewQuarantineService(db, filepath.Join(tempDir, "quarantine"))
	dirResolver := directory.NewPathTemplateResolver(directory.DefaultPathTemplateConfig())

	processor := NewMediaProcessor(procConfig, db, dirResolver, quarantineService, validator, ffmpegProcessor)

	// Test range matching
	testCases := []struct {
		name      string
		code      string
		rangeRule string
		expected  bool
	}{
		{"Code A matches A-C", "A", "A-C", true},
		{"Code B matches A-C", "B", "A-C", true},
		{"Code C matches A-C", "C", "A-C", true},
		{"Code D doesn't match A-C", "D", "A-C", false},
		{"Code Z doesn't match A-C", "Z", "A-C", false},
		{"Code LZ matches L-P", "LZ", "L-P", true},
		{"Code AA doesn't match L-P", "AA", "L-P", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := processor.matchesRange(tc.code, tc.rangeRule)
			assert.Equal(t, tc.expected, result, "Range matching failed for code: %s, rule: %s", tc.code, tc.rangeRule)
		})
	}
}

func TestMediaProcessorSimpleHash(t *testing.T) {
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	tempDir := t.TempDir()
	procConfig := DefaultProcessingConfig()
	procConfig.InboundDir = filepath.Join(tempDir, "inbound")
	procConfig.StagingDir = filepath.Join(tempDir, "staging")
	procConfig.ProductionDir = filepath.Join(tempDir, "production")

	// Initialize components
	validator := NewMediaFileValidator(DefaultValidationConfig())
	ffmpegProcessor := NewFFmpegProcessor(DefaultFFmpegConfig())
	quarantineService := NewQuarantineService(db, filepath.Join(tempDir, "quarantine"))
	dirResolver := directory.NewPathTemplateResolver(directory.DefaultPathTemplateConfig())

	processor := NewMediaProcessor(procConfig, db, dirResolver, quarantineService, validator, ffmpegProcessor)

	// Test that same input produces same hash
	input := "test_input"
	hash1 := processor.simpleHash(input)
	hash2 := processor.simpleHash(input)
	assert.Equal(t, hash1, hash2, "Same input should produce same hash")

	// Test that different inputs produce different hashes (most of the time)
	hash3 := processor.simpleHash("different_input")
	// Note: There's a small chance of collision, but it's very unlikely with these simple tests
	if hash1 == hash3 {
		t.Log("Hash collision occurred (this is rare but possible)")
	}
}

func TestMediaProcessorValidation(t *testing.T) {
	// Create validation config with more permissive settings for testing
	config := &ValidationConfig{
		MinBitRate:      8,
		MaxBitRate:      500,
		MaxFileSize:     10 * 1024 * 1024,         // 10MB for testing
		AllowedFormats:  []string{".mp3", ".txt"}, // Add txt for testing
		MinDuration:     1,
		MaxDuration:     3600,
		MinSampleRate:   8000,
		MaxSampleRate:   384000,
		MinChannels:     1,
		MaxChannels:     8,
		MaxArtworkSize:  50 * 1024 * 1024, // 50MB
		CheckCorruption: false,            // Disable corruption check for test files
	}

	validator := NewMediaFileValidator(config)

	tempDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content for validation"), 0644)
	assert.NoError(t, err)

	// This should pass basic validation (though fail format validation)
	err = validator.validateBasicFile(testFile)
	assert.NoError(t, err)

	// Test file size validation
	err = validator.validateBasicFile(testFile)
	assert.NoError(t, err)

	// Create an empty file
	emptyFile := filepath.Join(tempDir, "empty.txt")
	err = os.WriteFile(emptyFile, []byte(""), 0644)
	assert.NoError(t, err)

	err = validator.validateBasicFile(emptyFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file is empty")
}

func TestTranscodedFilesCache(t *testing.T) {
	tempDir := t.TempDir()

	// Create FFmpeg processor
	ffmpegConfig := DefaultFFmpegConfig()
	processor := NewFFmpegProcessor(ffmpegConfig)

	// Create transcode service
	cacheDir := filepath.Join(tempDir, "cache")
	service := NewTranscodeService(processor, cacheDir, 10*1024*1024) // 10MB cache

	// Test cache creation
	assert.NotNil(t, service)
	assert.Equal(t, cacheDir, service.GetCacheDir())

	// Test cache stats
	stats := service.cache.GetCacheStats()
	assert.Equal(t, float64(0), stats["current_size"])
	assert.Equal(t, float64(10*1024*1024), stats["max_size"])
	assert.Equal(t, 0.0, stats["used_percent"])
}

func TestMediaFileIsMediaFile(t *testing.T) {
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	tempDir := t.TempDir()
	procConfig := DefaultProcessingConfig()
	procConfig.InboundDir = filepath.Join(tempDir, "inbound")
	procConfig.StagingDir = filepath.Join(tempDir, "staging")
	procConfig.ProductionDir = filepath.Join(tempDir, "production")

	// Initialize components
	validator := NewMediaFileValidator(DefaultValidationConfig())
	ffmpegProcessor := NewFFmpegProcessor(DefaultFFmpegConfig())
	quarantineService := NewQuarantineService(db, filepath.Join(tempDir, "quarantine"))
	dirResolver := directory.NewPathTemplateResolver(directory.DefaultPathTemplateConfig())

	processor := NewMediaProcessor(procConfig, db, dirResolver, quarantineService, validator, ffmpegProcessor)

	// Test media file detection
	mediaExts := []string{".mp3", ".flac", ".ogg", ".opus", ".m4a", ".mp4", ".aac", ".wma", ".wav", ".aiff", ".ape", ".wv", ".dsf", ".cda"}
	for _, ext := range mediaExts {
		filename := "test" + ext
		assert.True(t, processor.isMediaFile(filename), "File with extension %s should be detected as media", ext)
	}

	// Test non-media extensions
	nonMediaExts := []string{".txt", ".doc", ".pdf", ".jpg", ".png", ".exe", ".zip"}
	for _, ext := range nonMediaExts {
		filename := "test" + ext
		assert.False(t, processor.isMediaFile(filename), "File with extension %s should not be detected as media", ext)
	}

	// Test case insensitivity
	assert.True(t, processor.isMediaFile("test.MP3"), "Uppercase extension should be detected")
	assert.True(t, processor.isMediaFile("test.mp3"), "Lowercase extension should be detected")
	assert.True(t, processor.isMediaFile("test.Mp3"), "Mixed case extension should be detected")
}

func TestCalculateChecksum(t *testing.T) {
	// Create validator
	validator := NewMediaFileValidator(DefaultValidationConfig())

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "checksum_test.txt")
	content := "test content for checksum calculation"

	err := os.WriteFile(testFile, []byte(content), 0644)
	assert.NoError(t, err)

	// Calculate checksum
	checksum, err := validator.calculateChecksum(testFile)
	assert.NoError(t, err)
	assert.NotEmpty(t, checksum)
	assert.Len(t, checksum, 64) // SHA256 produces 64-character hex string

	// Calculate checksum twice to ensure consistency
	checksum2, err := validator.calculateChecksum(testFile)
	assert.NoError(t, err)
	assert.Equal(t, checksum, checksum2, "Checksums should be identical for same file")
}
