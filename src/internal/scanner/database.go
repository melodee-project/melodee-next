package scanner

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// ScanDB wraps a SQLite database for scanning operations
type ScanDB struct {
	db       *sql.DB
	path     string
	scanID   string
	startedAt time.Time
}

// NewScanDB creates a new scan database at the specified path
func NewScanDB(basePath string) (*ScanDB, error) {
	scanID := fmt.Sprintf("scan_%s", time.Now().Format("20060102_150405"))
	dbPath := filepath.Join(basePath, scanID+".db")
	
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_synchronous=NORMAL&_cache_size=10000")
	if err != nil {
		return nil, fmt.Errorf("failed to open scan database: %w", err)
	}
	
	// Create schema
	if _, err := db.Exec(ScanDatabaseSchema); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}
	
	return &ScanDB{
		db:        db,
		path:      dbPath,
		scanID:    scanID,
		startedAt: time.Now(),
	}, nil
}

// Close closes the database connection
func (s *ScanDB) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// GetPath returns the database file path
func (s *ScanDB) GetPath() string {
	return s.path
}

// GetScanID returns the scan identifier
func (s *ScanDB) GetScanID() string {
	return s.scanID
}

// InsertFile inserts a scanned file into the database
func (s *ScanDB) InsertFile(file *ScannedFile) error {
	query := `
		INSERT INTO scanned_files (
			file_path, file_size, file_hash, modified_time,
			artist, album_artist, album, title, track_number, disc_number, year, genre,
			duration, bitrate, sample_rate,
			is_valid, validation_error
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err := s.db.Exec(query,
		file.FilePath, file.FileSize, file.FileHash, file.ModifiedTime,
		file.Artist, file.AlbumArtist, file.Album, file.Title,
		file.TrackNumber, file.DiscNumber, file.Year, file.Genre,
		file.Duration, file.Bitrate, file.SampleRate,
		file.IsValid, file.ValidationError,
	)
	
	return err
}

// InsertBatch inserts multiple files in a single transaction
func (s *ScanDB) InsertBatch(files []*ScannedFile) error {
	if len(files) == 0 {
		return nil
	}
	
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	stmt, err := tx.Prepare(`
		INSERT INTO scanned_files (
			file_path, file_size, file_hash, modified_time,
			artist, album_artist, album, title, track_number, disc_number, year, genre,
			duration, bitrate, sample_rate,
			is_valid, validation_error
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	
	for _, file := range files {
		_, err := stmt.Exec(
			file.FilePath, file.FileSize, file.FileHash, file.ModifiedTime,
			file.Artist, file.AlbumArtist, file.Album, file.Title,
			file.TrackNumber, file.DiscNumber, file.Year, file.Genre,
			file.Duration, file.Bitrate, file.SampleRate,
			file.IsValid, file.ValidationError,
		)
		if err != nil {
			return err
		}
	}
	
	return tx.Commit()
}

// NormalizeAlbumName normalizes an album name for grouping
// Removes all whitespace, converts to lowercase, removes remaster markers
func NormalizeAlbumName(name string) string {
	// Convert to lowercase
	normalized := strings.ToLower(name)
	
	// Remove remaster indicators
	remasterPatterns := []string{
		"(remaster)", "(remastered)", "[remaster]", "[remastered]",
		"(deluxe edition)", "[deluxe edition]",
		"(expanded edition)", "[expanded edition]",
		"(anniversary edition)", "[anniversary edition]",
	}
	
	for _, pattern := range remasterPatterns {
		normalized = strings.ReplaceAll(normalized, pattern, "")
	}
	
	// Remove leading "the "
	normalized = strings.TrimPrefix(normalized, "the ")
	
	// Remove all whitespace
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.ReplaceAll(normalized, "\t", "")
	normalized = strings.TrimSpace(normalized)
	
	return normalized
}

// ComputeAlbumGrouping implements the two-stage album grouping algorithm
func (s *ScanDB) ComputeAlbumGrouping() error {
	// Stage 1: Compute album group hash
	// We need to do this row by row since SQLite doesn't support custom functions easily
	rows, err := s.db.Query(`
		SELECT id, album FROM scanned_files WHERE is_valid = 1 AND album_group_hash IS NULL
	`)
	if err != nil {
		return err
	}
	defer rows.Close()
	
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	updateStmt, err := tx.Prepare(`
		UPDATE scanned_files 
		SET album_group_hash = ?
		WHERE id = ?
	`)
	if err != nil {
		return err
	}
	defer updateStmt.Close()
	
	// Process each row
	for rows.Next() {
		var id int64
		var album string
		if err := rows.Scan(&id, &album); err != nil {
			return err
		}
		
		// Get the artist for this row
		var artist, albumArtist sql.NullString
		err := tx.QueryRow("SELECT artist, album_artist FROM scanned_files WHERE id = ?", id).
			Scan(&artist, &albumArtist)
		if err != nil {
			return err
		}
		
		artistName := artist.String
		if albumArtist.Valid && albumArtist.String != "" {
			artistName = albumArtist.String
		}
		
		normalizedAlbum := NormalizeAlbumName(album)
		hash := fmt.Sprintf("%s::%s", strings.ToLower(strings.TrimSpace(artistName)), normalizedAlbum)
		
		if _, err := updateStmt.Exec(hash, id); err != nil {
			return err
		}
	}
	
	if err := tx.Commit(); err != nil {
		return err
	}
	
	// Stage 2: Refine with year (majority vote)
	yearQuery := `
		WITH year_majorities AS (
			SELECT 
				album_group_hash,
				year,
				COUNT(*) as year_count,
				ROW_NUMBER() OVER (
					PARTITION BY album_group_hash 
					ORDER BY COUNT(*) DESC, year DESC
				) as rn
			FROM scanned_files
			WHERE is_valid = 1 AND album_group_hash IS NOT NULL
			GROUP BY album_group_hash, year
		)
		UPDATE scanned_files
		SET album_group_id = album_group_hash || '_' || CAST((
			SELECT year FROM year_majorities 
			WHERE year_majorities.album_group_hash = scanned_files.album_group_hash 
			AND rn = 1
		) AS TEXT)
		WHERE is_valid = 1 AND album_group_hash IS NOT NULL
	`
	
	_, err = s.db.Exec(yearQuery)
	return err
}

// GetAlbumGroups returns all identified album groups
func (s *ScanDB) GetAlbumGroups() ([]AlbumGroup, error) {
	query := `
		SELECT 
			album_group_id,
			COALESCE(album_artist, artist) as artist_name,
			album as album_name,
			year,
			COUNT(*) as track_count,
			SUM(file_size) as total_size
		FROM scanned_files
		WHERE is_valid = 1 AND album_group_id IS NOT NULL
		GROUP BY album_group_id
		ORDER BY artist_name, year, album_name
	`
	
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var groups []AlbumGroup
	for rows.Next() {
		var group AlbumGroup
		err := rows.Scan(
			&group.AlbumGroupID,
			&group.ArtistName,
			&group.AlbumName,
			&group.Year,
			&group.TrackCount,
			&group.TotalSize,
		)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	
	return groups, rows.Err()
}

// GetFilesByAlbumGroup returns all files for a specific album group
func (s *ScanDB) GetFilesByAlbumGroup(groupID string) ([]*ScannedFile, error) {
	query := `
		SELECT 
			id, file_path, file_size, file_hash, modified_time,
			artist, album_artist, album, title, track_number, disc_number, year, genre,
			duration, bitrate, sample_rate,
			is_valid, validation_error,
			album_group_hash, album_group_id, created_at
		FROM scanned_files
		WHERE album_group_id = ?
		ORDER BY disc_number, track_number
	`
	
	rows, err := s.db.Query(query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var files []*ScannedFile
	for rows.Next() {
		file := &ScannedFile{}
		err := rows.Scan(
			&file.ID, &file.FilePath, &file.FileSize, &file.FileHash, &file.ModifiedTime,
			&file.Artist, &file.AlbumArtist, &file.Album, &file.Title,
			&file.TrackNumber, &file.DiscNumber, &file.Year, &file.Genre,
			&file.Duration, &file.Bitrate, &file.SampleRate,
			&file.IsValid, &file.ValidationError,
			&file.AlbumGroupHash, &file.AlbumGroupID, &file.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	
	return files, rows.Err()
}

// GetStats returns statistics about the scan
func (s *ScanDB) GetStats() (*ScanStats, error) {
	stats := &ScanStats{
		StartTime: s.startedAt,
		EndTime:   time.Now(),
	}
	stats.Duration = stats.EndTime.Sub(stats.StartTime)
	
	err := s.db.QueryRow(`
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN is_valid = 1 THEN 1 ELSE 0 END) as valid,
			SUM(CASE WHEN is_valid = 0 THEN 1 ELSE 0 END) as invalid,
			COUNT(DISTINCT album_group_id) as albums
		FROM scanned_files
	`).Scan(&stats.TotalFiles, &stats.ValidFiles, &stats.InvalidFiles, &stats.AlbumsFound)
	
	if err != nil {
		return nil, err
	}
	
	if stats.Duration.Seconds() > 0 {
		stats.FilesPerSecond = float64(stats.TotalFiles) / stats.Duration.Seconds()
	}
	
	return stats, nil
}
