package media

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"melodee/internal/directory"
)

// Job types for Asynq
const (
	TypeLibraryScan          = "library:scan"
	TypeLibraryProcess       = "library:process"
	TypeLibraryMoveOK        = "library:move_ok"
	TypeDirectoryRecalculate = "directory:recalculate"
	TypeMetadataWriteback    = "metadata:writeback"
	TypeMetadataEnhance      = "metadata:enhance"
	TypeStagingScan          = "staging:scan" // renamed from TypeStagingCron
)

// Job payloads and handlers

// LibraryScanPayload represents the payload for library scan jobs
type LibraryScanPayload struct {
	LibraryIDs []int32 `json:"library_ids"`
	Force      bool    `json:"force"`
}

// TaskHandler provides access to dependencies for task handlers
type TaskHandler struct {
	db                *gorm.DB
	directorySvc      *directory.DirectoryCodeGenerator
	pathResolver      *directory.PathTemplateResolver
	quarantineService *QuarantineService
	scanWorkers       int
	scanBufferSize    int
}

// NewTaskHandler creates a new task handler
func NewTaskHandler(
	db *gorm.DB,
	directorySvc *directory.DirectoryCodeGenerator,
	pathResolver *directory.PathTemplateResolver,
	quarantineService *QuarantineService,
	scanWorkers int,
	scanBufferSize int,
) *TaskHandler {
	// Use defaults if not provided
	if scanWorkers <= 0 {
		scanWorkers = 8
	}
	if scanBufferSize <= 0 {
		scanBufferSize = 1000
	}

	return &TaskHandler{
		db:                db,
		directorySvc:      directorySvc,
		pathResolver:      pathResolver,
		quarantineService: quarantineService,
		scanWorkers:       scanWorkers,
		scanBufferSize:    scanBufferSize,
	}
}

// getScanConfigFromDB retrieves scan configuration from database settings
// Falls back to handler's default values if settings don't exist
func (h *TaskHandler) getScanConfigFromDB() (workers int, bufferSize int, maxFiles int) {
	workers = h.scanWorkers
	bufferSize = h.scanBufferSize
	maxFiles = 0 // Default to no limit

	// Try to get scan_workers from database
	var workersSetting struct {
		Value string `gorm:"column:value"`
	}
	if err := h.db.Table("settings").Select("value").Where("key = ?", "processing.scan_workers").First(&workersSetting).Error; err == nil {
		if w, parseErr := strconv.Atoi(workersSetting.Value); parseErr == nil && w > 0 && w <= 32 {
			workers = w
		}
	}

	// Try to get scan_buffer_size from database
	var bufferSetting struct {
		Value string `gorm:"column:value"`
	}
	if err := h.db.Table("settings").Select("value").Where("key = ?", "processing.scan_buffer_size").First(&bufferSetting).Error; err == nil {
		if b, parseErr := strconv.Atoi(bufferSetting.Value); parseErr == nil && b >= 100 && b <= 10000 {
			bufferSize = b
		}
	}

	// Try to get scan_max_files from database
	var maxFilesSetting struct {
		Value string `gorm:"column:value"`
	}
	if err := h.db.Table("settings").Select("value").Where("key = ?", "processing.scan_max_files").First(&maxFilesSetting).Error; err == nil {
		if m, parseErr := strconv.Atoi(maxFilesSetting.Value); parseErr == nil && m >= 0 {
			maxFiles = m
		}
	}

	return workers, bufferSize, maxFiles
}

// HandleLibraryScan processes library scan jobs
func (h *TaskHandler) HandleLibraryScan(ctx context.Context, t *asynq.Task) error {
	var p LibraryScanPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal library scan payload: %w", err)
	}

	log.Printf("Scanning libraries: %v, force: %v", p.LibraryIDs, p.Force)

	// Scan each library
	for _, libraryID := range p.LibraryIDs {
		if err := h.scanLibrary(ctx, libraryID, p.Force); err != nil {
			log.Printf("Error scanning library %d: %v", libraryID, err)
			// Continue with other libraries even if one fails
			continue
		}
	}

	log.Printf("Library scan completed for libraries: %v", p.LibraryIDs)

	return nil
}

// scanLibrary scans a single library and discovers media files
func (h *TaskHandler) scanLibrary(ctx context.Context, libraryID int32, force bool) error {
	// Get library from database
	var library struct {
		ID   int32  `gorm:"column:id"`
		Name string `gorm:"column:name"`
		Path string `gorm:"column:path"`
		Type string `gorm:"column:type"`
	}

	if err := h.db.Table("libraries").Where("id = ?", libraryID).First(&library).Error; err != nil {
		return fmt.Errorf("failed to get library: %w", err)
	}

	log.Printf("Scanning library: %s (type: %s, path: %s)", library.Name, library.Type, library.Path)

	// For inbound libraries, scan the directory and count files
	if library.Type == "inbound" {
		return h.scanInboundLibrary(ctx, library.ID, library.Path)
	}

	log.Printf("Library %s is not an inbound library, skipping scan", library.Name)
	return nil
}

