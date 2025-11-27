# Quick Start Guide - Media Workflow

## Overview

Melodee's new media workflow uses two CLI tools to process scattered media files:

1. **scan-inbound**: Catalogs files → SQLite database
2. **process-scan**: Organizes files → Staging directory

## Installation

```bash
# Build both tools
cd /home/steven/source/melodee-next
go build -o scan-inbound ./src/cmd/scan-inbound/main.go
go build -o process-scan ./src/cmd/process-scan/main.go

# Or use pre-built binaries
ls /tmp/scan-inbound /tmp/process-scan
```

## Basic Usage

### Step 1: Scan Inbound Directory

```bash
# Scan your inbound directory
./scan-inbound -path /path/to/inbound -output /var/melodee/scans

# Example output:
# Scan ID: scan_20251127_150405
# Database: /var/melodee/scans/scan_20251127_150405.db
# 
# === Scan Complete ===
# Total files: 1,234
# Valid files: 1,200
# Invalid files: 34
# Albums found: 45
```

### Step 2: Process to Staging

```bash
# Organize files into staging
./process-scan \
  -scan /var/melodee/scans/scan_20251127_150405.db \
  -staging /var/melodee/staging

# Example output:
# === Processing Complete ===
# Total albums: 45
# Successful: 44
# Failed: 1
# Total tracks: 1,200
# Total size: 8,765.43 MB
```

### Step 3: Review Results

```bash
# List organized albums
ls -R /var/melodee/staging

# View metadata for an album
cat /var/melodee/staging/LZ/Led\ Zeppelin/1971\ -\ Led\ Zeppelin\ IV/album.melodee.json
```

## Advanced Usage

### Dry Run (Preview)

```bash
# See what would happen without moving files
./process-scan -scan scan.db -staging /staging -dry-run
```

### Rate Limiting

```bash
# Limit to 50 files per second (for NAS/shared storage)
./process-scan -scan scan.db -staging /staging -rate-limit 50
```

### More Workers

```bash
# Use 8 parallel workers for faster processing
./scan-inbound -path /inbound -output /scans -workers 8
./process-scan -scan scan.db -staging /staging -workers 8
```

### Database Integration

```bash
# Save staging items to PostgreSQL
./process-scan \
  -scan scan.db \
  -staging /staging \
  -db-host localhost \
  -db-pass your_password
```

## Directory Structure

### Inbound (Before)

```
/inbound/
├── random_folder1/
│   ├── track01.flac
│   └── track02.flac
├── another_folder/
│   ├── track03.flac
│   └── track04.flac
└── misc/
    └── track05.flac
```

### Staging (After)

```
/staging/
├── LZ/                              # Directory code
│   └── Led Zeppelin/                # Artist
│       └── 1971 - Led Zeppelin IV/  # Year - Album
│           ├── album.melodee.json   # Metadata
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

## Common Workflows

### Production Workflow

```bash
# 1. Scan new inbound files (run daily/hourly)
./scan-inbound -path /inbound -output /scans -workers 8

# 2. Review scan results
sqlite3 /scans/scan_*.db "SELECT artist, album, COUNT(*) FROM scanned_files GROUP BY album_group_id"

# 3. Process to staging with database integration
./process-scan \
  -scan /scans/scan_YYYYMMDD_HHMMSS.db \
  -staging /staging \
  -workers 8 \
  -db-host localhost \
  -db-pass secret

# 4. Review in UI (Phase 3)
# Browse to http://localhost:3000/staging

# 5. Approve/reject albums (Phase 3)
# Use UI or SQL: UPDATE staging_items SET status='approved' WHERE id=123

# 6. Promote to production (Phase 3)
# Move approved albums from staging to production library
```

### Testing Workflow

```bash
# 1. Create test data
mkdir -p /tmp/test-inbound/artist/album
echo "test" > /tmp/test-inbound/artist/album/01-track.mp3
echo "test" > /tmp/test-inbound/artist/album/02-track.mp3

# 2. Scan
./scan-inbound -path /tmp/test-inbound -output /tmp

# 3. Preview processing
./process-scan -scan /tmp/scan_*.db -staging /tmp/staging -dry-run

# 4. Process for real
./process-scan -scan /tmp/scan_*.db -staging /tmp/staging

# 5. Verify
find /tmp/staging -type f
cat /tmp/staging/*/*/*/album.melodee.json

# 6. Cleanup
rm -rf /tmp/test-inbound /tmp/staging /tmp/scan_*.db
```

### Recovery Workflow

```bash
# If processing fails, you can re-run safely
./process-scan -scan scan.db -staging /staging

# Files already moved will be skipped
# Failed albums can be processed again
```

## Troubleshooting

### "No files found"

```bash
# Check what extensions are supported
grep -A10 "isMediaFile" src/internal/scanner/scanner.go

# Currently supported: .mp3, .flac, .m4a, .aac, .ogg, .opus, .wma, .wav, .ape, .wv
```

### "Permission denied"

```bash
# Ensure directories are writable
chmod 755 /var/melodee/scans /var/melodee/staging

# Check file permissions
ls -la /inbound
```

### "Database locked"

```bash
# Close any open SQLite connections
# Each process opens and closes the DB properly
# This shouldn't happen in normal operation
```

### "Invalid files"

```bash
# Check validation errors in scan database
sqlite3 scan.db "SELECT file_path, validation_error FROM scanned_files WHERE is_valid=0"
```

## Performance Tips

1. **Use SSDs**: Significant speed improvement for scanning
2. **More Workers**: 8-16 workers for large libraries
3. **Rate Limiting**: Use on network storage to avoid overload
4. **Batch Processing**: Process multiple scans in sequence

## File Safety

✅ **Safe Operations**
- Original files are moved (not copied)
- Atomic rename when possible
- Copy+delete fallback for cross-filesystem
- Dry-run mode for testing
- Original paths preserved in metadata

⚠️ **Important**
- Always test with dry-run first
- Backup important files before processing
- Keep scan databases for audit trail
- Monitor disk space during processing

## Next Steps

After organizing files to staging:

1. **Review** albums in staging directory
2. **Approve** good albums via UI (Phase 3)
3. **Reject** problematic albums for re-processing
4. **Promote** approved albums to production library
5. **Archive** old scan databases after 90 days

## Support

- **Documentation**: `docs/` directory
- **Phase 1 Details**: `docs/PHASE1_IMPLEMENTATION.md`
- **Phase 2 Details**: `docs/PHASE2_IMPLEMENTATION.md`
- **Tool Help**: `./scan-inbound --help` or `./process-scan --help`

## Quick Reference

| Task | Command |
|------|---------|
| Scan inbound | `./scan-inbound -path /inbound -output /scans` |
| Process to staging | `./process-scan -scan scan.db -staging /staging` |
| Dry run | `./process-scan -scan scan.db -staging /staging -dry-run` |
| With database | `./process-scan -scan scan.db -staging /staging -db-host localhost -db-pass secret` |
| Rate limited | `./process-scan -scan scan.db -staging /staging -rate-limit 50` |
| View scan stats | `sqlite3 scan.db "SELECT * FROM scanned_files LIMIT 10"` |
| View metadata | `cat /staging/*/*/*/album.melodee.json` |
