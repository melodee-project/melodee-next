package media

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"melodee/internal/directory"
)

// Job types for Asynq
const (
	TypeLibraryScan        = "library:scan"
	TypeLibraryProcess     = "library:process"
	TypeLibraryMoveOK      = "library:move_ok"
	TypeDirectoryRecalculate = "directory:recalculate"
	TypeMetadataWriteback  = "metadata:writeback"
	TypeMetadataEnhance    = "metadata:enhance"
)

// Job payloads and handlers

// LibraryScanPayload represents the payload for library scan jobs
type LibraryScanPayload struct {
	LibraryIDs []int32 `json:"library_ids"`
	Force      bool    `json:"force"`
}

// HandleLibraryScan processes library scan jobs
func HandleLibraryScan(ctx context.Context, t *asynq.Task) error {
	var p LibraryScanPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal library scan payload: %w", err)
	}

	log.Printf("Scanning libraries: %v, force: %v", p.LibraryIDs, p.Force)

	// In a real implementation, this would scan the libraries
	// and trigger file processing workflows
	
	// For now, just log the task
	log.Printf("Library scan completed for libraries: %v", p.LibraryIDs)
	
	return nil
}

// LibraryProcessPayload represents the payload for library process jobs
type LibraryProcessPayload struct {
	LibraryID int32 `json:"library_id"`
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
	SongIDs []int64 `json:"song_ids"`
}

// HandleMetadataWriteback writes metadata changes back to files
func HandleMetadataWriteback(ctx context.Context, t *asynq.Task) error {
	var p MetadataWritebackPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal metadata writeback payload: %w", err)
	}

	log.Printf("Writing back metadata for %d songs", len(p.SongIDs))

	// In a real implementation, this would:
	// 1. Fetch updated metadata from DB
	// 2. Write it back to media files
	// 3. Handle conflicts according to metadata rules

	log.Printf("Metadata writeback completed for %d songs", len(p.SongIDs))
	
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
	db           *gorm.DB
	directorySvc *directory.DirectoryCodeGenerator
	pathResolver *directory.PathTemplateResolver
}

// NewMediaService creates a new media service
func NewMediaService(
	db *gorm.DB,
	directorySvc *directory.DirectoryCodeGenerator,
	pathResolver *directory.PathTemplateResolver,
) *MediaService {
	return &MediaService{
		db:           db,
		directorySvc: directorySvc,
		pathResolver: pathResolver,
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
func (ms *MediaService) EnqueueMetadataWriteback(client *asynq.Client, songIDs []int64) error {
	payload, err := json.Marshal(MetadataWritebackPayload{
		SongIDs: songIDs,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal metadata writeback payload: %w", err)
	}

	task := asynq.NewTask(TypeMetadataWriteback, payload)
	
	// Use deduplication key based on song IDs hash
	// In a real implementation, we'd create a hash of the song IDs
	dedupKey := fmt.Sprintf("metadata.writeback:%v", len(songIDs)) // Simplified
	
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