package scanner

const ScanDatabaseSchema = `
CREATE TABLE IF NOT EXISTS scanned_files (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_path TEXT NOT NULL UNIQUE,
    file_size INTEGER,
    file_hash TEXT,
    modified_time INTEGER,
    
    -- Extracted metadata
    artist TEXT,
    album_artist TEXT,
    album TEXT,
    title TEXT,
    track_number INTEGER,
    disc_number INTEGER,
    year INTEGER,
    genre TEXT,
    duration INTEGER,
    bitrate INTEGER,
    sample_rate INTEGER,
    
    -- Validation
    is_valid BOOLEAN DEFAULT 1,
    validation_error TEXT,
    
    -- Grouping (computed later)
    album_group_hash TEXT,
    album_group_id TEXT,
    
    created_at INTEGER DEFAULT (strftime('%s','now'))
);

CREATE INDEX IF NOT EXISTS idx_artist_album ON scanned_files(artist, album, year);
CREATE INDEX IF NOT EXISTS idx_album_group_hash ON scanned_files(album_group_hash);
CREATE INDEX IF NOT EXISTS idx_album_group_id ON scanned_files(album_group_id);
CREATE INDEX IF NOT EXISTS idx_is_valid ON scanned_files(is_valid);
CREATE INDEX IF NOT EXISTS idx_file_hash ON scanned_files(file_hash);
`