// scanInboundLibrary scans an inbound library directory with concurrent processing
func (h *TaskHandler) scanInboundLibrary(ctx context.Context, libraryID int32, path string) error {
	log.Printf("Scanning inbound library directory: %s", path)

	// Check if directory exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("library path does not exist: %s", path)
	}

	// Get scan configuration from database (falls back to config defaults)
	scanWorkers, scanBufferSize, maxFiles := h.getScanConfigFromDB()
	if maxFiles > 0 {
		log.Printf("Using scan configuration: workers=%d, bufferSize=%d, maxFiles=%d (limited)", scanWorkers, scanBufferSize, maxFiles)
	} else {
		log.Printf("Using scan configuration: workers=%d, bufferSize=%d, maxFiles=unlimited", scanWorkers, scanBufferSize)
	}

	// Use atomic counters for thread-safe incrementing
	var fileCount atomic.Int32
	var totalSize atomic.Int64
	var shouldStop atomic.Bool

	// Channel to send file paths for processing
	fileChan := make(chan string, scanBufferSize)

	// WaitGroup to track worker goroutines
	var wg sync.WaitGroup

	// Start worker goroutines to process files concurrently
	for i := 0; i < scanWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for filePath := range fileChan {
				// Check if we should stop (max files reached)
				if shouldStop.Load() {
					continue
				}

				// Get file info
				info, err := os.Stat(filePath)
				if err != nil {
					log.Printf("Error stating file %s: %v", filePath, err)
					continue
				}

				// Check if it's a media file
				if isMediaFile(filePath) {
					newCount := fileCount.Add(1)
					totalSize.Add(info.Size())

					// Check if we've reached the max files limit
					if maxFiles > 0 && newCount >= int32(maxFiles) {
						shouldStop.Store(true)
						log.Printf("Reached max files limit (%d), stopping scan", maxFiles)
					}
				}
			}
		}()
	}

	// Walk the directory and send files to workers
	walkErr := filepath.WalkDir(path, func(filePath string, d fs.DirEntry, err error) error {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Check if we should stop due to max files limit
		if shouldStop.Load() {
			return filepath.SkipAll
		}

		if err != nil {
			log.Printf("Error accessing path %s: %v", filePath, err)
			return nil // Continue walking despite errors
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Send file path to workers
		select {
		case fileChan <- filePath:
		case <-ctx.Done():
			return ctx.Err()
		}

		return nil
	})

	// Close the channel and wait for workers to finish
	close(fileChan)
	wg.Wait()

	if walkErr != nil && walkErr != context.Canceled && walkErr != filepath.SkipAll {
		return fmt.Errorf("failed to walk directory: %w", walkErr)
	}

	finalCount := fileCount.Load()
	finalSize := totalSize.Load()

	if maxFiles > 0 && finalCount >= int32(maxFiles) {
		log.Printf("Scan stopped at limit: %d media files (total size: %d bytes) in library path: %s", finalCount, finalSize, path)
	} else {
		log.Printf("Found %d media files (total size: %d bytes) in library path: %s", finalCount, finalSize, path)
	}

	// Update library statistics in the database
	updateData := map[string]interface{}{
		"track_count": finalCount,
		// Note: For inbound libraries, we're counting files, not necessarily database records
		// This gives users visibility into how many files are available to process
	}

	if err := h.db.Table("libraries").Where("id = ?", libraryID).Updates(updateData).Error; err != nil {
		return fmt.Errorf("failed to update library statistics: %w", err)
	}

	log.Printf("Inbound library scan completed for path: %s - %d files found", path, finalCount)

	return nil
}

// isMediaFile checks if a file is a media file based on extension
func isMediaFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	mediaExts := []string{
		".mp3", ".flac", ".ogg", ".opus", ".m4a", ".mp4",
		".aac", ".wma", ".wav", ".aiff", ".ape", ".wv",
		".dsf", ".dff", ".cda", ".alac", ".tak",
	}

	for _, mediaExt := range mediaExts {
		if ext == mediaExt {
			return true
		}
	}

	return false
}

