package media

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// FFmpegProfile represents an FFmpeg transcoding profile
type FFmpegProfile struct {
	Name        string `json:"name"`
	CommandLine string `json:"command_line"`
}

// FFmpegConfig holds FFmpeg-related configuration
type FFmpegConfig struct {
	FFmpegPath      string                   `mapstructure:"ffmpeg_path"`
	Profiles        map[string]FFmpegProfile `mapstructure:"profiles"`
	ConcurrentLimit int                      `mapstructure:"concurrent_limit"`
	Timeout         time.Duration            `mapstructure:"timeout"`
}

// DefaultFFmpegConfig returns the default FFmpeg configuration
func DefaultFFmpegConfig() *FFmpegConfig {
	return &FFmpegConfig{
		FFmpegPath: "ffmpeg",
		Profiles: map[string]FFmpegProfile{
			"transcode_high": {
				Name:        "transcode_high",
				CommandLine: "-c:a libmp3lame -b:a 320k -ar 44100 -ac 2",
			},
			"transcode_mid": {
				Name:        "transcode_mid",
				CommandLine: "-c:a libmp3lame -b:a 192k -ar 44100 -ac 2",
			},
			"transcode_opus_mobile": {
				Name:        "transcode_opus_mobile",
				CommandLine: "-c:a libopus -b:a 96k -application audio",
			},
		},
		ConcurrentLimit: 2,
		Timeout:         30 * time.Second,
	}
}

// FFmpegProcessor handles FFmpeg-based media processing
type FFmpegProcessor struct {
	config *FFmpegConfig
}

// NewFFmpegProcessor creates a new FFmpeg processor
func NewFFmpegProcessor(config *FFmpegConfig) *FFmpegProcessor {
	if config == nil {
		config = DefaultFFmpegConfig()
	}
	
	return &FFmpegProcessor{
		config: config,
	}
}

// TranscodeFile transcodes a media file using the specified profile
func (fp *FFmpegProcessor) TranscodeFile(inputPath, outputPath, profileName string) error {
	profile, exists := fp.config.Profiles[profileName]
	if !exists {
		return fmt.Errorf("profile %s not found", profileName)
	}

	// Build the command
	cmdArgs := []string{
		"-i", inputPath, // Input file
	}
	
	// Add profile-specific arguments
	cmdArgs = append(cmdArgs, strings.Split(profile.CommandLine, " ")...)
	
	// Add output file
	cmdArgs = append(cmdArgs, outputPath)

	// Create the command
	cmd := exec.Command(fp.config.FFmpegPath, cmdArgs...)
	
	// Run with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case <-time.After(fp.config.Timeout):
		if err := cmd.Process.Kill(); err != nil {
			fmt.Printf("Failed to kill timed-out process: %v\n", err)
		}
		return fmt.Errorf("transcode timed out after %v", fp.config.Timeout)
	case err := <-done:
		if err != nil {
			return fmt.Errorf("ffmpeg transcoding failed: %w", err)
		}
	}

	return nil
}

// ConvertFile converts a media file from one format to another
func (fp *FFmpegProcessor) ConvertFile(inputPath, outputPath, format string) error {
	// Determine the appropriate command for the output format
	var cmdArgs []string
	
	switch format {
	case "mp3":
		cmdArgs = []string{
			"-i", inputPath,
			"-c:a", "libmp3lame",
			"-b:a", "320k",
			"-ar", "44100",
			"-ac", "2",
			outputPath,
		}
	case "flac":
		cmdArgs = []string{
			"-i", inputPath,
			"-c:a", "flac",
			"-compression_level", "5",
			outputPath,
		}
	case "opus":
		cmdArgs = []string{
			"-i", inputPath,
			"-c:a", "libopus",
			"-b:a", "128k",
			"-application", "audio",
			outputPath,
		}
	case "m4a":
		cmdArgs = []string{
			"-i", inputPath,
			"-c:a", "aac",
			"-b:a", "256k",
			"-strict", "experimental",
			outputPath,
		}
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}

	// Create the command
	cmd := exec.Command(fp.config.FFmpegPath, cmdArgs...)
	
	// Run with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case <-time.After(fp.config.Timeout):
		if err := cmd.Process.Kill(); err != nil {
			fmt.Printf("Failed to kill timed-out process: %v\n", err)
		}
		return fmt.Errorf("convert timed out after %v", fp.config.Timeout)
	case err := <-done:
		if err != nil {
			return fmt.Errorf("ffmpeg conversion failed: %w", err)
		}
	}

	return nil
}

