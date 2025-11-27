package workflow

import (
	"testing"
	"time"

	"melodee/internal/scanner"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLogger implements the Logger interface for testing
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Info(args ...interface{}) {
	m.Called(args)
}

func (m *MockLogger) Infof(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) Warn(args ...interface{}) {
	m.Called(args)
}

func (m *MockLogger) Warnf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) Error(args ...interface{}) {
	m.Called(args)
}

func (m *MockLogger) Errorf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) Debug(args ...interface{}) {
	m.Called(args)
}

func (m *MockLogger) Debugf(format string, args ...interface{}) {
	m.Called(format, args)
}

// TestStagingJobConfig tests the basic configuration
func TestStagingJobConfig(t *testing.T) {
	cfg := StagingJobConfig{
		Workers:       4,
		RateLimit:     10,
		DryRun:        true,
		ScanOutputDir: "/tmp/test",
	}

	assert.Equal(t, 4, cfg.Workers)
	assert.Equal(t, 10, cfg.RateLimit)
	assert.True(t, cfg.DryRun)
	assert.Equal(t, "/tmp/test", cfg.ScanOutputDir)
}

// TestStagingJobServiceCreation tests creating a staging job service
func TestStagingJobServiceCreation(t *testing.T) {
	// For now, just test the function can be called
	// Testing with real DB would require more complex setup
	mockLogger := &MockLogger{}

	service := NewStagingJobService(nil, mockLogger)

	assert.NotNil(t, service)
	assert.Equal(t, mockLogger, service.logger)
}

// TestStagingJobConfigStructure tests the staging job config structure
func TestStagingJobConfigStructure(t *testing.T) {
	cfg := StagingJobConfig{
		Workers:       4,
		RateLimit:     10,
		DryRun:        true,
		ScanOutputDir: "/tmp/test",
	}

	assert.Equal(t, 4, cfg.Workers)
	assert.Equal(t, 10, cfg.RateLimit)
	assert.True(t, cfg.DryRun)
	assert.Equal(t, "/tmp/test", cfg.ScanOutputDir)
}

// TestStagingJobResultStructure tests the staging job result structure
func TestStagingJobResultStructure(t *testing.T) {
	result := &StagingJobResult{
		InboundPath:   "/path/to/inbound",
		StagingPath:   "/path/to/staging",
		ScanDBPath:    "/tmp/scan.db",
		AlbumsTotal:   5,
		AlbumsSuccess: 3,
		AlbumsFailed:  2,
		Duration:      10 * time.Second,
		ProcessedAt:   time.Now(),
		DryRun:        true,
		Error:         nil,
	}

	assert.Equal(t, "/path/to/inbound", result.InboundPath)
	assert.Equal(t, "/path/to/staging", result.StagingPath)
	assert.Equal(t, "/tmp/scan.db", result.ScanDBPath)
	assert.Equal(t, 5, result.AlbumsTotal)
	assert.Equal(t, 3, result.AlbumsSuccess)
	assert.Equal(t, 2, result.AlbumsFailed)
	assert.Equal(t, 10*time.Second, result.Duration)
	assert.Equal(t, true, result.DryRun)
	assert.Nil(t, result.Error)
}

// TestAlbumGroupResolution tests that we can work with album groups 
func TestAlbumGroupResolution(t *testing.T) {
	// Create a temporary directory for scan DB
	tempDir := t.TempDir()

	// Create a scan DB
	scanDB, err := scanner.NewScanDB(tempDir)
	assert.NoError(t, err)
	defer scanDB.Close()

	// Test that we can get empty album groups
	groups, err := scanDB.GetAlbumGroups()
	assert.NoError(t, err)
	// The scanner returns nil when there are no groups, not an empty slice
	if groups == nil {
		assert.Equal(t, 0, 0) // Just verify that we got a nil slice (which is correct)
	} else {
		assert.Equal(t, 0, len(groups))
	}
}