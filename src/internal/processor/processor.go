package processor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"melodee/internal/scanner"
)

// ProcessorConfig contains configuration for the processor
type ProcessorConfig struct {
	StagingRoot   string
	Workers       int
	RateLimit     int // files per second (0 = unlimited)
	DryRun        bool
}

// Processor handles moving files from inbound to staging
type Processor struct {
	config    *ProcessorConfig
	scanDB    *scanner.ScanDB
	semaphore chan struct{} // for rate limiting
}

// NewProcessor creates a new processor
func NewProcessor(config *ProcessorConfig, scanDB *scanner.ScanDB) *Processor {
	if config.Workers <= 0 {
		config.Workers = 4
	}

	var semaphore chan struct{}
	if config.RateLimit > 0 {
		semaphore = make(chan struct{}, config.RateLimit)
		// Start rate limiter goroutine
		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for range ticker.C {
				// Refill semaphore
				for i := 0; i < config.RateLimit; i++ {
					select {
					case semaphore <- struct{}{}:
					default:
					}
				}
			}
		}()
	}

	return &Processor{
		config:    config,
		scanDB:    scanDB,
		semaphore: semaphore,
	}
}

// ProcessResult contains the result of processing an album
type ProcessResult struct {
	AlbumGroupID  string
	StagingPath   string
	TrackCount    int
	TotalSize     int64
	MetadataFile  string
	Success       bool
	Error         error
}

// ProcessAllAlbums processes all album groups from the scan database
func (p *Processor) ProcessAllAlbums() ([]ProcessResult, error) {
	// Get all album groups
	groups, err := p.scanDB.GetAlbumGroups()
	if err != nil {
		return nil, fmt.Errorf("failed to get album groups: %w", err)
	}

	results := make([]ProcessResult, len(groups))
	var wg sync.WaitGroup
	resultsChan := make(chan ProcessResult, len(groups))
	albumsChan := make(chan scanner.AlbumGroup, len(groups))

	// Fill albums channel
	for _, group := range groups {
		albumsChan <- group
	}
	close(albumsChan)

	// Start workers
	for i := 0; i < p.config.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for group := range albumsChan {
				result := p.processAlbum(group)
				resultsChan <- result
			}
		}()
	}

	// Wait for completion
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	idx := 0
	for result := range resultsChan {
		results[idx] = result
		idx++
	}

	return results, nil
}

