# Melodee - Database Schema for Music Library Management

**Audience:** Backend engineers, DBAs, performance engineers

**Purpose:** Canonical definition of Melodee's relational schema and performance strategies.

**Source of truth for:** Table/column definitions, indexes, views, and DB performance playbook.

## Overview

This document defines the database schema for Melodee, designed for managing music libraries efficiently. The schema uses GORM for automatic schema management with PostgreSQL-specific optimizations including GIN indexes for fuzzy text search and proper foreign key relationships.

## Schema Management Strategy

### 1. GORM-Driven Schema
- All tables created and managed via GORM AutoMigrate
- Schema defined in Go models (`src/internal/models/models.go`)
- Indexes declared via GORM struct tags
- PostgreSQL extensions managed separately (uuid-ossp, pg_trgm, btree_gin)

### 2. Query Performance Targets
- Sub-200ms response times for common API operations
- Efficient handling of large result sets (pagination, streaming)
- Minimal JOIN overhead for common operations
- Optimized for read-heavy workloads (streaming APIs)

### 3. Future Scaling Considerations
- Current design uses standard PostgreSQL tables with comprehensive indexing
- Partitioning can be added later if needed at scale (10M+ songs)
- Schema changes should be made in Go models, not raw SQL

## Table Overview

All tables are defined in `src/internal/models/models.go` and created/managed by GORM AutoMigrate.

### Core Music Tables

**Artists** - Artist metadata with external ID mappings (MusicBrainz, Spotify, Last.fm, etc.)
- Primary identifier: `id`, `api_key` (UUID)
- Search: `name_normalized` with GIN index for fuzzy matching
- Filesystem: `directory_code` for organizing files
- Cached aggregates: `song_count`, `album_count`, `duration`

**Albums** - Album metadata with artist relationships
- Primary identifier: `id`, `api_key` (UUID)  
- Foreign key: `artist_id`
- Search: `name_normalized` with GIN index
- Filesystem: `directory` (relative path from library)
- Status: `album_status` (New/Ok/Invalid), `album_type` (Album/EP/Single/etc.)
- Cached: `song_count`, `duration`

**Songs** - Individual track metadata with full audio details
- Primary identifier: `id`, `api_key` (UUID)
- Foreign keys: `album_id`, `artist_id`
- Search: `name_normalized` with GIN index
- Filesystem: `relative_path`, `file_name`, `crc_hash` (deduplication)
- Audio: `duration`, `bit_rate`, `bit_depth`, `sample_rate`, `channels`

### User & Library Management

**Users** - User accounts with authentication
**Libraries** - Media library roots (inbound/staging/production)
**Playlists** - User-created playlists
**PlaylistSongs** - Junction table for playlist membership

### User Interaction Tracking

**UserSongs** - Per-user song play counts, ratings, starred status
**UserAlbums** - Per-user album play counts, ratings
**UserArtists** - Per-user artist preferences
**UserPins** - Pinned items for quick access
**Bookmarks** - Resume positions in tracks

### Playback & Sharing

**Players** - Active player sessions
**PlayQueues** - Current playback queues
**Shares** - Shared content links
**ShareActivities** - Share access logs

### System Tables

**Settings** - Application configuration
**SearchHistories** - User search tracking
**LibraryScanHistories** - Media scan audit trail
**ArtistRelations** - Artist collaboration/relationship graph
**RadioStations** - Internet radio stations
**Contributors** - Track-level contributor metadata
**CapacityStatus** - Storage capacity monitoring




## PostgreSQL Extensions

The following PostgreSQL extensions are automatically created during migrations:

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";  -- UUID generation
CREATE EXTENSION IF NOT EXISTS "pg_trgm";     -- Fuzzy text search (trigram matching)
CREATE EXTENSION IF NOT EXISTS "btree_gin";   -- GIN index support for btree-indexable types
```

These extensions enable:
- Automatic UUID generation for API keys
- Fast fuzzy text search on artist/album/song names
- Efficient GIN indexes for text search operations

## Performance Considerations

### Current Approach
- Standard PostgreSQL tables with comprehensive indexing via GORM
- GIN indexes for fuzzy text search using pg_trgm
- Foreign key indexes for efficient joins
- Suitable for most music library sizes

### Future Scaling Options

If you reach massive scale (10M+ songs, 100k+ artists), consider:

1. **Table Partitioning** - Partition large tables (artists, albums, songs) by hash or range
2. **Connection Pooling** - Tune PostgreSQL connection settings for high concurrency
3. **Materialized Views** - Pre-calculate expensive aggregations
4. **Read Replicas** - Separate read and write workloads

The current schema is designed to scale vertically first, with clear migration paths to horizontal scaling if needed.

## Schema Modification Process

All schema changes must be made through Go models:

1. Update struct definitions in `src/internal/models/models.go`
2. Add/modify GORM tags for indexes, constraints, defaults
3. Run the application - GORM AutoMigrate handles schema updates
4. Document high-level changes in this file for reference

**Do not** create manual SQL migrations for tables managed by GORM - this creates conflicts and inconsistencies.

## Index Strategy

GORM automatically creates indexes based on struct tags. Key index types used:

- **B-tree indexes** - Default for primary keys, foreign keys, unique constraints
- **GIN indexes** - Fuzzy text search on normalized names using pg_trgm extension
- **Composite indexes** - Multiple columns for complex query patterns (defined in GORM tags)

To see exact index definitions, inspect `src/internal/models/models.go` struct tags or query the database directly.

## Data Integrity

GORM automatically manages:
- Foreign key relationships with proper CASCADE rules
- Check constraints on enum-like fields (via model validation)
- Unique constraints on API keys and composite keys
- NOT NULL constraints as specified in struct tags
- Default values for timestamps, booleans, and integers

All business logic constraints defined in the Go models are enforced at the database level through GORM's AutoMigrate.

