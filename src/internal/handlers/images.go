package handlers

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"melodee/internal/services"
	"melodee/internal/utils"
)

// ImageHandler handles image-related requests
type ImageHandler struct {
	repo *services.Repository
}

// NewImageHandler creates a new image handler
func NewImageHandler(repo *services.Repository) *ImageHandler {
	return &ImageHandler{
		repo: repo,
	}
}

// UploadAvatar handles avatar upload requests
func (h *ImageHandler) UploadAvatar(c *fiber.Ctx) error {
	// Get the uploaded file from the multipart form
	file, err := c.FormFile("file")
	if err != nil {
		return utils.SendError(c, http.StatusBadRequest, "No file provided or invalid form data")
	}

	// Validate file size (max 2MB as per spec)
	maxSize := int64(2 * 1024 * 1024) // 2MB
	if file.Size > maxSize {
		return utils.SendError(c, http.StatusRequestEntityTooLarge, fmt.Sprintf("File size too large, maximum %d bytes allowed", maxSize))
	}

	// Validate file extension to prevent double extension attacks
	extension := strings.ToLower(filepath.Ext(file.Filename))
	if extension != ".jpg" && extension != ".jpeg" && extension != ".png" {
		return utils.SendError(c, http.StatusUnsupportedMediaType, "Invalid file type, only JPEG and PNG are allowed")
	}

	// Get the original filename without path for security
	filename := filepath.Base(file.Filename)
	if filename == "" || strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return utils.SendError(c, http.StatusBadRequest, "Invalid filename")
	}

	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to open uploaded file")
	}
	defer src.Close()

	// Read the file content to determine the actual MIME type (prevent MIME sniffing attacks and double extension issues)
	fileBytes := make([]byte, 512) // Read first 512 bytes to detect MIME type
	_, err = io.ReadFull(src, fileBytes)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return utils.SendInternalServerError(c, "Failed to read uploaded file")
	}

	// Determine the actual MIME type from the file content (not just extension)
	actualMIMEType := http.DetectContentType(fileBytes)
	if actualMIMEType != "image/jpeg" && actualMIMEType != "image/png" {
		return utils.SendError(c, http.StatusUnsupportedMediaType, "Invalid file type. Detected MIME type does not match allowed types. Only JPEG and PNG files are allowed.")
	}

	// Additional check: Validate the file header based on expected format
	if !isValidImageFile(fileBytes, actualMIMEType) {
		return utils.SendError(c, http.StatusUnsupportedMediaType, "Invalid file content. File has been tampered with or is not a valid image.")
	}

	// Reset file pointer to the beginning
	_, err = src.Seek(0, io.SeekStart)
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to reset file pointer")
	}

	// Validate image dimensions to prevent large memory allocation attacks
	imageConfig, _, err := image.DecodeConfig(src)
	if err != nil {
		return utils.SendError(c, http.StatusUnsupportedMediaType, "Could not decode image file.")
	}

	// Limit dimensions to prevent memory denial of service
	maxWidth := 2000  // pixels
	maxHeight := 2000 // pixels
	if imageConfig.Width > maxWidth || imageConfig.Height > maxHeight {
		return utils.SendError(c, http.StatusRequestEntityTooLarge, fmt.Sprintf("Image dimensions too large. Maximum allowed: %dx%d pixels", maxWidth, maxHeight))
	}

	// Additional security check: ensure the file is not too large in dimensions
	if imageConfig.Width*imageConfig.Height > 5000000 { // 5 megapixels max
		return utils.SendError(c, http.StatusRequestEntityTooLarge, "Image resolution too high. Maximum allowed: 5 megapixels")
	}

	// Reset file pointer again after reading dimensions
	_, err = src.Seek(0, io.SeekStart)
	if err != nil {
		return utils.SendInternalServerError(c, "Failed to reset file pointer")
	}

	// Generate a unique ID for the image
	imageID := uuid.New().String()

	// Here we would save the file to the appropriate location
	// In a real implementation, this would save to the filesystem and return the ID
	// For now, we'll just validate and return a success response

	// Create a response with the image details
	response := fiber.Map{
		"id":   imageID,
		"name": filename, // Include safe filename
		"etag": fmt.Sprintf("%s-%d", imageID, file.Size), // Simple ETag generation
		"size": file.Size,
		"type": actualMIMEType,
		"width": imageConfig.Width,
		"height": imageConfig.Height,
	}

	return c.JSON(response)
}

// GetImage handles image retrieval requests
func (h *ImageHandler) GetImage(c *fiber.Ctx) error {
	imageID := c.Params("id")

	// Validate image ID format (should be UUID)
	if _, err := uuid.Parse(imageID); err != nil {
		return utils.SendError(c, http.StatusBadRequest, "Invalid image ID format")
	}

	// In a real implementation, this would fetch the image from the filesystem or database
	// For now, return a placeholder response with proper ETag/Last-Modified headers
	// This method should serve the actual image binary with appropriate headers

	// Set appropriate headers for image response
	c.Set("Content-Type", "image/jpeg") // Default - would be determined by actual image type
	c.Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour

	// Placeholder - in real implementation, this would serve actual image data
	// For now, returning a 404 to indicate the image doesn't exist
	return utils.SendNotFoundError(c, "Image not found")
}

// isValidImageFile validates that the file bytes represent a valid image based on the expected MIME type
func isValidImageFile(fileBytes []byte, expectedMIMEType string) bool {
	// Create a reader from the first 512 bytes
	reader := bytes.NewReader(fileBytes)

	// Check for JPEG files
	if expectedMIMEType == "image/jpeg" {
		_, err := jpeg.DecodeConfig(reader)
		return err == nil
	}

	// Check for PNG files
	if expectedMIMEType == "image/png" {
		_, err := png.DecodeConfig(reader)
		return err == nil
	}

	// For other types, we could add additional validation logic
	return false
}