// HandleLibraryScan is the standalone function for backward compatibility
func HandleLibraryScan(ctx context.Context, t *asynq.Task) error {
	var p LibraryScanPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal library scan payload: %w", err)
	}

	log.Printf("WARNING: HandleLibraryScan called without dependencies - scan will not be performed")
	log.Printf("Libraries: %v, force: %v", p.LibraryIDs, p.Force)

	return nil
}

// LibraryProcessPayload represents the payload for library process jobs
type LibraryProcessPayload struct {
	LibraryID int32    `json:"library_id"`
	FilePaths []string `json:"file_paths"`
}

// HandleLibraryProcess processes inbound files to staging
func HandleLibraryProcess(ctx context.Context, t *asynq.Task) error {
	var p LibraryProcessPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal library process payload: %w", err)
	}

	log.Printf("Processing library ID: %d, with %d files", p.LibraryID, len(p.FilePaths))

	// In a real implementation, this would:
	// 1. Validate media files
	// 2. Extract metadata
	// 3. Organize files in staging area using directory codes
	// 4. Create staging records in DB

	log.Printf("Library process completed for library ID: %d", p.LibraryID)

	return nil
}

// LibraryMoveOKPayload represents the payload for promoting staging content to production
type LibraryMoveOKPayload struct {
	AlbumID int64 `json:"album_id"`
}

// HandleLibraryMoveOK moves OK-status albums from staging to production
func HandleLibraryMoveOK(ctx context.Context, t *asynq.Task) error {
	var p LibraryMoveOKPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal library move OK payload: %w", err)
	}

	log.Printf("Moving album ID: %d from staging to production", p.AlbumID)

	// In a real implementation, this would:
	// 1. Validate album is in OK status
	// 2. Select appropriate production library based on directory code
	// 3. Move files from staging to production location
	// 4. Update database records

	log.Printf("Album move completed for album ID: %d", p.AlbumID)

	return nil
}

// DirectoryRecalculatePayload represents the payload for directory code recalculation
type DirectoryRecalculatePayload struct {
	ArtistIDs []int64 `json:"artist_ids"`
}

// HandleDirectoryRecalculate recalculates directory codes
func HandleDirectoryRecalculate(ctx context.Context, t *asynq.Task) error {
	var p DirectoryRecalculatePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal directory recalculate payload: %w", err)
	}

	log.Printf("Recalculating directory codes for artists: %v", p.ArtistIDs)

	// In a real implementation, this would recalculate directory codes
	// for the specified artists and update references

	log.Printf("Directory recalculation completed for %d artists", len(p.ArtistIDs))

	return nil
}

// MetadataWritebackPayload represents the payload for metadata writeback
type MetadataWritebackPayload struct {
	TrackIDs []int64 `json:"track_ids"`
}

// HandleMetadataWriteback writes metadata changes back to files
func HandleMetadataWriteback(ctx context.Context, t *asynq.Task) error {
	var p MetadataWritebackPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal metadata writeback payload: %w", err)
	}

	log.Printf("Writing back metadata for %d tracks", len(p.TrackIDs))

	// In a real implementation, this would:
	// 1. Fetch updated metadata from DB
	// 2. Write it back to media files
	// 3. Handle conflicts according to metadata rules

	log.Printf("Metadata writeback completed for %d tracks", len(p.TrackIDs))

	return nil
}

// MetadataEnhancePayload represents the payload for metadata enhancement
type MetadataEnhancePayload struct {
	AlbumID int64    `json:"album_id"`
	Sources []string `json:"sources"`
}

// HandleMetadataEnhance enhances existing metadata with external sources
func HandleMetadataEnhance(ctx context.Context, t *asynq.Task) error {
	var p MetadataEnhancePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal metadata enhance payload: %w", err)
	}

	log.Printf("Enhancing metadata for album ID: %d with sources: %v", p.AlbumID, p.Sources)

	// In a real implementation, this would fetch metadata from external sources
	// like MusicBrainz, LastFM, etc., and update the database

	log.Printf("Metadata enhancement completed for album ID: %d", p.AlbumID)

	return nil
}

// MediaService handles media processing operations
type MediaService struct {
	db                *gorm.DB
	directorySvc      *directory.DirectoryCodeGenerator
	pathResolver      *directory.PathTemplateResolver
	QuarantineService *QuarantineService
}

// NewMediaService creates a new media service
func NewMediaService(
	db *gorm.DB,
	directorySvc *directory.DirectoryCodeGenerator,
	pathResolver *directory.PathTemplateResolver,
	quarantineSvc *QuarantineService,
) *MediaService {
	return &MediaService{
		db:                db,
		directorySvc:      directorySvc,
		pathResolver:      pathResolver,
		QuarantineService: quarantineSvc,
	}
}

