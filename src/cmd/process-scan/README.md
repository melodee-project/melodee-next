# Melodee Process Scan

## Overview

The `process-scan` CLI tool implements Phase 2 of the Media Workflow Refactor. It takes a scan database (created by `scan-inbound`) and processes it by:

1. Moving files from scattered inbound locations to organized staging directories
2. Creating JSON sidecar metadata files (`album.melodee.json`)
3. Optionally creating `staging_items` records in PostgreSQL

## Features

- **Organized Directory Structure**: `{DirectoryCode}/{ArtistName}/{Year} - {AlbumName}/`
- **Worker Pool**: Parallel processing with configurable workers
- **Rate Limiting**: Optional throttling of file operations
- **Dry Run Mode**: Preview changes without moving files
- **JSON Metadata**: Complete album/track metadata in sidecar files
- **Database Integration**: Optional PostgreSQL staging_items creation
- **Safe File Moving**: Copy+delete fallback if rename fails

## Usage

```bash
# Basic usage (files only, no database)
./process-scan -scan /tmp/scan_20241127_150405.db -staging /melodee/staging

# With database integration
./process-scan -scan /tmp/scan_20241127_150405.db \
  -staging /melodee/staging \
  -db-host localhost \
  -db-pass secret

# Dry run to preview
./process-scan -scan /tmp/scan_20241127_150405.db \
  -staging /melodee/staging \
  -dry-run

# With rate limiting and more workers
./process-scan -scan /tmp/scan_20241127_150405.db \
  -staging /melodee/staging \
  -workers 8 \
  -rate-limit 100
```

## Options

- `-scan string`: Path to scan database file (required)
- `-staging string`: Root directory for staging (default "/melodee/staging")
- `-workers int`: Number of worker goroutines (default 4)
- `-rate-limit int`: File operations per second, 0 = unlimited (default 0)
- `-dry-run`: Preview changes without moving files
- `-db-host string`: PostgreSQL host (optional)
- `-db-port int`: PostgreSQL port (default 5432)
- `-db-name string`: PostgreSQL database name (default "melodee")
- `-db-user string`: PostgreSQL user (default "melodee_user")
- `-db-pass string`: PostgreSQL password

## Directory Structure

The tool creates an organized directory structure:

```
staging/
├── LZ/                          # Directory code
│   └── Led Zeppelin/            # Artist name
│       └── 1971 - Led Zeppelin IV/  # Year - Album
│           ├── album.melodee.json   # Metadata file
│           ├── 01 - Black Dog.flac
│           ├── 02 - Rock and Roll.flac
│           └── ...
└── TB/
    └── The Beatles/
        └── 1969 - Abbey Road/
            ├── album.melodee.json
            ├── 01 - Come Together.flac
            └── ...
```

## Metadata File Format

`album.melodee.json`:

```json
{
  "version": "1.0",
  "processed_at": "2025-11-27T07:38:50Z",
  "scan_id": "scan_20251127_073845",
  "artist": {
    "name": "Led Zeppelin",
    "name_normalized": "led zeppelin",
    "directory_code": "LZ"
  },
  "album": {
    "name": "Led Zeppelin IV",
    "name_normalized": "led zeppelin iv",
    "album_type": "Album",
    "year": 1971,
    "genres": ["Rock"],
    "is_compilation": false
  },
  "tracks": [
    {
      "track_number": 1,
      "disc_number": 1,
      "name": "Black Dog",
      "duration": 295000,
      "file_path": "LZ/Led Zeppelin/1971 - Led Zeppelin IV/01 - Black Dog.flac",
      "file_size": 45678901,
      "bitrate": 1411,
      "sample_rate": 44100,
      "checksum": "abc123...",
      "original_path": "/inbound/scattered/BlackDog.flac"
    }
  ],
  "status": "pending_review",
  "validation": {
    "is_valid": true,
    "errors": [],
    "warnings": []
  }
}
```

## Example Output

```
Opening scan database: /tmp/scan_20251127_073845.db

=== Scan Database Info ===
Scan ID: scan_20251127_073845
Total files: 156
Valid files: 156
Albums found: 8

Processing albums to staging (/melodee/staging)...
Workers: 4

Saving staging items to database...
Saved 8 staging items to database

=== Processing Complete ===
Duration: 2.3s
Total albums: 8
Successful: 8
Failed: 0
Total tracks: 156
Total size: 1,234.56 MB

=== Staged Albums ===
1. /melodee/staging/LZ/Led Zeppelin/1971 - Led Zeppelin IV
   Tracks: 8, Size: 256.00 MB
2. /melodee/staging/TB/The Beatles/1969 - Abbey Road
   Tracks: 17, Size: 412.00 MB
...

Staging directory: /melodee/staging
```

## Database Integration

When database credentials are provided, the tool creates `staging_items` records:

```sql
INSERT INTO staging_items (
  scan_id, staging_path, metadata_file,
  artist_name, album_name, track_count, total_size,
  processed_at, status, checksum
) VALUES (...);
```

This enables:
- UI listing of staged albums
- Review workflow (approve/reject)
- Promotion to production tracking
- Audit trail

## Rate Limiting

Use `-rate-limit` to throttle file operations and avoid overwhelming storage:

```bash
# Limit to 50 files per second
./process-scan -scan scan.db -staging /staging -rate-limit 50
```

Useful for:
- Network-attached storage
- Busy production systems
- Large batch processing

## Next Steps

After processing:

1. **Review in UI**: Browse staged albums and their metadata
2. **Approve/Reject**: Mark albums for promotion or fix issues
3. **Promote**: Move approved albums from staging to production
4. **Archive Scan**: Clean up old scan databases (90-day retention)

## Building

```bash
cd /home/steven/source/melodee-next
go build -o process-scan ./src/cmd/process-scan/main.go
```

## Testing

```bash
# Create and scan test data
mkdir -p /tmp/test-inbound/Artist/Album
echo "test" > /tmp/test-inbound/Artist/Album/01-track.mp3
./scan-inbound -path /tmp/test-inbound -output /tmp

# Process with dry run
./process-scan -scan /tmp/scan_*.db -staging /tmp/staging -dry-run

# Process for real
./process-scan -scan /tmp/scan_*.db -staging /tmp/staging

# Verify structure
find /tmp/staging -type f
cat /tmp/staging/*/*/*/album.melodee.json
```
