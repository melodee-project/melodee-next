package handlers

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"melodee/internal/media"
	"melodee/internal/models"
	"melodee/open_subsonic/utils"
)

// MediaHandler handles OpenSubsonic media retrieval endpoints
type MediaHandler struct {
	db               *gorm.DB
	cfg              interface{} // Placeholder for config
	transcodeService *media.TranscodeService
}

// NewMediaHandler creates a new media handler
func NewMediaHandler(db *gorm.DB, cfg interface{}, transcodeService *media.TranscodeService) *MediaHandler {
	return &MediaHandler{
		db:               db,
		cfg:              cfg,
		transcodeService: transcodeService,
	}
}

// Stream handles audio streaming
func (h *MediaHandler) Stream(c *fiber.Ctx) error {
	id := c.QueryInt("id", -1)
	if id <= 0 {
		return utils.SendOpenSubsonicError(c, 10, "Missing required parameter id")
	}

	// Get the song
	var song models.Song
	if err := h.db.Preload("Album").Preload("Artist").First(&song, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return utils.SendOpenSubsonicError(c, 70, "Song not found")
		}
		return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve song")
	}

	// Check if file exists on disk
	// In a real implementation, we would look up the file path from the database
	// For demo purposes, we'll try to construct the path
	storagePath := "/melodee/storage" // Default from spec
	fullPath := filepath.Join(storagePath, song.RelativePath)

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return utils.SendOpenSubsonicError(c, 70, "File not found")
	}

	// Handle range requests for partial content
	rangeHeader := c.Get("Range")
	if rangeHeader != "" {
		return h.handleRangeRequest(c, fullPath, song)
	}

	// Apply transcoding if required
	maxBitRate := c.QueryInt("maxBitRate", 0)
	format := c.Query("format", "")
	
	if maxBitRate > 0 || format != "" {
		// In a real implementation, we would use FFmpeg transcoding
		// Here's a placeholder for transcoding functionality:
		transcodedPath, err := h.transcodeFile(fullPath, maxBitRate, format)
		if err != nil {
			return utils.SendOpenSubsonicError(c, 0, "Transcoding failed: "+err.Error())
		}
		if transcodedPath != "" {
			return c.SendFile(transcodedPath)
		}
	}

	// Add ETag and Last-Modified headers for caching
	if fileInfo, err := os.Stat(fullPath); err == nil {
		etag := fmt.Sprintf(`"%x"`, fileInfo.ModTime().Unix())
		c.Set("ETag", etag)
		c.Set("Last-Modified", fileInfo.ModTime().Format(http.TimeFormat))
		
		// Check if client has a cached version
		ifMatch := c.Get("If-None-Match")
		if ifMatch == etag {
			return c.SendStatus(304) // Not modified
		}
	}

	// Stream the full file
	return c.SendFile(fullPath)
}

// Download handles file downloads
func (h *MediaHandler) Download(c *fiber.Ctx) error {
	id := c.QueryInt("id", -1)
	if id <= 0 {
		return utils.SendOpenSubsonicError(c, 10, "Missing required parameter id")
	}

	// Get the song
	var song models.Song
	if err := h.db.Preload("Album").Preload("Artist").First(&song, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return utils.SendOpenSubsonicError(c, 70, "Song not found")
		}
		return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve song")
	}

	// Check if file exists on disk
	storagePath := "/melodee/storage"
	fullPath := filepath.Join(storagePath, song.RelativePath)

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return utils.SendOpenSubsonicError(c, 70, "File not found")
	}

	// Add ETag and Last-Modified headers for caching
	if fileInfo, err := os.Stat(fullPath); err == nil {
		etag := fmt.Sprintf(`"%x"`, fileInfo.ModTime().Unix())
		c.Set("ETag", etag)
		c.Set("Last-Modified", fileInfo.ModTime().Format(http.TimeFormat))
		
		// Check if client has a cached version
		ifMatch := c.Get("If-None-Match")
		if ifMatch == etag {
			return c.SendStatus(304) // Not modified
		}
	}

	// Set content disposition for download
	c.Set("Content-Disposition", "attachment; filename="+strconv.Quote(filepath.Base(fullPath)))

	return c.SendFile(fullPath)
}