// EnqueueLibraryScan creates and enqueues a library scan job
func (ms *MediaService) EnqueueLibraryScan(client *asynq.Client, libraryIDs []int32, force bool) error {
	payload, err := json.Marshal(LibraryScanPayload{
		LibraryIDs: libraryIDs,
		Force:      force,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal library scan payload: %w", err)
	}

	task := asynq.NewTask(TypeLibraryScan, payload)

	// Use deduplication key to prevent duplicate scans
	dedupKey := fmt.Sprintf("library.scan:%v", libraryIDs)

	_, err = client.Enqueue(task, asynq.TaskID(dedupKey), asynq.Timeout(5*time.Minute))
	if err != nil {
		return fmt.Errorf("failed to enqueue library scan: %w", err)
	}

	return nil
}

// EnqueueLibraryProcess creates and enqueues a library process job
func (ms *MediaService) EnqueueLibraryProcess(client *asynq.Client, libraryID int32, filePaths []string) error {
	payload, err := json.Marshal(LibraryProcessPayload{
		LibraryID: libraryID,
		FilePaths: filePaths,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal library process payload: %w", err)
	}

	task := asynq.NewTask(TypeLibraryProcess, payload)

	// Use deduplication key
	dedupKey := fmt.Sprintf("library.process:%d", libraryID)

	_, err = client.Enqueue(task, asynq.TaskID(dedupKey), asynq.Timeout(10*time.Minute))
	if err != nil {
		return fmt.Errorf("failed to enqueue library process: %w", err)
	}

	return nil
}

// EnqueueLibraryMoveOK creates and enqueues a library move OK job
func (ms *MediaService) EnqueueLibraryMoveOK(client *asynq.Client, albumID int64) error {
	payload, err := json.Marshal(LibraryMoveOKPayload{
		AlbumID: albumID,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal library move OK payload: %w", err)
	}

	task := asynq.NewTask(TypeLibraryMoveOK, payload)

	// Use deduplication key
	dedupKey := fmt.Sprintf("library.move_ok:%d", albumID)

	_, err = client.Enqueue(task, asynq.TaskID(dedupKey), asynq.Timeout(30*time.Minute))
	if err != nil {
		return fmt.Errorf("failed to enqueue library move OK: %w", err)
	}

	return nil
}

// EnqueueDirectoryRecalculate creates and enqueues a directory recalculate job
func (ms *MediaService) EnqueueDirectoryRecalculate(client *asynq.Client, artistIDs []int64) error {
	payload, err := json.Marshal(DirectoryRecalculatePayload{
		ArtistIDs: artistIDs,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal directory recalculate payload: %w", err)
	}

	task := asynq.NewTask(TypeDirectoryRecalculate, payload)

	// Use deduplication key
	dedupKey := fmt.Sprintf("directory.recalculate:%v", artistIDs)

	_, err = client.Enqueue(task, asynq.TaskID(dedupKey), asynq.Timeout(2*time.Minute))
	if err != nil {
		return fmt.Errorf("failed to enqueue directory recalculate: %w", err)
	}

	return nil
}

// EnqueueMetadataWriteback creates and enqueues a metadata writeback job
func (ms *MediaService) EnqueueMetadataWriteback(client *asynq.Client, trackIDs []int64) error {
	payload, err := json.Marshal(MetadataWritebackPayload{
		TrackIDs: trackIDs,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal metadata writeback payload: %w", err)
	}

	task := asynq.NewTask(TypeMetadataWriteback, payload)

	// Use deduplication key based on track IDs hash
	// In a real implementation, we'd create a hash of the track IDs
	dedupKey := fmt.Sprintf("metadata.writeback:%v", len(trackIDs)) // Simplified

	_, err = client.Enqueue(task, asynq.TaskID(dedupKey), asynq.Timeout(2*time.Minute))
	if err != nil {
		return fmt.Errorf("failed to enqueue metadata writeback: %w", err)
	}

	return nil
}

// EnqueueMetadataEnhance creates and enqueues a metadata enhance job
func (ms *MediaService) EnqueueMetadataEnhance(client *asynq.Client, albumID int64, sources []string) error {
	payload, err := json.Marshal(MetadataEnhancePayload{
		AlbumID: albumID,
		Sources: sources,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal metadata enhance payload: %w", err)
	}

	task := asynq.NewTask(TypeMetadataEnhance, payload)

	// Use deduplication key
	dedupKey := fmt.Sprintf("metadata.enhance:%d", albumID)

	_, err = client.Enqueue(task, asynq.TaskID(dedupKey), asynq.Timeout(3*time.Minute))
	if err != nil {
		return fmt.Errorf("failed to enqueue metadata enhance: %w", err)
	}

	return nil
}
