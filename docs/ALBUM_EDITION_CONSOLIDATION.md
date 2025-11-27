# Album Edition Consolidation (Release Groups)

**Status**: V2 Feature - Future Enhancement  
**Date**: November 26, 2025  
**Priority**: Medium - Quality of Life Improvement  
**Prerequisite**: Core workflow (V1) must be stable and working

## Executive Summary

Implement MusicBrainz-style **Release Groups** to consolidate multiple editions of the same album into a single virtual album. This solves the long-standing problem of album duplicates consuming disk space when an artist releases Original, Deluxe, Remaster, Anniversary, and other editions of the same album.

**Core Concept**: 
- **Release Group** = The conceptual album (what the artist talks about: "Highway 101")
- **Release** = Physical/digital edition you can buy (Original 1980, Deluxe 1980, 20th Anniversary 2000)
- **Track** = Individual song (deduplicated across releases)

**User Benefit**: See one "Highway 101" entry in library with 20 unique tracks, but maintain all 3 editions (Original, Deluxe, Anniversary) for completeness and historical preservation.

## Problem Statement

### Current Reality

When an artist releases multiple editions:
1. **Highway 101** (1980) - 10 tracks
2. **Highway 101 (Deluxe Edition)** (1980) - 12 tracks (same 10 + 2 new)
3. **Highway 101 (20th Anniversary)** (2000) - 20 tracks (12 from Deluxe + 8 bonus on Disc 2)

**Current behavior**: User sees 3 separate albums in library
- Album clutter (3 entries instead of 1)
- Disk space waste (10 duplicate track files)
- Organizational nightmare (which version to play?)
- No way to know they're the same album

### Real-World Impact

- **Disk Space**: 30-50% of music collection is duplicates across editions
- **Library Bloat**: 1000 unique albums becomes 1500+ entries with editions
- **User Confusion**: "Do I have this album or not?"
- **Quality Questions**: "Which version is the remaster?"

### What Users Want

1. **Single library entry** - "Highway 101" appears once
2. **Complete collection** - See all 20 unique tracks
3. **Edition awareness** - Know which tracks are bonus/live/remix
4. **Playback flexibility** - Choose to hear "Original 1980" or "Complete Collection"
5. **Smart cleanup** - Optionally delete duplicate files to save space

## MusicBrainz Release Group Model

### Core Entities

```
Release Group (Virtual Album)
    ├── Release 1: Original (1980)
    │   ├── Disc 1: Track 1-10
    │   └── Files: /staging/Artist/Album_Original_1980/
    │
    ├── Release 2: Deluxe Edition (1980)
    │   ├── Disc 1: Track 1-12 (10 duplicates + 2 new)
    │   └── Files: /staging/Artist/Album_Deluxe_1980/
    │
    └── Release 3: 20th Anniversary (2000)
        ├── Disc 1: Track 1-12 (duplicates from Deluxe)
        ├── Disc 2: Track 1-8 (new bonus tracks)
        └── Files: /staging/Artist/Album_Anniversary_2000/
```

### Key Principles

1. **Release Group is user-facing** - What shows up in library browse
2. **Releases are hidden by default** - Expand to see editions
3. **Tracks are deduplicated** - Same song = same database entry
4. **Files are preserved** - Keep all editions until user chooses cleanup
5. **Quality matters** - Best version of each track becomes default playback

## Database Schema

### Release Groups (Virtual Albums)

```sql
CREATE TABLE release_groups (
    id BIGSERIAL PRIMARY KEY,
    artist_id BIGINT NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    
    -- Core identity
    name TEXT NOT NULL,                    -- "Highway 101"
    name_normalized TEXT NOT NULL,         -- "highway101" for matching
    sort_name TEXT,                        -- For alphabetization
    
    -- Metadata from primary release
    type VARCHAR(50) NOT NULL DEFAULT 'album',  -- album, single, ep, compilation, soundtrack
    year INTEGER,                          -- Original release year
    
    -- Aggregated stats
    total_releases INTEGER DEFAULT 0,      -- Number of editions (3 in example)
    total_unique_tracks INTEGER DEFAULT 0, -- Deduplicated track count (20 in example)
    total_disc_count INTEGER DEFAULT 1,    -- Max discs across all releases
    
    -- Primary release (user's preferred edition)
    primary_release_id BIGINT,             -- FK to releases table (set after creation)
    
    -- MusicBrainz integration
    musicbrainz_id UUID,                   -- MBID for release group
    
    -- Cover art (from primary release)
    cover_art_path TEXT,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(artist_id, name_normalized)
);

CREATE INDEX idx_release_groups_artist ON release_groups(artist_id);
CREATE INDEX idx_release_groups_type ON release_groups(type);
CREATE INDEX idx_release_groups_year ON release_groups(year);
CREATE INDEX idx_release_groups_mbid ON release_groups(musicbrainz_id);
```

### Releases (Physical/Digital Editions)