// GetCoverArt returns cover art for an album or artist
func (h *MediaHandler) GetCoverArt(c *fiber.Ctx) error {
	// Get id from query
	id := c.Query("id", "")
	if id == "" {
		return utils.SendOpenSubsonicError(c, 10, "Missing required parameter id")
	}

	// Check if it's an album or artist cover art request
	// The ID format is typically "al-<albumId>" or "ar-<artistId>"
	var coverPath string
	
	if strings.HasPrefix(id, "al-") {
		// Album cover art request
		albumID := strings.TrimPrefix(id, "al-")
		albumIDInt, err := strconv.Atoi(albumID)
		if err != nil {
			return utils.SendOpenSubsonicError(c, 10, "Invalid album id format")
		}

		// Get the album and find its cover art
		var album models.Album
		if err := h.db.First(&album, albumIDInt).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return utils.SendOpenSubsonicError(c, 70, "Album not found")
			}
			return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve album")
		}

		// In a real implementation, we'd look for cover.jpg in the album directory
		// For demo purposes, we'll construct a path
		storagePath := "/melodee/storage"
		coverPath = filepath.Join(storagePath, filepath.Dir(album.Directory), "cover.jpg")
	} else if strings.HasPrefix(id, "ar-") {
		// Artist cover art request
		artistID := strings.TrimPrefix(id, "ar-")
		artistIDInt, err := strconv.Atoi(artistID)
		if err != nil {
			return utils.SendOpenSubsonicError(c, 10, "Invalid artist id format")
		}

		// Get the artist
		var artist models.Artist
		if err := h.db.First(&artist, artistIDInt).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return utils.SendOpenSubsonicError(c, 70, "Artist not found")
			}
			return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve artist")
		}

		// For artist covers, we may need to get the first album's cover or a dedicated artist image
		// In this simplified version, we'll look for a default artist image
		// This would be implemented based on system configuration
		coverPath = filepath.Join("/melodee/storage", artist.DirectoryCode, artist.Name, "folder.jpg")
	} else {
		// If no prefix, assume it's just the album ID
		albumID, err := strconv.Atoi(id)
		if err == nil {
			// Get the album and find its cover art
			var album models.Album
			if err := h.db.First(&album, albumID).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					return utils.SendOpenSubsonicError(c, 70, "Album not found")
				}
				return utils.SendOpenSubsonicError(c, 0, "Failed to retrieve album")
			}

			storagePath := "/melodee/storage"
			coverPath = filepath.Join(storagePath, filepath.Dir(album.Directory), "cover.jpg")
		} else {
			return utils.SendOpenSubsonicError(c, 10, "Invalid id format")
		}
	}

	// Check if cover art file exists
	if _, err := os.Stat(coverPath); os.IsNotExist(err) {
		// If the specific cover doesn't exist, try alternative names
		altPaths := []string{
			strings.Replace(coverPath, "cover.jpg", "folder.jpg", 1),
			strings.Replace(coverPath, "cover.jpg", "front.jpg", 1),
		}

		found := false
		for _, altPath := range altPaths {
			if _, err := os.Stat(altPath); err == nil {
				coverPath = altPath
				found = true
				break
			}
		}

		if !found {
			return utils.SendOpenSubsonicError(c, 70, "Cover art not found")
		}
	}

	// Add ETag and Last-Modified headers for caching
	if fileInfo, err := os.Stat(coverPath); err == nil {
		etag := fmt.Sprintf(`"%x"`, fileInfo.ModTime().Unix())
		c.Set("ETag", etag)
		c.Set("Last-Modified", fileInfo.ModTime().Format(http.TimeFormat))
		
		// Check if client has a cached version
		ifMatch := c.Get("If-None-Match")
		if ifMatch == etag {
			return c.SendStatus(304) // Not modified
		}
	}

	return c.SendFile(coverPath)
}

// GetAvatar returns user avatar
func (h *MediaHandler) GetAvatar(c *fiber.Ctx) error {
	username := c.Query("username", "")
	
	// If no username provided, use the authenticated user
	if username == "" {
		// In a real implementation, we'd get this from the auth context
		return utils.SendOpenSubsonicError(c, 50, "not authorized")
	}

	// Construct avatar path
	// In a real implementation, we'd look up the specific user and their avatar
	avatarPath := filepath.Join("/melodee/user_images", username+".jpg")
	
	// Check if avatar exists
	if _, err := os.Stat(avatarPath); os.IsNotExist(err) {
		// Try other common avatar formats
		altPaths := []string{
			strings.Replace(avatarPath, ".jpg", ".png", 1),
			strings.Replace(avatarPath, ".jpg", ".gif", 1),
		}

		found := false
		for _, altPath := range altPaths {
			if _, err := os.Stat(altPath); err == nil {
				avatarPath = altPath
				found = true
				break
			}
		}

		if !found {
			return utils.SendOpenSubsonicError(c, 70, "Avatar not found")
		}
	}

	// Add ETag and Last-Modified headers for caching
	if fileInfo, err := os.Stat(avatarPath); err == nil {
		etag := fmt.Sprintf(`"%x"`, fileInfo.ModTime().Unix())
		c.Set("ETag", etag)
		c.Set("Last-Modified", fileInfo.ModTime().Format(http.TimeFormat))
		
		// Check if client has a cached version
		ifMatch := c.Get("If-None-Match")
		if ifMatch == etag {
			return c.SendStatus(304) // Not modified
		}
	}

	return c.SendFile(avatarPath)
}

