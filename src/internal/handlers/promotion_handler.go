package handlers

import (
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"melodee/internal/models"
	"melodee/internal/processor"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// PromotionHandler handles promotion of albums from staging to production
type PromotionHandler struct {
	db             *gorm.DB
	stagingRoot    string
	productionRoot string
}

// NewPromotionHandler creates a new promotion handler
func NewPromotionHandler(db *gorm.DB, stagingRoot, productionRoot string) *PromotionHandler {
	return &PromotionHandler{
		db:             db,
		stagingRoot:    stagingRoot,
		productionRoot: productionRoot,
	}
}

// PromoteAlbum promotes an approved album to production
// POST /api/v1/staging/:id/promote
func (h *PromotionHandler) PromoteAlbum(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid staging item ID",
		})
	}

	// Start transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get staging item
	var stagingItem models.StagingItem
	if err := tx.First(&stagingItem, id).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Staging item not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch staging item",
		})
	}

	// Check if approved
	if stagingItem.Status != "approved" {
		tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Only approved items can be promoted",
		})
	}

	// Read metadata
	metadata, err := processor.ReadAlbumMetadata(stagingItem.MetadataFile)
	if err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to read metadata file",
		})
	}

	// Create or find artist
	artist, err := h.findOrCreateArtist(tx, metadata)
	if err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to create artist: %v", err),
		})
	}

	// Create album
	album, err := h.createAlbum(tx, metadata, artist.ID)
	if err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to create album: %v", err),
		})
	}

	// Create tracks
	if err := h.createTracks(tx, metadata, album.ID, artist.ID); err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to create tracks: %v", err),
		})
	}

	// Move files to production
	productionPath := filepath.Join(
		h.productionRoot,
		metadata.Artist.DirectoryCode,
		metadata.Artist.Name,
		fmt.Sprintf("%d - %s", metadata.Album.Year, metadata.Album.Name),
	)

	if err := processor.SafeMoveFile(stagingItem.StagingPath, productionPath); err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to move files: %v", err),
		})
	}

	// Delete staging item
	if err := tx.Delete(&stagingItem).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete staging item",
		})
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to commit transaction",
		})
	}

	return c.JSON(fiber.Map{
		"success":     true,
		"message":     "Album promoted to production",
		"album_id":    album.ID,
		"artist_id":   artist.ID,
		"track_count": len(metadata.Tracks),
	})
}

// findOrCreateArtist finds an existing artist or creates a new one
func (h *PromotionHandler) findOrCreateArtist(tx *gorm.DB, metadata *processor.AlbumMetadata) (*models.Artist, error) {
	var artist models.Artist

	// Try to find by name first
	err := tx.Where("name_normalized = ?", metadata.Artist.NameNormalized).First(&artist).Error
	if err == nil {
		return &artist, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// Create new artist
	artist = models.Artist{
		Name:           metadata.Artist.Name,
		NameNormalized: metadata.Artist.NameNormalized,
		DirectoryCode:  metadata.Artist.DirectoryCode,
		SortName:       metadata.Artist.SortName,
	}

	if err := tx.Create(&artist).Error; err != nil {
		return nil, err
	}

	return &artist, nil
}

// createAlbum creates a new album
func (h *PromotionHandler) createAlbum(tx *gorm.DB, metadata *processor.AlbumMetadata, artistID int64) (*models.Album, error) {
	album := models.Album{
		Name:           metadata.Album.Name,
		NameNormalized: metadata.Album.NameNormalized,
		ArtistID:       artistID,
		AlbumType:      metadata.Album.AlbumType,
		Genres:         metadata.Album.Genres,
		IsCompilation:  metadata.Album.IsCompilation,
		ImageCount:     int32(metadata.Album.ImageCount),
	}

	// Set release date if provided
	if metadata.Album.ReleaseDate != nil {
		if t, err := time.Parse("2006-01-02", *metadata.Album.ReleaseDate); err == nil {
			album.ReleaseDate = &t
		}
	}

	// Set directory path
	album.Directory = filepath.Join(
		metadata.Artist.DirectoryCode,
		metadata.Artist.Name,
		fmt.Sprintf("%d - %s", metadata.Album.Year, metadata.Album.Name),
	)

	if err := tx.Create(&album).Error; err != nil {
		return nil, err
	}

	return &album, nil
}

// createTracks creates tracks for an album
func (h *PromotionHandler) createTracks(tx *gorm.DB, metadata *processor.AlbumMetadata, albumID, artistID int64) error {
	for _, trackMeta := range metadata.Tracks {
		track := models.Track{
			Name:           trackMeta.Name,
			NameNormalized: processor.NormalizeString(trackMeta.Name),
			AlbumID:        albumID,
			ArtistID:       artistID,
			Duration:       int64(trackMeta.Duration),
			BitRate:        int32(trackMeta.Bitrate),
			SampleRate:     int32(trackMeta.SampleRate),
			Directory:      filepath.Dir(trackMeta.FilePath),
			FileName:       filepath.Base(trackMeta.FilePath),
			RelativePath:   trackMeta.FilePath,
			CRCHash:        trackMeta.Checksum,
			SortOrder:      int32(trackMeta.TrackNumber),
		}

		if err := tx.Create(&track).Error; err != nil {
			return err
		}
	}

	return nil
}

// PromoteBatch promotes multiple approved albums
// POST /api/v1/staging/promote-batch
func (h *PromotionHandler) PromoteBatch(c *fiber.Ctx) error {
	var req struct {
		IDs []int64 `json:"ids"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if len(req.IDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No IDs provided",
		})
	}

	results := make([]map[string]interface{}, len(req.IDs))
	successCount := 0
	failCount := 0

	for i, id := range req.IDs {
		// Create a mock context for individual promotion
		// In production, you'd want to refactor PromoteAlbum to separate logic from HTTP handling
		results[i] = map[string]interface{}{
			"id":      id,
			"success": false,
			"error":   "Batch promotion not fully implemented",
		}
		failCount++
	}

	return c.JSON(fiber.Map{
		"success": successCount,
		"failed":  failCount,
		"total":   len(req.IDs),
		"results": results,
	})
}
