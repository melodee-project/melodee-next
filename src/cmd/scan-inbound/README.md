# Melodee Inbound Scanner

## Overview

The `scan-inbound` CLI tool implements Phase 1 of the Media Workflow Refactor. It scans a directory of media files and creates a SQLite database containing metadata and album groupings.

## Features

- **High-Performance Scanning**: Uses worker pools for parallel metadata extraction
- **SQLite Database**: Creates a standalone scan database for each scan operation
- **Two-Stage Album Grouping**: 
  - Stage 1: Normalizes artist + album names and creates hash
  - Stage 2: Refines groups using majority-voted year
- **Batch Inserts**: Inserts files in batches of 1000 for performance
- **Validation**: Tracks valid and invalid files separately

## Usage

```bash
# Basic usage
./scan-inbound -path /path/to/inbound

# With custom output directory and workers
./scan-inbound -path /path/to/inbound -output /var/melodee/scans -workers 8

# Full options
./scan-inbound --help
```

## Options

- `-path string`: Path to inbound directory to scan (required)
- `-output string`: Directory to store scan database (default "/tmp")
- `-workers int`: Number of worker goroutines (default 4)

## Output

The tool creates a SQLite database named `scan_YYYYMMDD_HHMMSS.db` containing:

- **scanned_files table**: All discovered media files with metadata
- **Album grouping**: Files grouped by artist + album + year

## Example Output

```
Creating scan database in /tmp...
Scan ID: scan_20251127_073221
Database: /tmp/scan_20251127_073221.db

Scanning /tmp/test-inbound with 2 workers...

Computing album grouping...

=== Scan Complete ===
Total files: 2
Valid files: 2
Invalid files: 0
Albums found: 1
Duration: 1.120844ms
Files/sec: 1784.37

=== Album Groups ===
1. Led Zeppelin - Led Zeppelin IV (1971)
   Tracks: 8, Size: 256.00 MB
```

## Database Schema

```sql
CREATE TABLE scanned_files (
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
    
    -- Grouping
    album_group_hash TEXT,
    album_group_id TEXT,
    
    created_at INTEGER
);
```

## Album Grouping Algorithm

### Stage 1: Normalization and Hashing

1. Normalize album name:
   - Convert to lowercase
   - Remove all whitespace
   - Remove remaster markers: "(Remaster)", "(Remastered)", etc.
   - Remove leading "the "
2. Create hash: `{artist}::{normalized_album}`

### Stage 2: Year Voting

1. For each hash group, find the most common year
2. Create final group ID: `{hash}_{year}`

This handles:
- Scattered files across multiple directories
- Inconsistent spacing in tags
- Remaster vs original releases
- Same album with different years

## Next Steps

After scanning, use the `process` command (Phase 2) to:
1. Move files from scattered inbound locations to organized staging
2. Create JSON sidecar metadata files
3. Create `staging_items` records in PostgreSQL

## Building

```bash
cd /home/steven/source/melodee-next
go build -o scan-inbound ./src/cmd/scan-inbound/main.go
```

## Testing

```bash
# Create test directory
mkdir -p /tmp/test-inbound/ArtistName/AlbumName
echo "test" > /tmp/test-inbound/ArtistName/AlbumName/01-track1.mp3

# Run scanner
./scan-inbound -path /tmp/test-inbound

# Query results
sqlite3 /tmp/scan_*.db "SELECT * FROM scanned_files;"
```
