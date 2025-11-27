package scanner

import "time"

// ScannedFile represents a file found during inbound scanning
type ScannedFile struct {
	ID          int64  `db:"id"`
	FilePath    string `db:"file_path"`
	FileSize    int64  `db:"file_size"`
	FileHash    string `db:"file_hash"`
	ModifiedTime int64  `db:"modified_time"`
	
	// Extracted metadata
	Artist      string `db:"artist"`
	AlbumArtist string `db:"album_artist"`
	Album       string `db:"album"`
	Title       string `db:"title"`
	TrackNumber int    `db:"track_number"`
	DiscNumber  int    `db:"disc_number"`
	Year        int    `db:"year"`
	Genre       string `db:"genre"`
	Duration    int    `db:"duration"`     // milliseconds
	Bitrate     int    `db:"bitrate"`      // kbps
	SampleRate  int    `db:"sample_rate"` // Hz
	
	// Validation
	IsValid         bool   `db:"is_valid"`
	ValidationError string `db:"validation_error"`
	
	// Grouping (computed after scan)
	AlbumGroupHash string `db:"album_group_hash"`
	AlbumGroupID   string `db:"album_group_id"`
	
	CreatedAt int64 `db:"created_at"`
}

// AlbumGroup represents a group of files identified as belonging to the same album
type AlbumGroup struct {
	AlbumGroupID string
	ArtistName   string
	AlbumName    string
	Year         int
	TrackCount   int
	TotalSize    int64
	FilePaths    []string
}

// ScanStats holds statistics about a scan operation
type ScanStats struct {
	TotalFiles      int
	ValidFiles      int
	InvalidFiles    int
	AlbumsFound     int
	StartTime       time.Time
	EndTime         time.Time
	Duration        time.Duration
	FilesPerSecond  float64
}
