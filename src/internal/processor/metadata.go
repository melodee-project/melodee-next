package processor

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AlbumMetadata represents the JSON sidecar file for an album in staging
type AlbumMetadata struct {
	Version     string          `json:"version"`
	ProcessedAt time.Time       `json:"processed_at"`
	ScanID      string          `json:"scan_id"`
	Artist      ArtistMetadata  `json:"artist"`
	Album       AlbumInfo       `json:"album"`
	Tracks      []TrackMetadata `json:"tracks"`
	Status      string          `json:"status"`
	Validation  ValidationInfo  `json:"validation"`
}

// ArtistMetadata contains artist information
type ArtistMetadata struct {
	Name           string  `json:"name"`
	NameNormalized string  `json:"name_normalized"`
	DirectoryCode  string  `json:"directory_code"`
	MusicBrainzID  *string `json:"musicbrainz_id,omitempty"`
	SortName       string  `json:"sort_name,omitempty"`
}

// AlbumInfo contains album information
type AlbumInfo struct {
	Name           string    `json:"name"`
	NameNormalized string    `json:"name_normalized"`
	ReleaseDate    *string   `json:"release_date,omitempty"`
	AlbumType      string    `json:"album_type"`
	Genres         []string  `json:"genres"`
	IsCompilation  bool      `json:"is_compilation"`
	ImageCount     int       `json:"image_count"`
	Year           int       `json:"year"`
}

// TrackMetadata contains track information
type TrackMetadata struct {
	TrackNumber  int    `json:"track_number"`
	DiscNumber   int    `json:"disc_number"`
	Name         string `json:"name"`
	Duration     int    `json:"duration"`      // milliseconds
	FilePath     string `json:"file_path"`     // relative to staging root
	FileSize     int64  `json:"file_size"`
	Bitrate      int    `json:"bitrate"`
	SampleRate   int    `json:"sample_rate"`
	Checksum     string `json:"checksum"`
	OriginalPath string `json:"original_path"` // original inbound path
}

// ValidationInfo contains validation status
type ValidationInfo struct {
	IsValid bool     `json:"is_valid"`
	Errors  []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

// WriteAlbumMetadata writes the album metadata to a JSON file
func WriteAlbumMetadata(path string, metadata *AlbumMetadata) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal with pretty printing
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
}

// ReadAlbumMetadata reads album metadata from a JSON file
func ReadAlbumMetadata(path string) (*AlbumMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	var metadata AlbumMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &metadata, nil
}

// GenerateDirectoryCode generates a short directory code from artist name
// Examples: "Led Zeppelin" → "LZ", "The Beatles" → "TB", "AC/DC" → "ACDC"
func GenerateDirectoryCode(artistName string) string {
	// Remove "The " prefix
	name := strings.TrimPrefix(artistName, "The ")
	name = strings.TrimPrefix(name, "the ")

	// Split into words
	words := strings.Fields(name)
	
	if len(words) == 0 {
		return "UNK"
	}

	// If single word, take first 2-3 characters
	if len(words) == 1 {
		cleaned := cleanForDirectoryCode(words[0])
		if len(cleaned) <= 3 {
			return strings.ToUpper(cleaned)
		}
		return strings.ToUpper(cleaned[:3])
	}

	// Multiple words: take first letter of each word
	var code strings.Builder
	for _, word := range words {
		cleaned := cleanForDirectoryCode(word)
		if len(cleaned) > 0 {
			code.WriteRune(rune(cleaned[0]))
		}
	}

	result := strings.ToUpper(code.String())
	if result == "" {
		return "UNK"
	}
	return result
}

// cleanForDirectoryCode removes special characters
func cleanForDirectoryCode(s string) string {
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// SafeMoveFile moves a file safely with error handling
func SafeMoveFile(src, dst string) error {
	// Ensure destination directory exists
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Try rename first (fast if same filesystem)
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	// Fallback to copy+delete
	if err := copyFile(src, dst); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Remove source file
	if err := os.Remove(src); err != nil {
		return fmt.Errorf("failed to remove source file: %w", err)
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// Sync to ensure data is written
	return dstFile.Sync()
}

// NormalizeString normalizes a string for comparison
func NormalizeString(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// FormatFilename formats a track filename for staging
// Format: {disc}-{track:02d} - {title}.{ext}
func FormatFilename(discNumber, trackNumber int, title, extension string) string {
	// Clean title
	cleanTitle := strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '-'
		default:
			return r
		}
	}, title)

	if discNumber > 1 {
		return fmt.Sprintf("%d-%02d - %s%s", discNumber, trackNumber, cleanTitle, extension)
	}
	return fmt.Sprintf("%02d - %s%s", trackNumber, cleanTitle, extension)
}