// transcodeFile handles transcoding using FFmpeg with caching
func (h *MediaHandler) transcodeFile(filePath string, maxBitRate int, format string) (string, error) {
	// Use the transcode service if available
	if h.transcodeService == nil {
		// Fallback to original behavior if no transcode service is configured
		fmt.Printf("No transcode service configured, returning original file: %s\n", filePath)
		return filePath, nil
	}

	// Determine the best profile based on maxBitRate
	profileName := "transcode_mid" // Default profile
	if maxBitRate > 0 {
		if maxBitRate > 256 {
			profileName = "transcode_high"
		} else if maxBitRate < 128 {
			profileName = "transcode_opus_mobile"
		}
	} else {
		// If maxBitRate is 0 or negative, use default
		maxBitRate = 192 // Default bitrate
	}

	// If format is empty, try to determine from file extension
	if format == "" {
		ext := strings.ToLower(filepath.Ext(filePath))
		switch ext {
		case ".mp3":
			format = "mp3"
		case ".flac":
			format = "flac"
		case ".ogg", ".opus":
			format = "opus"
		case ".m4a":
			format = "m4a"
		default:
			format = "mp3" // Default format
		}
	}

	// Use cached transcoding
	outputPath, err := h.transcodeService.TranscodeWithCache(filePath, profileName, maxBitRate, format)
	if err != nil {
		return "", fmt.Errorf("transcoding failed: %w", err)
	}

	return outputPath, nil
}

// handleRangeRequest handles HTTP range requests for partial content
func (h *MediaHandler) handleRangeRequest(c *fiber.Ctx, filePath string, song models.Song) error {
	rangeHeader := c.Get("Range")
	
	// Parse the range header (format: "bytes=start-end" or "bytes=start-" for range from start to end)
	if rangeHeader != "" && strings.HasPrefix(rangeHeader, "bytes=") {
		rangeStr := strings.TrimPrefix(rangeHeader, "bytes=")
		parts := strings.Split(rangeStr, "-")
		
		if len(parts) == 2 {
			var start, end int64
			var err error
			
			if parts[0] != "" {
				start, err = strconv.ParseInt(parts[0], 10, 64)
				if err != nil {
					return c.Status(416).SendString("Range Not Satisfiable")
				}
			} else {
				start = 0 // If start is not specified, start from the beginning
			}
			
			if parts[1] != "" {
				end, err = strconv.ParseInt(parts[1], 10, 64)
				if err != nil {
					return c.Status(416).SendString("Range Not Satisfiable")
				}
			} else {
				// If end is not specified, use the file size - 1
				if fileInfo, err := os.Stat(filePath); err == nil {
					end = fileInfo.Size() - 1
				} else {
					return c.Status(500).SendString("Internal Server Error")
				}
			}
			
			// Validate range
			if start < 0 || end < start {
				return c.Status(416).SendString("Range Not Satisfiable")
			}
			
			// Open the file
			file, err := os.Open(filePath)
			if err != nil {
				return c.Status(500).SendString("Internal Server Error")
			}
			defer file.Close()
			
			// Get file info for Content-Length calculation
			fileInfo, err := file.Stat()
			if err != nil {
				return c.Status(500).SendString("Internal Server Error")
			}
			
			// Adjust end if it's beyond the file size
			totalSize := fileInfo.Size()
			if end >= totalSize {
				end = totalSize - 1
			}
			
			// Calculate content length
			contentLength := end - start + 1
			
			// Set the file pointer to the start position
			_, err = file.Seek(start, 0)
			if err != nil {
				return c.Status(500).SendString("Internal Server Error")
			}
			
			// Set response headers
			contentRange := fmt.Sprintf("bytes %d-%d/%d", start, end, totalSize)
			c.Set("Content-Range", contentRange)
			c.Set("Content-Length", strconv.FormatInt(contentLength, 10))
			c.Set("Content-Type", getContentType(filepath.Base(filePath))) // Determine content type
			c.Set("Accept-Ranges", "bytes")
			
			// Create a limited reader that only reads the requested amount
			limitedReader := io.LimitReader(file, contentLength)
			
			// Send the partial content with correct status code
			c.Set("X-Status-Code", "206")
			return c.Status(206).SendStream(limitedReader)
		}
	}
	
	// If we reach here, the range header was malformed
	return c.Status(416).SendString("Range Not Satisfiable")
}

// getContentType returns content type based on file extension
func getContentType(filename string) string {
	switch {
	case strings.HasSuffix(strings.ToLower(filename), ".mp3"):
		return "audio/mpeg"
	case strings.HasSuffix(strings.ToLower(filename), ".flac"):
		return "audio/flac"
	case strings.HasSuffix(strings.ToLower(filename), ".m4a"):
		return "audio/mp4"
	case strings.HasSuffix(strings.ToLower(filename), ".mp4"):
		return "audio/mp4"
	case strings.HasSuffix(strings.ToLower(filename), ".aac"):
		return "audio/aac"
	case strings.HasSuffix(strings.ToLower(filename), ".ogg"):
		return "audio/ogg"
	case strings.HasSuffix(strings.ToLower(filename), ".opus"):
		return "audio/opus"
	case strings.HasSuffix(strings.ToLower(filename), ".wav"):
		return "audio/wav"
	case strings.HasSuffix(strings.ToLower(filename), ".jpg"):
		return "image/jpeg"
	case strings.HasSuffix(strings.ToLower(filename), ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(strings.ToLower(filename), ".png"):
		return "image/png"
	case strings.HasSuffix(strings.ToLower(filename), ".gif"):
		return "image/gif"
	default:
		return "application/octet-stream"
	}
}