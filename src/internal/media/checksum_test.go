package media

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"melodee/internal/test"
)

func TestChecksumService_CalculateAndValidate(t *testing.T) {
	// Create a test database instance
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "checksum_test_*.txt")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	// Write test content to the file
	testContent := "This is test content for checksum validation"
	_, err = tempFile.WriteString(testContent)
	assert.NoError(t, err)
	tempFile.Close()

	// Create checksum service
	service := NewChecksumService(db, nil)

	// Test checksum calculation
	checksum1, err := service.CalculateChecksum(tempFile.Name())
	assert.NoError(t, err)
	assert.NotEmpty(t, checksum1)

	// Calculate again and verify it's the same (idempotency)
	checksum2, err := service.CalculateChecksum(tempFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, checksum1, checksum2)

	// Test validation with correct checksum
	isValid, err := service.ValidateChecksum(tempFile.Name(), checksum1)
	assert.NoError(t, err)
	assert.True(t, isValid)

	// Test validation with wrong checksum
	wrongChecksum := "wrong_checksum_value_that_does_not_match"
	isValid, err = service.ValidateChecksum(tempFile.Name(), wrongChecksum)
	assert.NoError(t, err)
	assert.False(t, isValid)
}

func TestChecksumService_Idempotency(t *testing.T) {
	// Create a test database instance
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "idempotency_test_*.txt")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	// Write test content to the file
	testContent := "Test content for idempotency check"
	_, err = tempFile.WriteString(testContent)
	assert.NoError(t, err)
	tempFile.Close()

	// Create checksum service
	service := NewChecksumService(db, &ChecksumConfig{
		Algorithm:     "SHA256",
		EnableCaching: true,
		CacheTTL:      5 * time.Second,
		StoreLocation: "DB",
	})

	// Calculate checksum multiple times
	checksum1, err := service.CalculateChecksum(tempFile.Name())
	assert.NoError(t, err)
	assert.NotEmpty(t, checksum1)

	// Modify the file cache entry (simulate another process calculating at the same time)
	checksum2, err := service.CalculateChecksum(tempFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, checksum1, checksum2)

	// Verify the file integrity
	isValid, err := service.ValidateChecksum(tempFile.Name(), checksum1)
	assert.NoError(t, err)
	assert.True(t, isValid)
}

func TestChecksumService_Caching(t *testing.T) {
	// Create a test database instance
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "caching_test_*.txt")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	// Write test content to the file
	testContent := "Test content for caching"
	_, err = tempFile.WriteString(testContent)
	assert.NoError(t, err)
	tempFile.Close()

	// Create checksum service with caching enabled
	service := NewChecksumService(db, &ChecksumConfig{
		Algorithm:     "SHA256",
		EnableCaching: true,
		CacheTTL:      1 * time.Hour, // Long TTL for testing
		StoreLocation: "memory",
	})

	// Calculate checksum the first time
	checksum1, err := service.CalculateChecksum(tempFile.Name())
	assert.NoError(t, err)
	assert.NotEmpty(t, checksum1)

	// Check that it's cached
	cachedEntry, exists := service.getCachedChecksum(tempFile.Name())
	assert.True(t, exists)
	assert.Equal(t, checksum1, cachedEntry.Checksum)

	// Calculate again - should return cached value
	checksum2, err := service.CalculateChecksum(tempFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, checksum1, checksum2)

	// Change the file content to make cache invalid
	err = os.WriteFile(tempFile.Name(), []byte("modified content"), 0644)
	assert.NoError(t, err)

	// Calculate again - should detect file change and return new checksum
	checksum3, err := service.CalculateChecksum(tempFile.Name())
	assert.NoError(t, err)
	assert.NotEqual(t, checksum1, checksum3)
}