```sql
CREATE TABLE releases (
    id BIGSERIAL PRIMARY KEY,
    release_group_id BIGINT NOT NULL REFERENCES release_groups(id) ON DELETE CASCADE,
    
    -- Edition identity
    name TEXT NOT NULL,                    -- "Highway 101 (Deluxe Edition)"
    edition_type VARCHAR(50),              -- "original", "deluxe", "remaster", "anniversary", "live", "compilation"
    
    -- Release-specific metadata
    release_date DATE,                     -- Can differ from release_group year
    release_year INTEGER,
    country_code CHAR(2),                  -- "US", "JP", "UK", etc.
    label TEXT,                            -- Record label
    catalog_number TEXT,                   -- Label catalog number
    barcode TEXT,                          -- UPC/EAN barcode
    
    -- Physical properties
    format VARCHAR(50),                    -- "CD", "Digital", "Vinyl", "Cassette"
    disc_count INTEGER DEFAULT 1,
    track_count INTEGER DEFAULT 0,         -- Total tracks in this release
    
    -- Quality/preference
    is_primary BOOLEAN DEFAULT FALSE,      -- User's preferred edition for this release group
    quality_score INTEGER DEFAULT 0,       -- Auto-calculated: bitrate, format, completeness
    
    -- File storage
    staging_path TEXT NOT NULL,            -- Directory: /staging/Artist/Album_Edition_Year/
    metadata_file TEXT,                    -- .melodee.json sidecar
    total_file_size BIGINT DEFAULT 0,      -- Bytes
    
    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'pending_review',  -- Same as staging_items
    reviewed_by BIGINT REFERENCES users(id),
    reviewed_at TIMESTAMP,
    
    -- MusicBrainz integration
    musicbrainz_id UUID,                   -- MBID for release
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(release_group_id, name, release_year)
);

CREATE INDEX idx_releases_group ON releases(release_group_id);
CREATE INDEX idx_releases_primary ON releases(is_primary) WHERE is_primary = TRUE;
CREATE INDEX idx_releases_status ON releases(status);
CREATE INDEX idx_releases_mbid ON releases(musicbrainz_id);
```

### Tracks (Deduplicated)

```sql
-- Modify existing tracks table to support multiple releases
CREATE TABLE tracks (
    id BIGSERIAL PRIMARY KEY,
    
    -- Belongs to release group (not album anymore)
    release_group_id BIGINT NOT NULL REFERENCES release_groups(id) ON DELETE CASCADE,
    artist_id BIGINT NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    
    -- Track identity
    title TEXT NOT NULL,
    title_normalized TEXT NOT NULL,        -- For fuzzy matching
    isrc TEXT,                             -- International Standard Recording Code
    
    -- Audio fingerprint for deduplication
    acoustic_fingerprint TEXT,             -- Chromaprint/AcoustID hash
    fingerprint_version INTEGER DEFAULT 1, -- Algorithm version
    
    -- Best version metadata (points to highest quality file)
    duration INTEGER,                      -- Milliseconds
    primary_file_path TEXT NOT NULL,       -- Best quality file
    primary_file_format VARCHAR(10),       -- "FLAC", "MP3", "M4A"
    primary_bitrate INTEGER,               -- kbps
    primary_sample_rate INTEGER,           -- Hz (44100, 48000, 96000)
    primary_release_id BIGINT,             -- Which release has the best file
    
    -- Track numbering (from primary release)
    disc_number INTEGER DEFAULT 1,
    track_number INTEGER,
    
    -- Metadata
    genre TEXT,
    composer TEXT,
    lyrics TEXT,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tracks_release_group ON tracks(release_group_id);
CREATE INDEX idx_tracks_artist ON tracks(artist_id);
CREATE INDEX idx_tracks_title_normalized ON tracks(title_normalized);
CREATE INDEX idx_tracks_fingerprint ON tracks(acoustic_fingerprint);
CREATE INDEX idx_tracks_isrc ON tracks(isrc);
```

### Release Tracks (Join Table with File Locations)

