package scanner

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FileScanner scans a directory tree and extracts metadata from media files
type FileScanner struct {
	workers int
	scanDB  *ScanDB
}

// NewFileScanner creates a new file scanner
func NewFileScanner(scanDB *ScanDB, workers int) *FileScanner {
	if workers <= 0 {
		workers = 4 // Default to 4 workers
	}
	return &FileScanner{
		workers: workers,
		scanDB:  scanDB,
	}
}

// ScanDirectory scans a directory and all subdirectories for media files
func (fs *FileScanner) ScanDirectory(rootPath string) error {
	// Channel for file paths to process
	filePaths := make(chan string, 1000)
	
	// Channel for scanned files
	scannedFiles := make(chan *ScannedFile, 1000)
	
	// Channel for errors
	errChan := make(chan error, fs.workers)
	
	var wg sync.WaitGroup
	
	// Start workers
	for i := 0; i < fs.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for filePath := range filePaths {
				file, err := fs.scanFile(filePath)
				if err != nil {
					// Log error but continue
					fmt.Printf("Error scanning %s: %v\n", filePath, err)
					continue
				}
				if file != nil {
					scannedFiles <- file
				}
			}
		}()
	}
	
	// Start batch inserter
	insertDone := make(chan struct{})
	go func() {
		defer close(insertDone)
		batch := make([]*ScannedFile, 0, 1000)
		
		for file := range scannedFiles {
			batch = append(batch, file)
			
			if len(batch) >= 1000 {
				if err := fs.scanDB.InsertBatch(batch); err != nil {
					errChan <- err
					return
				}
				batch = batch[:0]
			}
		}
		
		// Insert remaining files
		if len(batch) > 0 {
			if err := fs.scanDB.InsertBatch(batch); err != nil {
				errChan <- err
			}
		}
	}()
	
	// Walk directory tree
	walkErr := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if info.IsDir() {
			return nil
		}
		
		// Check if it's a media file
		if isMediaFile(path) {
			filePaths <- path
		}
		
		return nil
	})
	
	close(filePaths)
	wg.Wait()
	close(scannedFiles)
	<-insertDone
	
	// Check for errors
	select {
	case err := <-errChan:
		return err
	default:
	}
	
	return walkErr
}

// scanFile extracts metadata from a single file
func (fs *FileScanner) scanFile(filePath string) (*ScannedFile, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	
	file := &ScannedFile{
		FilePath:     filePath,
		FileSize:     info.Size(),
		ModifiedTime: info.ModTime().Unix(),
		IsValid:      true,
	}
	
	// Calculate file hash
	hash, err := calculateFileHash(filePath)
	if err != nil {
		file.IsValid = false
		file.ValidationError = fmt.Sprintf("hash calculation failed: %v", err)
		return file, nil
	}
	file.FileHash = hash
	
	// Extract metadata (basic implementation - can be enhanced with taglib/ffprobe)
	metadata, err := extractBasicMetadata(filePath)
	if err != nil {
		file.IsValid = false
		file.ValidationError = fmt.Sprintf("metadata extraction failed: %v", err)
		return file, nil
	}
	
	// Populate metadata fields
	file.Artist = metadata.Artist
	file.AlbumArtist = metadata.AlbumArtist
	file.Album = metadata.Album
	file.Title = metadata.Title
	file.TrackNumber = metadata.TrackNumber
	file.DiscNumber = metadata.DiscNumber
	file.Year = metadata.Year
	file.Genre = metadata.Genre
	file.Duration = metadata.Duration
	file.Bitrate = metadata.Bitrate
	file.SampleRate = metadata.SampleRate
	
	// Validate required fields
	if file.Artist == "" || file.Album == "" || file.Title == "" {
		file.IsValid = false
		file.ValidationError = "missing required metadata (artist, album, or title)"
	}
	
	return file, nil
}

// Metadata holds basic metadata extracted from a file
type Metadata struct {
	Artist      string
	AlbumArtist string
	Album       string
	Title       string
	TrackNumber int
	DiscNumber  int
	Year        int
	Genre       string
	Duration    int // milliseconds
	Bitrate     int // kbps
	SampleRate  int // Hz
}

// extractBasicMetadata extracts metadata from a file
// This is a placeholder implementation that parses filenames
// TODO: Replace with proper tag reading (taglib, ffprobe)
func extractBasicMetadata(filePath string) (*Metadata, error) {
	fileName := filepath.Base(filePath)
	dirName := filepath.Base(filepath.Dir(filePath))
	
	metadata := &Metadata{
		DiscNumber: 1,
	}
	
	// Try to parse filename: "01 - Title.mp3" or "Track 01 - Title.flac"
	fileName = strings.TrimSuffix(fileName, filepath.Ext(fileName))
	parts := strings.SplitN(fileName, " - ", 2)
	
	if len(parts) == 2 {
		// Try to parse track number
		trackStr := strings.TrimSpace(parts[0])
		trackStr = strings.TrimPrefix(trackStr, "Track ")
		trackStr = strings.TrimPrefix(trackStr, "track ")
		var track int
		fmt.Sscanf(trackStr, "%d", &track)
		metadata.TrackNumber = track
		metadata.Title = strings.TrimSpace(parts[1])
	} else {
		metadata.Title = fileName
	}
	
	// Try to parse directory name for artist and album
	// Common patterns: "Artist - Album", "Artist/Album", "Year - Album"
	dirParts := strings.SplitN(dirName, " - ", 2)
	if len(dirParts) == 2 {
		metadata.Artist = strings.TrimSpace(dirParts[0])
		metadata.Album = strings.TrimSpace(dirParts[1])
		
		// Check if artist is actually a year
		var year int
		n, _ := fmt.Sscanf(metadata.Artist, "%d", &year)
		if n == 1 && year > 1900 && year < 2100 {
			metadata.Year = year
			metadata.Artist = "" // Will be set from parent directory
			
			// Try to get artist from parent directory
			parentDir := filepath.Base(filepath.Dir(filepath.Dir(filePath)))
			metadata.Artist = parentDir
		}
	} else {
		metadata.Album = dirName
	}
	
	// Set default values for testing
	if metadata.Artist == "" {
		metadata.Artist = "Unknown Artist"
	}
	if metadata.Album == "" {
		metadata.Album = "Unknown Album"
	}
	
	metadata.Duration = 180000    // 3 minutes default
	metadata.Bitrate = 320        // 320 kbps default
	metadata.SampleRate = 44100   // 44.1 kHz default
	
	return metadata, nil
}

// calculateFileHash calculates SHA256 hash of a file
func calculateFileHash(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	
	return hex.EncodeToString(h.Sum(nil)), nil
}

// isMediaFile checks if a file is a supported media file
func isMediaFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	mediaExtensions := map[string]bool{
		".mp3":  true,
		".flac": true,
		".m4a":  true,
		".aac":  true,
		".ogg":  true,
		".opus": true,
		".wma":  true,
		".wav":  true,
		".ape":  true,
		".wv":   true,
	}
	return mediaExtensions[ext]
}
