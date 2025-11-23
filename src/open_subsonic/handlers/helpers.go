package handlers

import (
	"path/filepath"
	"strconv"
	"strings"
)

// getContentType determines the content type based on file extension
func getContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".mp3":
		return "audio/mpeg"
	case ".flac":
		return "audio/flac"
	case ".ogg", ".oga":
		return "audio/ogg"
	case ".m4a", ".mp4":
		return "audio/mp4"
	case ".wav":
		return "audio/wav"
	case ".opus":
		return "audio/opus"
	case ".wma":
		return "audio/x-ms-wma"
	case ".aac":
		return "audio/aac"
	default:
		return "audio/mpeg" // Default fallback
	}
}

// getSuffix extracts the file extension/suffix from a filename
func getSuffix(filename string) string {
	ext := filepath.Ext(filename)
	if len(ext) > 0 {
		return ext[1:] // Remove the leading dot
	}
	return ""
}

// getCoverArtID generates a cover art ID from an album or artist ID
func getCoverArtID(entityType string, id int64) string {
	switch entityType {
	case "album", "al":
		return "al-" + strconv.FormatInt(id, 10)
	case "artist", "ar":
		return "ar-" + strconv.FormatInt(id, 10)
	default:
		return strconv.FormatInt(id, 10)
	}
}