```sql
-- Maps which tracks appear in which releases, with edition-specific file paths
CREATE TABLE release_tracks (
    id BIGSERIAL PRIMARY KEY,
    release_id BIGINT NOT NULL REFERENCES releases(id) ON DELETE CASCADE,
    track_id BIGINT NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    
    -- Position in this release
    disc_number INTEGER DEFAULT 1,
    track_number INTEGER NOT NULL,
    
    -- File location for this edition
    file_path TEXT NOT NULL,               -- Edition-specific file
    file_format VARCHAR(10),               -- "FLAC", "MP3", "M4A"
    file_size BIGINT,                      -- Bytes
    file_hash TEXT,                        -- SHA256 for deduplication
    
    -- Audio quality for this version
    duration INTEGER,                      -- Milliseconds
    bitrate INTEGER,                       -- kbps
    sample_rate INTEGER,                   -- Hz
    channels INTEGER,                      -- 1=mono, 2=stereo, 6=5.1
    bit_depth INTEGER,                     -- 16, 24, 32
    
    -- Deduplication metadata
    is_primary_version BOOLEAN DEFAULT FALSE,  -- TRUE if this is the best quality version
    quality_score INTEGER DEFAULT 0,       -- Calculated score for ranking
    
    -- Track-specific metadata (can differ by release)
    title_variant TEXT,                    -- If title differs from canonical (rare)
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(release_id, disc_number, track_number),
    UNIQUE(release_id, file_path)
);

CREATE INDEX idx_release_tracks_release ON release_tracks(release_id);
CREATE INDEX idx_release_tracks_track ON release_tracks(track_id);
CREATE INDEX idx_release_tracks_file_hash ON release_tracks(file_hash);
CREATE INDEX idx_release_tracks_primary ON release_tracks(is_primary_version) WHERE is_primary_version = TRUE;
```

## Quality Scoring Algorithm

Automatically determine the "best" version of each track:

```sql
-- Calculate quality score (higher = better)
UPDATE release_tracks
SET quality_score = 
    -- Format preference (FLAC > ALAC > AAC > MP3)
    CASE file_format
        WHEN 'FLAC' THEN 1000
        WHEN 'ALAC' THEN 900
        WHEN 'AAC'  THEN 700
        WHEN 'M4A'  THEN 700
        WHEN 'MP3'  THEN 500
        ELSE 100
    END
    +
    -- Bitrate bonus (capped at 320 for lossy, unlimited for lossless)
    CASE 
        WHEN file_format IN ('FLAC', 'ALAC') THEN LEAST(bitrate / 100, 100)
        ELSE LEAST(bitrate, 320) / 10
    END
    +
    -- Sample rate bonus
    CASE
        WHEN sample_rate >= 96000 THEN 50
        WHEN sample_rate >= 48000 THEN 30
        WHEN sample_rate >= 44100 THEN 20
        ELSE 0
    END
    +
    -- Bit depth bonus (lossless only)
    CASE
        WHEN file_format IN ('FLAC', 'ALAC') AND bit_depth >= 24 THEN 25
        ELSE 0
    END
;

-- Mark primary version for each song (highest quality)
WITH ranked_versions AS (
    SELECT 
        id,
        track_id,
        quality_score,
        ROW_NUMBER() OVER (PARTITION BY track_id ORDER BY quality_score DESC, created_at ASC) as rank
    FROM release_tracks
)
UPDATE release_tracks rt
SET is_primary_version = (rv.rank = 1)
FROM ranked_versions rv
WHERE rt.id = rv.id;

-- Update tracks table to point to primary version
UPDATE tracks t
SET 
    primary_file_path = rt.file_path,
    primary_file_format = rt.file_format,
    primary_bitrate = rt.bitrate,
    primary_sample_rate = rt.sample_rate,
    primary_release_id = rt.release_id,
    duration = rt.duration
FROM release_tracks rt
WHERE t.id = rt.track_id 
  AND rt.is_primary_version = TRUE;
```

## Workflow Integration

### V1 Foundation (Current Workflow)

```
Scan Inbound → SQLite DB → Process to Staging → Review → Approve → Production
```

**V1 Result**: Each album edition is separate:
- `albums` table has 3 entries (Original, Deluxe, Anniversary)
- `tracks` table has 30 entries (10+10+10 duplicates)
- Files in `/production/Artist/Album_Edition/`

### V2 Enhancement (Release Group Consolidation)

**New Step**: After V1 approval, before moving to production:

```
Staging → Edition Detection → Consolidation Prompt → Release Group Creation → Production
```

#### Step 1: Detect Potential Release Groups

When user approves a staging item, check for existing release groups:

```sql
-- Find potential matches using fuzzy album name matching
SELECT 
    rg.id,
    rg.name,
    rg.total_releases,
    similarity(rg.name_normalized, normalize_album_name(?)) as match_score
FROM release_groups rg
WHERE rg.artist_id = ?
  AND similarity(rg.name_normalized, normalize_album_name(?)) > 0.85
ORDER BY match_score DESC
LIMIT 5;
```

**Normalization for Matching**:
```go
func normalizeAlbumName(name string) string {
    // Remove edition markers
    patterns := []string{
        `(?i)\(deluxe.*?\)`,
        `(?i)\(remaster.*?\)`,
        `(?i)\(anniversary.*?\)`,
        `(?i)\(expanded.*?\)`,
        `(?i)\(bonus.*?\)`,
        `(?i)\(special.*?\)`,
        `(?i)\[.*?\]`,  // Remove all bracketed text
        `(?i)\s+edition$`,
    }
    
    normalized := name
    for _, pattern := range patterns {
        re := regexp.MustCompile(pattern)
        normalized = re.ReplaceAllString(normalized, "")
    }
    
    // Lowercase, trim, remove extra whitespace
    normalized = strings.ToLower(strings.TrimSpace(normalized))
    normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, "")
    
    return normalized
}
```

