package handlers

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"melodee/internal/models"
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

	// Validate file extension
	extension := strings.ToLower(filepath.Ext(file.Filename))
	if extension != ".jpg" && extension != ".jpeg" && extension != ".png" {
		return utils.SendError(c, http.StatusUnsupportedMediaType, "Invalid file type, only JPEG and PNG are allowed")
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

	// Validate file dimensions (optional, but can prevent extremely large images)
	// This could be done using the image package, but for now we'll just return the validation response
	// In a real implementation, we might want to check dimensions to prevent very large images

	// Generate a unique ID for the image
	imageID := uuid.New().String()

	// Here we would save the file to the appropriate location
	// In a real implementation, this would save to the filesystem and return the ID
	// For now, we'll just validate and return a success response

	// Create a response with the image details
	response := fiber.Map{
		"id":   imageID,
		"etag": fmt.Sprintf("%s-%d", imageID, file.Size), // Simple ETag generation
		"size": file.Size,
		"type": actualMIMEType,
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
	// Check if image exists (in a real implementation, we'd check actual image existence)
	var image models.User // Placeholder - in real implementation would check actual image
	
	// For now, return a placeholder for image retrieval
	// In the future, this would serve the actual image binary with proper headers
	return utils.SendNotFoundError(c, "Image")
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