// ExtractAudioFromVideo extracts audio from a video file
func (fp *FFmpegProcessor) ExtractAudioFromVideo(inputPath, outputPath string, format string) error {
	var cmdArgs []string
	
	switch format {
	case "mp3":
		cmdArgs = []string{
			"-i", inputPath,
			"-vn",                    // No video
			"-c:a", "libmp3lame",
			"-b:a", "320k",
			"-ar", "44100",
			"-ac", "2",
			outputPath,
		}
	case "flac":
		cmdArgs = []string{
			"-i", inputPath,
			"-vn",           // No video
			"-c:a", "flac",
			"-compression_level", "5",
			outputPath,
		}
	default:
		return fmt.Errorf("unsupported audio format: %s", format)
	}

	// Create the command
	cmd := exec.Command(fp.config.FFmpegPath, cmdArgs...)
	
	// Run with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case <-time.After(fp.config.Timeout):
		if err := cmd.Process.Kill(); err != nil {
			fmt.Printf("Failed to kill timed-out process: %v\n", err)
		}
		return fmt.Errorf("audio extraction timed out after %v", fp.config.Timeout)
	case err := <-done:
		if err != nil {
			return fmt.Errorf("ffmpeg audio extraction failed: %w", err)
		}
	}

	return nil
}

// GetFileInfo gets information about a media file using FFprobe
func (fp *FFmpegProcessor) GetFileInfo(filePath string) (*FileInfo, error) {
	cmd := exec.Command("ffprobe", 
		"-v", "quiet",
		"-show_format",
		"-show_streams",
		"-print_format", "json",
		filePath,
	)
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	// In a real implementation, we would parse the JSON output
	// For now, return a basic FileInfo struct
	return &FileInfo{
		FilePath: filePath,
		Size:     0, // Would be parsed from JSON
		Duration: 0, // Would be parsed from JSON
	}, nil
}

// FileInfo represents information about a media file
type FileInfo struct {
	FilePath string
	Size     int64
	Duration time.Duration
	BitRate  int
	Format   string
	Streams  []StreamInfo
}

// StreamInfo represents information about a media stream
type StreamInfo struct {
	Index    int
	Type     string // "audio", "video", etc.
	Codec    string
	BitRate  int
	SampleRate int
	Channels int
}

// ProcessArtwork processes artwork using FFmpeg
func (fp *FFmpegProcessor) ProcessArtwork(inputPath, outputPath string, maxWidth, maxHeight int, quality int) error {
	// Command to resize and optimize artwork
	cmdArgs := []string{
		"-i", inputPath,
		"-vf", fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease", maxWidth, maxHeight),
		"-q:v", fmt.Sprintf("%d", quality), // Quality for JPEG (1-31, lower is better)
		"-y", // Overwrite output file if it exists
		outputPath,
	}

	cmd := exec.Command(fp.config.FFmpegPath, cmdArgs...)
	
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case <-time.After(fp.config.Timeout):
		if err := cmd.Process.Kill(); err != nil {
			fmt.Printf("Failed to kill timed-out process: %v\n", err)
		}
		return fmt.Errorf("artwork processing timed out after %v", fp.config.Timeout)
	case err := <-done:
		if err != nil {
			return fmt.Errorf("ffmpeg artwork processing failed: %w", err)
		}
	}

	return nil
}