#### Step 2: Present Consolidation UI

When match found, show dialog:

```
╔══════════════════════════════════════════════════════════════╗
║ Potential Album Edition Detected                            ║
╠══════════════════════════════════════════════════════════════╣
║                                                              ║
║ This album may be an edition of an existing release:        ║
║                                                              ║
║ NEW ALBUM:                                                   ║
║   • Highway 101 (Deluxe Edition)                            ║
║   • 12 tracks                                                ║
║   • 1980                                                     ║
║                                                              ║
║ EXISTING RELEASE GROUP:                                      ║
║   • Highway 101                                              ║
║   • 1 release: Original (1980) - 10 tracks                  ║
║                                                              ║
║ What would you like to do?                                   ║
║                                                              ║
║ [✓] Add as new release to existing release group            ║
║     → Consolidate into single library entry                 ║
║     → Deduplicate tracks                                     ║
║     → Keep both editions                                     ║
║                                                              ║
║ [ ] Create separate album                                    ║
║     → Treat as different album (user override)              ║
║                                                              ║
║          [Cancel]  [Continue]                                ║
╚══════════════════════════════════════════════════════════════╝
```

#### Step 3: Consolidation Process

If user chooses "Add as new release":

```go
func ConsolidateIntoReleaseGroup(releaseGroupID int64, stagingItem StagingItem) error {
    // 1. Create release record
    release := Release{
        ReleaseGroupID: releaseGroupID,
        Name:          stagingItem.AlbumName,
        EditionType:   detectEditionType(stagingItem.AlbumName),
        ReleaseYear:   stagingItem.Year,
        StagingPath:   stagingItem.StagingPath,
        TrackCount:    stagingItem.SongCount,
    }
    db.Create(&release)
    
    // 2. For each track in new release, find or create song
    for _, newTrack := range stagingItem.Tracks {
        // Generate acoustic fingerprint
        fingerprint := generateAcousticFingerprint(newTrack.FilePath)
        
        // Try to find existing song by fingerprint
        var existingSong Song
        err := db.Where("release_group_id = ? AND acoustic_fingerprint = ?", 
                       releaseGroupID, fingerprint).First(&existingSong).Error
        
        var trackID int64
        
        if err == nil {
            // Found duplicate - use existing song
            trackID = existingSong.ID
            
            // Update primary file if new version is better quality
            newQuality := calculateQualityScore(newTrack)
            if newQuality > existingSong.QualityScore {
                updatePrimaryVersion(trackID, newTrack, release.ID)
            }
        } else {
            // New unique track - create song
            song := Song{
                ReleaseGroupID:      releaseGroupID,
                Title:              newTrack.Title,
                TitleNormalized:    normalizeTitle(newTrack.Title),
                AcousticFingerprint: fingerprint,
                PrimaryFilePath:    newTrack.FilePath,
                PrimaryFileFormat:  newTrack.Format,
                Duration:           newTrack.Duration,
                PrimaryReleaseID:   release.ID,
            }
            db.Create(&song)
            trackID = song.ID
        }
        
        // Create release_track entry (tracks file location for this edition)
        releaseTrack := ReleaseTrack{
            ReleaseID:    release.ID,
            TrackID:      trackID,
            DiscNumber:   newTrack.DiscNumber,
            TrackNumber:  newTrack.TrackNumber,
            FilePath:     newTrack.FilePath,
            FileFormat:   newTrack.Format,
            FileSize:     newTrack.FileSize,
            FileHash:     newTrack.SHA256,
            Duration:     newTrack.Duration,
            Bitrate:      newTrack.Bitrate,
            SampleRate:   newTrack.SampleRate,
            QualityScore: calculateQualityScore(newTrack),
        }
        db.Create(&releaseTrack)
    }
    
    // 3. Recalculate release group stats
    updateReleaseGroupStats(releaseGroupID)
    
    // 4. Recalculate primary versions
    recalculatePrimaryVersions(releaseGroupID)
    
    return nil
}
```

#### Step 4: Acoustic Fingerprinting

Use Chromaprint for track deduplication:

```go
import "github.com/go-musicfox/go-musicfox/pkg/chromaprint"

func generateAcousticFingerprint(filePath string) string {
    // Extract audio fingerprint using Chromaprint/AcoustID
    fingerprint, err := chromaprint.Calculate(filePath)
    if err != nil {
        // Fallback to metadata-based matching
        return ""
    }
    
    // Return compact hash representation
    return fingerprint.Hash()
}

func fingerprintsMatch(fp1, fp2 string, threshold float64) bool {
    // Calculate similarity score (0.0 to 1.0)
    similarity := chromaprint.Compare(fp1, fp2)
    return similarity >= threshold  // e.g., 0.95 = 95% match
}
```

