package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CheckFFmpeg verifies that FFmpeg is available and accessible
func CheckFFmpeg(ffmpegPath string) error {
	// If no path provided, use default "ffmpeg"
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}

	// Check if the command exists and can be executed
	cmd := exec.Command(ffmpegPath, "-version")
	err := cmd.Run()
	
	if err != nil {
		// Try to find ffmpeg in PATH if the default wasn't found
		if ffmpegPath == "ffmpeg" {
			// Try to locate ffmpeg in system PATH
			if _, lookErr := exec.LookPath("ffmpeg"); lookErr != nil {
				return fmt.Errorf("ffmpeg not found in system PATH: %w", lookErr)
			}
		}
		return fmt.Errorf("failed to run ffmpeg command: %w", err)
	}

	return nil
}

// CheckExternalTokens checks for required external tokens/environment variables
func CheckExternalTokens() error {
	requiredTokens := []string{
		"MELODEE_JWT_SECRET",
		"MELODEE_DB_PASSWORD",
		"MELODEE_REDIS_PASSWORD",
		"MELODEE_BOOTSTRAP_ADMIN_PASSWORD", // For first-run admin setup
	}

	var missingTokens []string
	for _, token := range requiredTokens {
		if os.Getenv(token) == "" {
			missingTokens = append(missingTokens, token)
		}
	}

	if len(missingTokens) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missingTokens, ", "))
	}

	return nil
}

// ValidateFFmpegAndTokens validates both FFmpeg and required tokens
func ValidateFFmpegAndTokens(ffmpegPath string) error {
	// Check FFmpeg availability
	if err := CheckFFmpeg(ffmpegPath); err != nil {
		return fmt.Errorf("ffmpeg validation failed: %w", err)
	}

	// Check external tokens
	if err := CheckExternalTokens(); err != nil {
		return fmt.Errorf("external token validation failed: %w", err)
	}

	return nil
}