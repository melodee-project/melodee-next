package utils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckFFmpeg(t *testing.T) {
	// Test with valid ffmpeg path (this might not work in all environments)
	// We'll test the error case which definitely works
	err := CheckFFmpeg("nonexistent-ffmpeg-command")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to run ffmpeg command")

	// Test with empty path (should try default ffmpeg)
	err = CheckFFmpeg("")
	assert.NoError(t, err) // This one may pass or fail depending on environment
}

func TestCheckExternalTokens(t *testing.T) {
	// Test with no tokens set
	err := CheckExternalTokens()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required environment variables")

	// Set required tokens
	os.Setenv("MELODEE_JWT_SECRET", "test-secret")
	os.Setenv("MELODEE_DB_PASSWORD", "test-db-pass")
	os.Setenv("MELODEE_REDIS_PASSWORD", "test-redis-pass")
	os.Setenv("MELODEE_BOOTSTRAP_ADMIN_PASSWORD", "test-admin-pass")
	defer os.Unsetenv("MELODEE_JWT_SECRET")
	defer os.Unsetenv("MELODEE_DB_PASSWORD")
	defer os.Unsetenv("MELODEE_REDIS_PASSWORD")
	defer os.Unsetenv("MELODEE_BOOTSTRAP_ADMIN_PASSWORD")

	// Now test with all tokens set
	err = CheckExternalTokens()
	assert.NoError(t, err)
}

func TestValidateFFmpegAndTokens(t *testing.T) {
	// Test with no tokens set and nonexistent ffmpeg
	err := ValidateFFmpegAndTokens("nonexistent-ffmpeg-command")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")

	// Set required tokens
	os.Setenv("MELODEE_JWT_SECRET", "test-secret")
	os.Setenv("MELODEE_DB_PASSWORD", "test-db-pass")
	os.Setenv("MELODEE_REDIS_PASSWORD", "test-redis-pass")
	os.Setenv("MELODEE_BOOTSTRAP_ADMIN_PASSWORD", "test-admin-pass")
	defer os.Unsetenv("MELODEE_JWT_SECRET")
	defer os.Unsetenv("MELODEE_DB_PASSWORD")
	defer os.Unsetenv("MELODEE_REDIS_PASSWORD")
	defer os.Unsetenv("MELODEE_BOOTSTRAP_ADMIN_PASSWORD")

	// Test with tokens set but invalid ffmpeg
	err = ValidateFFmpegAndTokens("nonexistent-ffmpeg-command")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ffmpeg validation failed")
	assert.NotContains(t, err.Error(), "external token validation failed")
}