**Fallback Matching** (if fingerprinting fails):

```sql
-- Match by normalized title + similar duration
SELECT t.id, t.title, t.duration
FROM tracks t
WHERE t.release_group_id = ?
  AND t.title_normalized = normalize_title(?)
  AND ABS(t.duration - ?) < 10000  -- Within 10 seconds
LIMIT 1;
```

## UI/UX Design

### Library View (Browse Albums)

**V1 (Current)**:
```
╔════════════════════════════════════════════════════════╗
║ Albums                                                 ║
╠════════════════════════════════════════════════════════╣
║ [Cover] Highway 101                   Artist • 1980   ║
║         10 tracks                                      ║
║                                                        ║
║ [Cover] Highway 101 (Deluxe Edition)  Artist • 1980   ║
║         12 tracks                                      ║
║                                                        ║
║ [Cover] Highway 101 (Anniversary)     Artist • 2000   ║
║         20 tracks                                      ║
╚════════════════════════════════════════════════════════╝
```

**V2 (Release Groups)**:
```
╔════════════════════════════════════════════════════════╗
║ Albums                                                 ║
╠════════════════════════════════════════════════════════╣
║ [Cover] Highway 101                   Artist • 1980   ║
║         20 unique tracks • 3 releases                  ║
║         [▼ Show editions]                              ║
╚════════════════════════════════════════════════════════╝
```

**Expanded View**:
```
╔════════════════════════════════════════════════════════╗
║ [Cover] Highway 101                   Artist • 1980   ║
║         20 unique tracks • 3 releases                  ║
║         [▲ Hide editions]                              ║
║                                                        ║
║         Releases:                                      ║
║         • Original (1980) - 10 tracks                  ║
║         • Deluxe Edition (1980) - 12 tracks            ║
║         • 20th Anniversary (2000) - 20 tracks [★]      ║
╚════════════════════════════════════════════════════════╝
```

### Album Detail View

```
╔══════════════════════════════════════════════════════════════════╗
║ Highway 101                                      Artist Name     ║
║ 1980 • Album • 20 unique tracks                                  ║
╠══════════════════════════════════════════════════════════════════╣
║ [Album Cover]                                                    ║
║                                                                  ║
║ View: [Complete Collection ▼]                                   ║
║   • Complete Collection (20 tracks)                              ║
║   • Original (1980) - 10 tracks                                  ║
║   • Deluxe Edition (1980) - 12 tracks                            ║
║   • 20th Anniversary (2000) - 20 tracks [★ Primary]              ║
║                                                                  ║
║ ┌────────────────────────────────────────────────────────────┐  ║
║ │ Disc 1                                                     │  ║
║ │ 1. Track One                          3:45  [ODA]  ▶       │  ║
║ │ 2. Track Two                          4:12  [ODA]  ▶       │  ║
║ │ 3. Track Three                        3:28  [ODA]  ▶       │  ║
║ │ ...                                                        │  ║
║ │ 11. Bonus Track                       3:55  [DA]   ▶       │  ║
║ │     Added in Deluxe Edition (1980)                         │  ║
║ │ 12. Another Bonus                     4:20  [DA]   ▶       │  ║
║ │     Added in Deluxe Edition (1980)                         │  ║
║ │                                                            │  ║
║ │ Disc 2                                                     │  ║
║ │ 1. Remix Version                      5:10  [A]    ▶       │  ║
║ │    Added in Anniversary Edition (2000)                     │  ║
║ │ 2. Live Performance                   6:45  [A]    ▶       │  ║
║ │    Added in Anniversary Edition (2000)                     │  ║
║ │ ...                                                        │  ║
║ └────────────────────────────────────────────────────────────┘  ║
║                                                                  ║
║ Legend: O=Original D=Deluxe A=Anniversary                        ║
╚══════════════════════════════════════════════════════════════════╝
```

**Edition Switcher** changes view:

**View: Original (1980)**:
- Show only 10 original tracks
- Hide bonus tracks
- Play files from `/production/.../Album_Original_1980/`

**View: Complete Collection**:
- Show all 20 unique tracks
- Group by disc
- Play best quality version of each track (might mix files from different editions)

### Release Management Page

New admin page: **Manage Release Groups**