// CreateMosaic creates a mosaic of multiple images
func (fp *FFmpegProcessor) CreateMosaic(inputImages []string, outputPath string, rows, cols int) error {
	if len(inputImages) == 0 {
		return fmt.Errorf("no input images provided")
	}

	// Create a filter chain for the mosaic
	var inputs string
	for _, img := range inputImages {
		inputs += fmt.Sprintf("-i %s ", img)
	}

	// Build filter complex for the mosaic
	var tileFilter string
	if rows*cols >= len(inputImages) {
		// Use the 'tile' filter
		tileFilter = fmt.Sprintf("tile=%dx%d", cols, rows)
	} else {
		// For complex layouts, we'd need to build a more complex filter
		// For simplicity, we'll just use tile
		tileFilter = fmt.Sprintf("tile=%dx%d", cols, rows)
	}

	cmdArgs := strings.Fields(fmt.Sprintf(
		"%s -filter_complex %s -y %s",
		inputs,
		tileFilter,
		outputPath,
	))

	cmd := exec.Command(fp.config.FFmpegPath, cmdArgs...)
	
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case <-time.After(fp.config.Timeout * 2): // Double timeout for mosaic creation
		if err := cmd.Process.Kill(); err != nil {
			fmt.Printf("Failed to kill timed-out process: %v\n", err)
		}
		return fmt.Errorf("mosaic creation timed out after %v", fp.config.Timeout*2)
	case err := <-done:
		if err != nil {
			return fmt.Errorf("ffmpeg mosaic creation failed: %w", err)
		}
	}

	return nil
}

// ExtractArtwork extracts embedded artwork from audio files
func (fp *FFmpegProcessor) ExtractArtwork(audioPath, artworkPath string) error {
	cmdArgs := []string{
		"-i", audioPath,
		"-map", "0:v", // Map the first video (artwork) stream
		"-c:v", "copy", // Copy the video stream without re-encoding
		"-y", // Overwrite output file if it exists
		artworkPath,
	}

	cmd := exec.Command(fp.config.FFmpegPath, cmdArgs...)
	
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case <-time.After(fp.config.Timeout):
		if err := cmd.Process.Kill(); err != nil {
			fmt.Printf("Failed to kill timed-out process: %v\n", err)
		}
		return fmt.Errorf("artwork extraction timed out after %v", fp.config.Timeout)
	case err := <-done:
		if err != nil {
			// FFmpeg returns an error if no artwork is found, which is not necessarily a failure
			// Check if it's just "No such file or directory" for the artwork output
			if strings.Contains(err.Error(), "Output file #0 does not contain any stream") {
				return fmt.Errorf("no embedded artwork found in file: %s", audioPath)
			}
			return fmt.Errorf("ffmpeg artwork extraction failed: %w", err)
		}
	}

	return nil
}

// NormalizeAudio normalizes audio using loudnorm filter
func (fp *FFmpegProcessor) NormalizeAudio(inputPath, outputPath string) error {
	cmdArgs := []string{
		"-i", inputPath,
		"-af", "loudnorm", // Use loudnorm filter for audio normalization
		"-c:a", "copy", // Preserve original codec where possible
		"-y", // Overwrite output file if it exists
		outputPath,
	}

	cmd := exec.Command(fp.config.FFmpegPath, cmdArgs...)
	
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case <-time.After(fp.config.Timeout):
		if err := cmd.Process.Kill(); err != nil {
			fmt.Printf("Failed to kill timed-out process: %v\n", err)
		}
		return fmt.Errorf("audio normalization timed out after %v", fp.config.Timeout)
	case err := <-done:
		if err != nil {
			return fmt.Errorf("ffmpeg audio normalization failed: %w", err)
		}
	}

	return nil
}

// ProcessGaplessMetadata processes gapless playback metadata
func (fp *FFmpegProcessor) ProcessGaplessMetadata(inputPath, outputPath string, encoderDelay, encoderPadding int) error {
	cmdArgs := []string{
		"-i", inputPath,
		// Add encoder delay and padding metadata for gapless playback
		"-metadata", fmt.Sprintf("encoder_delay=%d", encoderDelay),
		"-metadata", fmt.Sprintf("encoder_padding=%d", encoderPadding),
		"-c", "copy", // Copy streams without re-encoding
		"-y", // Overwrite output file if it exists
		outputPath,
	}

	cmd := exec.Command(fp.config.FFmpegPath, cmdArgs...)
	
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case <-time.After(fp.config.Timeout):
		if err := cmd.Process.Kill(); err != nil {
			fmt.Printf("Failed to kill timed-out process: %v\n", err)
		}
		return fmt.Errorf("gapless metadata processing timed out after %v", fp.config.Timeout)
	case err := <-done:
		if err != nil {
			return fmt.Errorf("ffmpeg gapless metadata processing failed: %w", err)
		}
	}

	return nil
}