// processAlbum processes a single album group
func (p *Processor) processAlbum(group scanner.AlbumGroup) ProcessResult {
	result := ProcessResult{
		AlbumGroupID: group.AlbumGroupID,
		Success:      false,
	}

	// Get files for this album
	files, err := p.scanDB.GetFilesByAlbumGroup(group.AlbumGroupID)
	if err != nil {
		result.Error = fmt.Errorf("failed to get files: %w", err)
		return result
	}

	if len(files) == 0 {
		result.Error = fmt.Errorf("no files found for album group")
		return result
	}

	// Generate directory structure
	dirCode := GenerateDirectoryCode(group.ArtistName)
	albumDirName := fmt.Sprintf("%d - %s", group.Year, cleanDirectoryName(group.AlbumName))
	stagingPath := filepath.Join(
		p.config.StagingRoot,
		dirCode,
		cleanDirectoryName(group.ArtistName),
		albumDirName,
	)

	result.StagingPath = stagingPath
	result.TrackCount = len(files)

	// Create staging directory
	if !p.config.DryRun {
		if err := os.MkdirAll(stagingPath, 0755); err != nil {
			result.Error = fmt.Errorf("failed to create staging directory: %w", err)
			return result
		}
	}

	// Prepare metadata
	metadata := &AlbumMetadata{
		Version:     "1.0",
		ProcessedAt: time.Now(),
		ScanID:      p.scanDB.GetScanID(),
		Artist: ArtistMetadata{
			Name:           group.ArtistName,
			NameNormalized: NormalizeString(group.ArtistName),
			DirectoryCode:  dirCode,
		},
		Album: AlbumInfo{
			Name:           group.AlbumName,
			NameNormalized: NormalizeString(group.AlbumName),
			AlbumType:      "Album",
			Year:           group.Year,
			Genres:         []string{},
			IsCompilation:  false,
			ImageCount:     0,
		},
		Tracks: make([]TrackMetadata, 0, len(files)),
		Status: "pending_review",
		Validation: ValidationInfo{
			IsValid:  true,
			Errors:   []string{},
			Warnings: []string{},
		},
	}

	// Process each file
	var totalSize int64
	for _, file := range files {
		// Rate limiting
		if p.semaphore != nil {
			<-p.semaphore
		}

		// Determine destination filename
		ext := filepath.Ext(file.FilePath)
		newFilename := FormatFilename(file.DiscNumber, file.TrackNumber, file.Title, ext)
		dstPath := filepath.Join(stagingPath, newFilename)

		// Move file
		if !p.config.DryRun {
			if err := SafeMoveFile(file.FilePath, dstPath); err != nil {
				metadata.Validation.IsValid = false
				metadata.Validation.Errors = append(metadata.Validation.Errors,
					fmt.Sprintf("Failed to move %s: %v", filepath.Base(file.FilePath), err))
				continue
			}
		}

		// Calculate relative path
		relPath := filepath.Join(dirCode, cleanDirectoryName(group.ArtistName), albumDirName, newFilename)

		// Add to metadata
		metadata.Tracks = append(metadata.Tracks, TrackMetadata{
			TrackNumber:  file.TrackNumber,
			DiscNumber:   file.DiscNumber,
			Name:         file.Title,
			Duration:     file.Duration,
			FilePath:     relPath,
			FileSize:     file.FileSize,
			Bitrate:      file.Bitrate,
			SampleRate:   file.SampleRate,
			Checksum:     file.FileHash,
			OriginalPath: file.FilePath,
		})

		totalSize += file.FileSize
	}

	result.TotalSize = totalSize

	// Write metadata file
	metadataPath := filepath.Join(stagingPath, "album.melodee.json")
	result.MetadataFile = metadataPath

	if !p.config.DryRun {
		if err := WriteAlbumMetadata(metadataPath, metadata); err != nil {
			result.Error = fmt.Errorf("failed to write metadata: %w", err)
			return result
		}
	}

	// Calculate metadata checksum (for future use in database)
	_, err = calculateMetadataChecksum(metadata)
	if err != nil {
		result.Error = fmt.Errorf("failed to calculate checksum: %w", err)
		return result
	}

	// Mark as successful if validation passed
	result.Success = metadata.Validation.IsValid
	if !result.Success {
		result.Error = fmt.Errorf("validation failed: %d errors", len(metadata.Validation.Errors))
	}

	return result
}

// cleanDirectoryName removes or replaces characters that are problematic in directory names
func cleanDirectoryName(name string) string {
	// Replace problematic characters
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "",
		"?", "",
		"\"", "",
		"<", "",
		">", "",
		"|", "",
	)
	return strings.TrimSpace(replacer.Replace(name))
}

// calculateMetadataChecksum calculates SHA256 checksum of metadata
func calculateMetadataChecksum(metadata *AlbumMetadata) (string, error) {
	data, err := json.Marshal(metadata)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// ProcessStats contains statistics about processing
type ProcessStats struct {
	TotalAlbums    int
	SuccessAlbums  int
	FailedAlbums   int
	TotalTracks    int
	TotalSize      int64
	Duration       time.Duration
}

// GetProcessStats calculates statistics from process results
func GetProcessStats(results []ProcessResult, duration time.Duration) ProcessStats {
	stats := ProcessStats{
		TotalAlbums: len(results),
		Duration:    duration,
	}

	for _, result := range results {
		if result.Success {
			stats.SuccessAlbums++
		} else {
			stats.FailedAlbums++
		}
		stats.TotalTracks += result.TrackCount
		stats.TotalSize += result.TotalSize
	}

	return stats
}