```
╔══════════════════════════════════════════════════════════╗
║ Release Group: Highway 101                              ║
╠══════════════════════════════════════════════════════════╣
║                                                          ║
║ Releases (3):                                            ║
║                                                          ║
║ ┌────────────────────────────────────────────────────┐  ║
║ │ [★] Original (1980)                                │  ║
║ │     • 10 tracks                                    │  ║
║ │     • Format: FLAC                                 │  ║
║ │     • Path: /production/.../Album_Original_1980/   │  ║
║ │     • Disk usage: 420 MB                           │  ║
║ │     [Set as Primary] [Delete]                      │  ║
║ └────────────────────────────────────────────────────┘  ║
║                                                          ║
║ ┌────────────────────────────────────────────────────┐  ║
║ │ Deluxe Edition (1980)                              │  ║
║ │     • 12 tracks (10 duplicates)                    │  ║
║ │     • Format: FLAC                                 │  ║
║ │     • Path: /production/.../Album_Deluxe_1980/     │  ║
║ │     • Disk usage: 480 MB (350 MB duplicates ⚠️)    │  ║
║ │     [Set as Primary] [Delete Duplicates] [Delete]  │  ║
║ └────────────────────────────────────────────────────┘  ║
║                                                          ║
║ ┌────────────────────────────────────────────────────┐  ║
║ │ 20th Anniversary (2000) [PRIMARY]                  │  ║
║ │     • 20 tracks (12 duplicates)                    │  ║
║ │     • Format: FLAC 96kHz/24bit                     │  ║
║ │     • Path: /production/.../Album_Anniversary_2000/│  ║
║ │     • Disk usage: 1.2 GB (600 MB duplicates ⚠️)    │  ║
║ │     [Set as Primary] [Delete Duplicates] [Delete]  │  ║
║ └────────────────────────────────────────────────────┘  ║
║                                                          ║
║ Total disk usage: 2.1 GB                                 ║
║ Potential savings: 950 MB (45%) if duplicates removed   ║
║                                                          ║
║ [Optimize Storage] - Delete duplicate files             ║
╚══════════════════════════════════════════════════════════╝
```

**Optimize Storage** action:
1. Keeps best quality version of each track
2. Deletes duplicate files from other releases
3. Updates `release_tracks` to point to remaining files
4. Shows summary: "Deleted 12 duplicate files, saved 950 MB"

## Duplicate Cleanup Strategies

User has options for managing duplicate files:

### Strategy 1: Keep All (Default)
- Preserve every edition exactly as released
- Historical completeness
- Maximum disk usage
- **Use case**: Archivists, collectors

### Strategy 2: Keep Best Quality
- For each track, keep only highest quality version
- Delete duplicate files from other editions
- `release_tracks` entries point to same file
- Significant disk savings (30-50%)
- **Use case**: Quality-focused users with limited disk

### Strategy 3: Keep Original + Best
- Preserve historical original release (even if lower quality)
- Keep best quality version if different from original
- Delete intermediate editions
- Moderate disk savings
- **Use case**: Balance between history and quality

### Strategy 4: Manual Review
- Show duplicate files side-by-side
- User chooses which to keep/delete per track
- Maximum control
- Most time-consuming
- **Use case**: Perfectionist audiophiles

## MusicBrainz Integration

### Automatic Matching

During staging review, query MusicBrainz API:

```go
func FindMusicBrainzReleaseGroup(artist, album string, year int) (*MBReleaseGroup, error) {
    // Query MusicBrainz API
    query := fmt.Sprintf("artist:%s AND release:%s AND date:%d", 
                        url.QueryEscape(artist),
                        url.QueryEscape(album),
                        year)
    
    resp, err := http.Get(fmt.Sprintf(
        "https://musicbrainz.org/ws/2/release-group/?query=%s&fmt=json",
        query,
    ))
    
    // Parse response, extract MBID
    // Return release group info
}
```

**Benefits**:
- Automatically link to authoritative music database
- Import canonical metadata (artist, title, year)
- Find related releases (other editions)
- Cover art from MusicBrainz
- Disambiguation (which "Weezer" album?)

### Semi-Automatic Consolidation

If MusicBrainz match found:

```
╔══════════════════════════════════════════════════════════╗
║ MusicBrainz Match Found                                  ║
╠══════════════════════════════════════════════════════════╣
║                                                          ║
║ Your album: Highway 101 (Deluxe Edition) - 1980         ║
║                                                          ║
║ MusicBrainz Release Group:                               ║
║   • Highway 101                                          ║
║   • Release Group ID: abc-123-def                        ║
║   • 4 known releases:                                    ║
║     - Original (1980, US, CD)                            ║
║     - Original (1980, JP, CD) [bonus track]              ║
║     - Deluxe Edition (1980, US, 2xCD)                    ║
║     - 20th Anniversary (2000, US, 2xCD)                  ║
║                                                          ║
║ Your library already has:                                ║
║   ✓ Original (1980) - matched to US release             ║
║                                                          ║
║ [✓] Link to MusicBrainz Release Group                   ║
║ [✓] Add as "Deluxe Edition" release                     ║
║                                                          ║
║          [Skip]  [Link & Continue]                       ║
╚══════════════════════════════════════════════════════════╝
```

## Migration Path

### Phase 1: V1 Working (Prerequisite)

Current workflow must be stable:
- ✅ Scan → SQLite → Process → Staging → Approve
- ✅ Simple one album = one `albums` entry
- ✅ Production system working with OpenSubsonic

### Phase 2: Schema Evolution

Add new tables without breaking V1:

```sql
-- Add release group tables (V2)
CREATE TABLE release_groups (...);
CREATE TABLE releases (...);
CREATE TABLE release_tracks (...);

-- Add new columns to existing tracks table
ALTER TABLE tracks ADD COLUMN release_group_id BIGINT REFERENCES release_groups(id);
ALTER TABLE tracks ADD COLUMN acoustic_fingerprint TEXT;
ALTER TABLE tracks ADD COLUMN primary_release_id BIGINT REFERENCES releases(id);

-- Keep album_id for backward compatibility during migration
-- Don't drop it until V2 migration complete
```

### Phase 3: Parallel Mode

Both systems coexist:

**New albums** (post-V2):
- Go through release group workflow
- `release_group_id` populated
- `album_id` = NULL

**Old albums** (pre-V2):
- Stay in `albums` table
- `album_id` populated
- `release_group_id` = NULL
- Can be migrated on-demand

### Phase 4: Batch Migration

Migrate existing albums to release groups:

```sql
-- For each existing album, create release group + release
INSERT INTO release_groups (artist_id, name, name_normalized, year, total_releases, total_unique_tracks)
SELECT 
    a.artist_id,
    a.title,
    normalize_album_name(a.title),
    a.year,
    1,  -- Single release initially
    (SELECT COUNT(*) FROM songs WHERE album_id = a.id)
FROM albums a;

-- Create corresponding releases
INSERT INTO releases (release_group_id, name, edition_type, release_year, track_count, is_primary)
SELECT 
    rg.id,
    rg.name,
    'original',
    rg.year,
    rg.total_unique_tracks,
    TRUE
FROM release_groups rg;

-- Update tracks to point to release groups
UPDATE tracks t
SET release_group_id = rg.id,
    primary_release_id = r.id
FROM albums a
JOIN release_groups rg ON rg.name_normalized = normalize_album_name(a.title) AND rg.artist_id = a.artist_id
JOIN releases r ON r.release_group_id = rg.id
WHERE t.album_id = a.id;

-- Create release_tracks entries
INSERT INTO release_tracks (release_id, track_id, disc_number, track_number, file_path, ...)
SELECT 
    t.primary_release_id,
    t.id,
    t.disc_number,
    t.track_number,
    t.file_path,
    ...
FROM tracks t
WHERE t.primary_release_id IS NOT NULL;
```

### Phase 5: Deprecate Old Schema

Once all albums migrated:

```sql
-- Drop old album_id column
ALTER TABLE tracks DROP COLUMN album_id;

-- Drop old albums table
DROP TABLE albums;

-- Rename release_groups → albums (for API compatibility)
-- Or update API to use "release groups" terminology
```

## API Changes

### V1 API (Unchanged)

```
GET /api/albums                    - List albums
GET /api/albums/:id                - Album details
GET /api/albums/:id/tracks         - Album tracks
```

**V1 Behavior**: Returns albums table data

### V2 API (New Endpoints)

```
GET /api/release-groups                        - List release groups
GET /api/release-groups/:id                    - Release group details
GET /api/release-groups/:id/releases           - List releases in group
GET /api/release-groups/:id/tracks             - All unique tracks (deduplicated)
GET /api/release-groups/:id/tracks?release=:id - Tracks filtered by release

GET /api/releases/:id                          - Release details
GET /api/releases/:id/tracks                   - Track listing with file paths

POST /api/staging/:id/consolidate              - Link to release group
GET  /api/staging/:id/potential-matches        - Find duplicate release groups

GET /api/admin/duplicates                      - List all duplicate files
POST /api/admin/optimize-storage               - Delete duplicates (keep best)
```

### OpenSubsonic Compatibility

Map release groups to OpenSubsonic `album` type:

```go
// When client requests /rest/getAlbum.view?id=123
func GetAlbum(releaseGroupID int64, releaseFilter string) OpenSubsonicAlbum {
    rg := getReleaseGroup(releaseGroupID)
    
    var tracks []Track
    if releaseFilter != "" {
        // User selected specific release (edition)
        tracks = getReleaseTracks(releaseFilter)
    } else {
        // Default: all unique tracks (deduplicated)
        tracks = getReleaseGroupTracks(releaseGroupID)
    }
    
    return OpenSubsonicAlbum{
        ID:     rg.ID,
        Name:   rg.Name,
        Artist: rg.Artist.Name,
        Year:   rg.Year,
        Tracks: tracks,
        // ...
    }
}
```

**Client sees**: Single "Highway 101" album (release group)  
**Server knows**: 3 releases, 20 deduplicated tracks

## Performance Implications

### Database Impact

**Additional joins**:
```sql
-- V1 query (simple)
SELECT a.*, t.*
FROM albums a
JOIN tracks t ON t.album_id = a.id
WHERE a.id = 123;

-- V2 query (more complex)
SELECT rg.*, t.*, rt.*
FROM release_groups rg
JOIN tracks t ON t.release_group_id = rg.id
LEFT JOIN release_tracks rt ON rt.track_id = t.id
WHERE rg.id = 123;
```

