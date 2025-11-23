package media

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"melodee/internal/test"
)

func TestChecksumService_CalculateFileChecksum(t *testing.T) {
	// Create temporary directory and test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.mp3")
	testContent := "test content for checksum validation"
	
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	assert.NoError(t, err)

	// Setup test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Create checksum service with SHA256 algorithm
	config := &ChecksumConfig{
		Algorithm: "sha256",
	}
	service := NewChecksumService(db, config)

	// Test checksum calculation
	checksum, err := service.CalculateFileChecksum(testFile)
	assert.NoError(t, err)
	assert.NotEmpty(t, checksum)
	assert.Len(t, checksum, 64) // SHA256 produces 64-character hex string

	// Calculate expected checksum manually to verify
	expectedChecksum, err := calculateSHA256Manually(testContent)
	assert.NoError(t, err)
	assert.Equal(t, expectedChecksum, checksum)
}

func TestChecksumService_CalculateFileChecksumWithCRC32(t *testing.T) {
	// Create temporary directory and test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.mp3")
	testContent := "test content for CRC32 checksum validation"
	
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	assert.NoError(t, err)

	// Setup test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Create checksum service with CRC32 algorithm
	config := &ChecksumConfig{
		Algorithm: "crc32",
	}
	service := NewChecksumService(db, config)

	// Test checksum calculation
	checksum, err := service.CalculateFileChecksum(testFile)
	assert.NoError(t, err)
	assert.NotEmpty(t, checksum)
	assert.Len(t, checksum, 8) // CRC32 produces 8-character hex string
}

func TestChecksumService_Idempotency(t *testing.T) {
	// Create temporary directory and test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "idempotency_test.mp3")
	testContent := "idempotency test content"
	
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	assert.NoError(t, err)

	// Setup test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Create checksum service
	service := NewChecksumService(db, nil) // Use default config (SHA256)

	// Calculate checksum twice - should be the same
	checksum1, err := service.CalculateFileChecksum(testFile)
	assert.NoError(t, err)
	assert.NotEmpty(t, checksum1)

	checksum2, err := service.CalculateFileChecksum(testFile)
	assert.NoError(t, err)
	assert.NotEmpty(t, checksum2)

	// Checksums should be identical (idempotency)
	assert.Equal(t, checksum1, checksum2)
}

func TestChecksumService_IsFileProcessed(t *testing.T) {
	// Create temporary directory and test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "process_test.mp3")
	testContent := "process test content"
	
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	assert.NoError(t, err)

	// Setup test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Create checksum service
	service := NewChecksumService(db, nil)

	// Initially, file should not be processed (no record in DB)
	processed, checksum, err := service.IsFileProcessed(testFile)
	assert.NoError(t, err)
	assert.False(t, processed)
	assert.NotEmpty(t, checksum)

	// Test with a non-existent file
	processed, _, err = service.IsFileProcessed(filepath.Join(tempDir, "nonexistent.mp3"))
	assert.Error(t, err)
}

func TestChecksumService_BatchCalculateChecksums(t *testing.T) {
	// Create temporary directory and test files
	tempDir := t.TempDir()
	
	files := []struct{
		name string
		content string
	}{
		{"file1.mp3", "content 1"},
		{"file2.mp3", "content 2"},
		{"file3.mp3", "content 3"},
	}

	filePaths := make([]string, len(files))
	for i, f := range files {
		path := filepath.Join(tempDir, f.name)
		err := os.WriteFile(path, []byte(f.content), 0644)
		assert.NoError(t, err)
		filePaths[i] = path
	}

	// Setup test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Create checksum service
	service := NewChecksumService(db, nil)

	// Test batch checksum calculation
	results, err := service.BatchCalculateChecksums(filePaths)
	assert.NoError(t, err)
	assert.Len(t, results, len(filePaths))

	// Verify all files have checksums
	for _, path := range filePaths {
		_, exists := results[path]
		assert.True(t, exists)
		assert.NotEmpty(t, results[path])
	}
}

func TestChecksumService_DefaultConfig(t *testing.T) {
	// Test that the default config is properly set
	config := DefaultChecksumConfig()
	assert.Equal(t, "sha256", config.Algorithm)
	
	// Setup test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Create service with nil config (should use defaults)
	service := NewChecksumService(db, nil)
	
	// Verify the service uses SHA256 by default
	assert.Equal(t, "sha256", service.config.Algorithm)
}

func TestChecksumService_InvalidAlgorithm(t *testing.T) {
	// Create temporary directory and test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "invalid_algo_test.mp3")
	testContent := "invalid algorithm test content"
	
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	assert.NoError(t, err)

	// Setup test database
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Create checksum service with invalid algorithm (should default to SHA256)
	config := &ChecksumConfig{
		Algorithm: "invalid_algorithm",
	}
	service := NewChecksumService(db, config)

	// Test that it defaults to SHA256 (the default case)
	checksum, err := service.CalculateFileChecksum(testFile)
	assert.NoError(t, err)
	assert.NotEmpty(t, checksum)
	// Should still produce a valid SHA256 hash
	assert.Len(t, checksum, 64)
}

// Helper function to calculate SHA256 manually for testing
func calculateSHA256Manually(content string) (string, error) {
	h := sha256.New()
	h.Write([]byte(content))
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}