func TestChecksumService_BatchValidate(t *testing.T) {
	// Create a test database instance
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Create temporary files for testing
	tempFile1, err := os.CreateTemp("", "batch_test_1_*.txt")
	assert.NoError(t, err)
	defer os.Remove(tempFile1.Name())

	tempFile2, err := os.CreateTemp("", "batch_test_2_*.txt")
	assert.NoError(t, err)
	defer os.Remove(tempFile2.Name())

	// Write different content to files
	_, err = tempFile1.WriteString("Content for file 1")
	assert.NoError(t, err)
	tempFile1.Close()

	_, err = tempFile2.WriteString("Content for file 2")
	assert.NoError(t, err)
	tempFile2.Close()

	// Create checksum service
	service := NewChecksumService(db, nil)

	// Calculate checksums for both files
	checksum1, err := service.CalculateChecksum(tempFile1.Name())
	assert.NoError(t, err)

	checksum2, err := service.CalculateChecksum(tempFile2.Name())
	assert.NoError(t, err)

	// Create batch validation map
	files := map[string]string{
		tempFile1.Name(): checksum1,
		tempFile2.Name(): checksum2,
	}

	// Validate all files in batch
	results, errors, err := service.BatchValidateChecksums(files)
	assert.NoError(t, err)
	assert.Empty(t, errors)
	assert.Len(t, results, 2)
	assert.True(t, results[tempFile1.Name()])
	assert.True(t, results[tempFile2.Name()])
}

func TestChecksumService_FileIntegrityVerification(t *testing.T) {
	// Create a test database instance
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "integrity_test_*.txt")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	// Write test content to the file
	testContent := "Original test content"
	_, err = tempFile.WriteString(testContent)
	assert.NoError(t, err)
	tempFile.Close()

	// Create checksum service
	service := NewChecksumService(db, nil)

	// Calculate initial checksum
	initialChecksum, err := service.CalculateChecksum(tempFile.Name())
	assert.NoError(t, err)

	// Verify file integrity (should be valid)
	isValid, err := service.VerifyFileIntegrity(tempFile.Name(), initialChecksum)
	assert.NoError(t, err)
	assert.True(t, isValid)

	// Modify the file content
	modifiedContent := "Modified test content"
	err = os.WriteFile(tempFile.Name(), []byte(modifiedContent), 0644)
	assert.NoError(t, err)

	// Verify file integrity (should be invalid now)
	isValid, err = service.VerifyFileIntegrity(tempFile.Name(), initialChecksum)
	assert.NoError(t, err)
	assert.False(t, isValid)
}

func TestChecksumService_IsAlreadyProcessed(t *testing.T) {
	// Create a test database instance
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "processed_test_*.txt")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	// Write test content to the file
	testContent := "Content to test processing status"
	_, err = tempFile.WriteString(testContent)
	assert.NoError(t, err)
	tempFile.Close()

	// Create a test song record
	testSong := &models.Track{
		CRCHash:      "some_hash_value",
		RelativePath: tempFile.Name(),
		Directory:    filepath.Dir(tempFile.Name()),
		FileName:     filepath.Base(tempFile.Name()),
		Name:         "Test Song",
	}

	// Create checksum service
	service := NewChecksumService(db, nil)

	// Initially should not be processed
	isProcessed, existingSong, err := service.IsAlreadyProcessed(tempFile.Name(), "different_hash")
	assert.NoError(t, err)
	assert.False(t, isProcessed)
	assert.Nil(t, existingSong)

	// Test with content-only check
	isProcessed, existingSong, err = service.IsAlreadyProcessedByContentOnly("different_hash")
	assert.NoError(t, err)
	assert.False(t, isProcessed)
	assert.Nil(t, existingSong)
}

func TestChecksumService_CalculateAndVerifyFile(t *testing.T) {
	// Create a test database instance
	db, tearDown := test.GetTestDB(t)
	defer tearDown()

	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "verify_test_*.txt")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	// Write test content to the file
	testContent := "Test content for verification"
	_, err = tempFile.WriteString(testContent)
	assert.NoError(t, err)
	tempFile.Close()

	// Create checksum service
	service := NewChecksumService(db, nil)

	// Calculate and verify with correct expected checksum
	expectedChecksum, err := service.CalculateChecksum(tempFile.Name())
	assert.NoError(t, err)

	calculated, matches, err := service.CalculateAndVerifyFile(tempFile.Name(), expectedChecksum)
	assert.NoError(t, err)
	assert.Equal(t, expectedChecksum, calculated)
	assert.True(t, matches)

	// Calculate and verify with incorrect expected checksum
	calculated, matches, err = service.CalculateAndVerifyFile(tempFile.Name(), "wrong_expected_checksum")
	assert.NoError(t, err)
	assert.Equal(t, expectedChecksum, calculated)
	assert.False(t, matches)
}