**Mitigation**:
- Proper indexes on `release_group_id`, `track_id`
- Materialized view for "complete collection" query
- Cache release group stats

### Fingerprinting Cost

**Chromaprint extraction**:
- ~500ms per track (CPU-intensive)
- Must decode entire audio file
- Can't parallelize too much (I/O bottleneck)

**Mitigation**:
- Generate fingerprints during scan phase (SQLite)
- Store in scan database
- Reuse when creating releases
- Background job for existing tracks

### Storage Overhead

**Duplicate files**:
- 30-50% of disk space in duplicates (estimated)
- Offset by quality-based cleanup

**Database size**:
- More tables, more rows
- Offset by better organization

## Implementation Phases

### Phase 4.1: Schema & Models
- [ ] Add `release_groups`, `releases`, `release_tracks` tables
- [ ] Update `tracks` table with new columns
- [ ] Create GORM models
- [ ] Add indexes
- [ ] Write migration scripts
- [ ] **Deliverable**: Schema ready, backward compatible with V1

### Phase 4.2: Acoustic Fingerprinting
- [ ] Integrate Chromaprint library
- [ ] Add fingerprint generation to scan phase
- [ ] Store fingerprints in SQLite scan DB
- [ ] Add fingerprint matching logic
- [ ] Build fallback matching (title + duration)
- [ ] **Deliverable**: Can identify duplicate tracks

### Phase 4.3: Release Group Creation
- [ ] Build album name normalization
- [ ] Implement fuzzy matching for release groups
- [ ] Create consolidation workflow (detect → prompt → merge)
- [ ] Build quality scoring algorithm
- [ ] Implement primary version selection
- [ ] **Deliverable**: Can consolidate editions during staging approval

### Phase 4.4: UI Integration
- [ ] Update library browse (show release groups)
- [ ] Build edition switcher in album detail
- [ ] Create release management page
- [ ] Add duplicate file browser
- [ ] Implement storage optimization UI
- [ ] **Deliverable**: Full UI for managing release groups

### Phase 4.5: MusicBrainz Integration
- [ ] Integrate MusicBrainz API client
- [ ] Add MBID search during staging
- [ ] Build semi-automatic linking UI
- [ ] Import canonical metadata
- [ ] **Deliverable**: Can auto-link to MusicBrainz

### Phase 4.6: Migration & Cleanup
- [ ] Batch migrate V1 albums → release groups
- [ ] Deprecate old `albums` table
- [ ] Update all API endpoints
- [ ] Add OpenSubsonic compatibility layer
- [ ] **Deliverable**: Complete V2 migration

## Success Metrics

### User Experience
- **Library Clarity**: 30-50% fewer album entries (editions consolidated)
- **Disk Awareness**: Show duplicate file counts and savings potential
- **Playback Flexibility**: Can switch between editions without browsing
- **Quality Confidence**: Always playing best available version

### Technical Performance
- **Fingerprint Accuracy**: > 95% duplicate detection rate
- **Matching Speed**: < 2 seconds to find potential release groups
- **Query Performance**: < 100ms for release group details
- **Storage Optimization**: 30-50% disk savings when duplicates removed

### Data Quality
- **MusicBrainz Linkage**: > 70% of albums linked to MBID
- **Edition Detection**: > 90% accuracy on edition type classification
- **Primary Version Selection**: > 95% user agreement with auto-selected best version

## Future Enhancements (V3+)

### Multi-Artist Compilations
- Various Artists release groups
- Per-track artist credits
- Compilation vs. studio album distinction

### Live Recordings
- Concert/venue metadata
- Recording date vs. release date
- Bootleg quality indicators

### Classical Music
- Work/composition hierarchy
- Performer vs. composer
- Movement-level granularity

### User Preferences
- Per-user primary release selection
- Custom quality scoring weights
- Edition visibility filters

## References

- **MusicBrainz Release Group**: https://musicbrainz.org/doc/Release_Group
- **MusicBrainz Release**: https://musicbrainz.org/doc/Release
- **Chromaprint/AcoustID**: https://acoustid.org/chromaprint
- **ISRC Standard**: https://www.usisrc.org/

---

## Summary

This V2 enhancement solves the long-standing album edition duplication problem by:

1. **Consolidating editions** into virtual release groups (MusicBrainz model)
2. **Deduplicating tracks** using acoustic fingerprinting
3. **Preserving all editions** with file-level tracking
4. **Optimizing storage** through intelligent duplicate cleanup
5. **Maintaining quality** by auto-selecting best versions

**User sees**: Single "Highway 101" with 20 tracks  
**System knows**: 3 releases, track-level deduplication, quality rankings

**Prerequisite**: V1 core workflow must be stable before implementing V2.
