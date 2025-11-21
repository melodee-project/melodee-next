# Melodee - Optimized Database Schema for High-Performance Music Library Management

## Overview

This document defines the optimized database schema for Melodee, designed specifically for handling massive music libraries (tens of millions of songs, 300k+ artists) with optimal performance. The schema incorporates partitioning, efficient indexing, and performance-oriented design patterns to address the scaling challenges of large music collections.

## Performance Optimization Strategy

### 1. Scale-First Design
- Schema optimized for 10M+ song libraries from day one
- Partitioning strategies for large tables
- Efficient indexing for common query patterns
- Directory code integration for filesystem performance

### 2. Query Performance Targets
- Sub-200ms response times for common API operations
- Efficient handling of large result sets (pagination, streaming)
- Minimal JOIN overhead for common operations
- Optimized for read-heavy workloads (streaming APIs)

## Core Tables

### Users Table
```sql
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    api_key UUID UNIQUE DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255),
    password_hash VARCHAR(255) NOT NULL,
    is_admin BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP WITH TIME ZONE,
    -- Performance optimization for login
    INDEX idx_users_username (username),
    INDEX idx_users_api_key (api_key)
);
```

### Libraries Table
```sql
CREATE TABLE libraries (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    path TEXT NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('inbound', 'staging', 'production')),
    is_locked BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    song_count INTEGER DEFAULT 0,
    album_count INTEGER DEFAULT 0,
    duration BIGINT DEFAULT 0, -- duration in milliseconds
    base_path VARCHAR(512) NOT NULL, -- For optimized file path storage
    INDEX idx_libraries_type (type),
    INDEX idx_libraries_production (type) WHERE type = 'production'
);
```

### Artists Table (Partitioned for Scale)
```sql
-- Main table with partitioning
CREATE TABLE artists (
    id BIGSERIAL PRIMARY KEY,
    api_key UUID UNIQUE DEFAULT gen_random_uuid(),
    is_locked BOOLEAN DEFAULT FALSE,
    name VARCHAR(255) NOT NULL,
    name_normalized VARCHAR(255) NOT NULL, -- For efficient searching
    directory_code VARCHAR(20), -- Directory code for filesystem performance
    sort_name VARCHAR(255),
    alternate_names TEXT[],
    song_count INTEGER DEFAULT 0, -- Pre-calculated for performance
    album_count INTEGER DEFAULT 0, -- Pre-calculated for performance
    duration BIGINT DEFAULT 0, -- Pre-calculated for performance
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_scanned_at TIMESTAMP WITH TIME ZONE,
    tags JSONB,
    musicbrainz_id UUID,
    spotify_id VARCHAR(255),
    lastfm_id VARCHAR(255),
    discogs_id VARCHAR(255),
    itunes_id VARCHAR(255),
    amg_id VARCHAR(255),
    wikidata_id VARCHAR(255),
    sort_order INTEGER DEFAULT 0
) PARTITION BY HASH (id);

-- Performance indexes
CREATE INDEX idx_artists_name_normalized_gin ON artists USING gin(name_normalized gin_trgm_ops);
CREATE INDEX idx_artists_directory_code ON artists(directory_code);
CREATE INDEX idx_artists_api_key ON artists(api_key);
CREATE INDEX idx_artists_musicbrainz_id ON artists(musicbrainz_id);
-- Partial index for active artists only
CREATE INDEX idx_artists_active ON artists(name_normalized, sort_order) WHERE is_locked = FALSE;

-- Create partitions (example with 4 partitions, can be scaled)
CREATE TABLE artists_0 PARTITION OF artists FOR VALUES WITH (MODULUS 4, REMAINDER 0);
CREATE TABLE artists_1 PARTITION OF artists FOR VALUES WITH (MODULUS 4, REMAINDER 1);
CREATE TABLE artists_2 PARTITION OF artists FOR VALUES WITH (MODULUS 4, REMAINDER 2);
CREATE TABLE artists_3 PARTITION OF artists FOR VALUES WITH (MODULUS 4, REMAINDER 3);
```

### Albums Table (Partitioned for Scale)
```sql
-- Main table with partitioning
CREATE TABLE albums (
    id BIGSERIAL PRIMARY KEY,
    api_key UUID UNIQUE DEFAULT gen_random_uuid(),
    is_locked BOOLEAN DEFAULT FALSE,
    name VARCHAR(255) NOT NULL,
    name_normalized VARCHAR(255) NOT NULL,
    alternate_names TEXT[],
    artist_id BIGINT REFERENCES artists(id),
    song_count INTEGER DEFAULT 0, -- Pre-calculated for performance
    duration BIGINT, -- duration in milliseconds
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    tags JSONB,
    release_date DATE,
    original_release_date DATE,
    album_status VARCHAR(50) DEFAULT 'New' CHECK (album_status IN ('New', 'Ok', 'Invalid')),
    album_type VARCHAR(50) DEFAULT 'NotSet' CHECK (album_type IN ('NotSet', 'Album', 'EP', 'Single', 'Compilation', 'Live', 'Remix', 'Soundtrack', 'SpokenWord', 'Interview', 'Audiobook')),
    directory VARCHAR(512) NOT NULL, -- Relative path from library base
    sort_name VARCHAR(255),
    sort_order INTEGER DEFAULT 0,
    image_count INTEGER DEFAULT 0,
    comment TEXT,
    description TEXT,
    genres TEXT[],
    moods TEXT[],
    notes TEXT,
    deezer_id VARCHAR(255),
    musicbrainz_id UUID,
    spotify_id VARCHAR(255),
    lastfm_id VARCHAR(255),
    discogs_id VARCHAR(255),
    itunes_id VARCHAR(255),
    amg_id VARCHAR(255),
    wikidata_id VARCHAR(255),
    is_compilation BOOLEAN DEFAULT FALSE
) PARTITION BY RANGE (created_at);

-- Performance indexes
CREATE INDEX idx_albums_artist_id ON albums(artist_id);
CREATE INDEX idx_albums_name_normalized_gin ON albums USING gin(name_normalized gin_trgm_ops);
CREATE INDEX idx_albums_api_key ON albums(api_key);
CREATE INDEX idx_albums_musicbrainz_id ON albums(musicbrainz_id);
-- Covering index for common API operations (getArtist, getAlbum)
CREATE INDEX idx_albums_artist_status_covering ON albums(artist_id, album_status, name_normalized, directory, sort_order) 
WHERE album_status = 'Ok';
-- Partial index for active albums only
CREATE INDEX idx_albums_active ON albums(artist_id, name_normalized, sort_order) WHERE album_status = 'Ok';

-- Create monthly partitions (example starting from 2025)
CREATE TABLE albums_2025_01 PARTITION OF albums 
FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
CREATE TABLE albums_2025_02 PARTITION OF albums 
FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');
-- Additional monthly partitions will be created automatically by application logic
```

### Songs Table (Partitioned for Scale - Most Critical for Performance)
```sql
-- Main table with partitioning by creation date for optimal performance
CREATE TABLE songs (
    id BIGSERIAL PRIMARY KEY,
    api_key UUID UNIQUE DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    name_normalized VARCHAR(255) NOT NULL,
    sort_name VARCHAR(255),
    album_id BIGINT REFERENCES albums(id),
    artist_id BIGINT REFERENCES artists(id), -- Denormalized for performance
    duration BIGINT, -- duration in milliseconds
    bit_rate INTEGER, -- in kbps
    bit_depth INTEGER,
    sample_rate INTEGER, -- in Hz
    channels INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    tags JSONB,
    directory VARCHAR(512) NOT NULL, -- Relative path from library base
    file_name TEXT NOT NULL, -- Just the filename for optimized storage
    relative_path TEXT NOT NULL, -- directory + file_name
    crc_hash VARCHAR(255) NOT NULL,
    sort_order INTEGER DEFAULT 0,
    -- Performance optimization indexes included at table level
    INDEX idx_songs_album_id_hash (album_id),
    INDEX idx_songs_artist_id_hash (artist_id)
) PARTITION BY RANGE (created_at);

-- Performance indexes (must be defined per partition)
-- Covering index for streaming operations (most critical)
CREATE INDEX idx_songs_album_order_covering ON songs(album_id, sort_order, name_normalized, relative_path, duration, api_key)
WHERE album_id IS NOT NULL;

-- Covering index for search operations
CREATE INDEX idx_songs_search_covering ON songs(name_normalized, artist_id, album_id, duration, relative_path)
WHERE id IN (
    SELECT id FROM songs s 
    JOIN albums a ON s.album_id = a.id 
    WHERE a.album_status = 'Ok'
);

-- Full-text search for advanced search capabilities
CREATE INDEX idx_songs_fulltext ON songs USING gin(to_tsvector('english', name_normalized || ' ' || COALESCE(tags->>'artist', '') || ' ' || COALESCE(tags->>'album', '')));

-- Partial index for active (Ok status) songs only
CREATE INDEX idx_songs_active ON songs(album_id, sort_order) WHERE album_id IN (
    SELECT id FROM albums WHERE album_status = 'Ok'
);

-- Create monthly partitions (example starting from 2025)
CREATE TABLE songs_2025_01 PARTITION OF songs 
FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
CREATE TABLE songs_2025_02 PARTITION OF songs 
FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');
-- Additional monthly partitions will be created automatically by application logic
```

### Playlists Table
```sql
CREATE TABLE playlists (
    id SERIAL PRIMARY KEY,
    api_key UUID UNIQUE DEFAULT gen_random_uuid(),
    user_id INTEGER REFERENCES users(id),
    name VARCHAR(255) NOT NULL,
    comment TEXT,
    public BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    changed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    duration BIGINT, -- duration in milliseconds
    song_count INTEGER,
    cover_art_id INTEGER, -- foreign key to images table
    INDEX idx_playlists_user_id (user_id),
    INDEX idx_playlists_api_key (api_key),
    INDEX idx_playlists_public (public) WHERE public = TRUE
);
```

### Playlist Songs Junction Table
```sql
CREATE TABLE playlist_songs (
    id SERIAL PRIMARY KEY,
    playlist_id INTEGER REFERENCES playlists(id) ON DELETE CASCADE,
    song_id INTEGER REFERENCES songs(id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(playlist_id, position),
    INDEX idx_playlist_songs_playlist_pos (playlist_id, position),
    INDEX idx_playlist_songs_song_id (song_id)
);
```

## Performance-Optimized Views and Materialized Views

### Materialized View for Library Statistics
```sql
-- Fast aggregate statistics for dashboard
CREATE MATERIALIZED VIEW library_stats AS
SELECT 
    l.id as library_id,
    l.name as library_name,
    l.type,
    COUNT(s.id) as total_songs,
    COUNT(DISTINCT a.id) as total_artists,
    COUNT(DISTINCT al.id) as total_albums,
    SUM(s.duration) as total_duration,
    SUM(s.bit_rate * s.duration / 8 / 1000) as approx_size_mb -- Approximate size
FROM libraries l
LEFT JOIN albums al ON al.directory LIKE l.path || '%'
LEFT JOIN songs s ON s.album_id = al.id
LEFT JOIN artists a ON a.id = al.artist_id
GROUP BY l.id, l.name, l.type;

CREATE INDEX idx_library_stats_library_id ON library_stats(library_id);
```

### View for Efficient Directory Code Resolution
```sql
-- Fast lookup for directory code to file path resolution
CREATE VIEW artist_directory_paths AS
SELECT 
    a.id,
    a.name,
    a.directory_code,
    a.sort_name,
    CONCAT(a.directory_code, '/', a.name) as calculated_path
FROM artists a
WHERE a.is_locked = FALSE;
```

## Advanced Performance Optimizations

### 1. Connection Pooling Configuration
Optimized for high-concurrency streaming:
```sql
-- Recommended PostgreSQL settings for massive libraries:
-- max_connections = 200
-- shared_buffers = 25% of RAM
-- effective_cache_size = 75% of RAM
-- maintenance_work_mem = 1GB
-- checkpoint_completion_target = 0.9
-- wal_buffers = 16MB
-- default_statistics_target = 100
```

### 2. Sequence Optimization
Using BIGSERIAL for all ID columns to handle massive scale from day one.

### 3. Partial Indexes for Active Content
Most queries will be for 'Ok' status albums/songs, so partial indexes optimize these common operations.

### 4. Covering Indexes
Critical for API response performance, reducing the need for table lookups.

### 5. Materialized Views for Aggregates
Pre-calculated statistics to avoid expensive aggregation queries on large datasets.

## Maintenance and Scalability Considerations

### 1. Partition Management
- Automatic monthly partition creation for new data
- Archive old partitions for inactive content
- Regular ANALYZE operations on partitions

**Partition Playbook**
- Creation cadence: job `partition:create-next-month` runs weekly; creates `albums_YYYY_MM` and `songs_YYYY_MM` with covering/partial indexes applied per partition.
- Index template per new partition:
  - `albums`: `idx_albums_artist_id`, `idx_albums_name_normalized_gin`, `idx_albums_artist_status_covering` (partial), `idx_albums_active` (partial).
  - `songs`: `idx_songs_album_id_hash`, `idx_songs_artist_id_hash`, `idx_songs_album_order_covering` (partial), `idx_songs_search_covering` (partial), `idx_songs_fulltext`, `idx_songs_active` (partial).
- Retention: partitions older than 36 months move to `archive` schema; keep indexes but set `autovacuum_freeze_max_age` tuned for cold data.
- Verification: after creation run `EXPLAIN (ANALYZE,BUFFERS)` for `getAlbum` and `stream` sample queries to ensure index usage; record results in release notes.
- Idempotency: partition create scripts use `IF NOT EXISTS` and transactional execution; safe to rerun.

### 2. Statistics Updates
- Frequent statistics updates for large tables
- Custom statistics for complex queries
- Monitoring query plans for optimization

### 3. Monitoring Points
- Slow query logging
- Index usage statistics
- Partition hit rates
- Connection pool utilization

## Additional User Interaction Tables

### User Songs Table (for tracking user interactions with songs)
```sql
CREATE TABLE user_songs (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    song_id BIGINT NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    played_count INTEGER DEFAULT 0,
    last_played_at TIMESTAMP WITH TIME ZONE,
    is_starred BOOLEAN DEFAULT FALSE,
    is_hated BOOLEAN DEFAULT FALSE, -- When true, don't include in randomization
    starred_at TIMESTAMP WITH TIME ZONE,
    rating INTEGER DEFAULT 0 CHECK (rating >= 0 AND rating <= 5),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, song_id),
    INDEX idx_user_songs_user_id (user_id),
    INDEX idx_user_songs_song_id (song_id),
    INDEX idx_user_songs_last_played (last_played_at),
    INDEX idx_user_songs_starred (is_starred),
    INDEX idx_user_songs_played_count (played_count)
);
```

### User Albums Table (for tracking user interactions with albums)
```sql
CREATE TABLE user_albums (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    album_id BIGINT NOT NULL REFERENCES albums(id) ON DELETE CASCADE,
    played_count INTEGER DEFAULT 0,
    last_played_at TIMESTAMP WITH TIME ZONE,
    is_starred BOOLEAN DEFAULT FALSE,
    is_hated BOOLEAN DEFAULT FALSE, -- When true, don't include in randomization
    starred_at TIMESTAMP WITH TIME ZONE,
    rating INTEGER DEFAULT 0 CHECK (rating >= 0 AND rating <= 5),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, album_id),
    INDEX idx_user_albums_user_id (user_id),
    INDEX idx_user_albums_album_id (album_id),
    INDEX idx_user_albums_last_played (last_played_at),
    INDEX idx_user_albums_starred (is_starred),
    INDEX idx_user_albums_played_count (played_count)
);
```

### User Artists Table (for tracking user interactions with artists)
```sql
CREATE TABLE user_artists (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    artist_id BIGINT NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    is_starred BOOLEAN DEFAULT FALSE,
    is_hated BOOLEAN DEFAULT FALSE, -- When true, don't include in randomization
    starred_at TIMESTAMP WITH TIME ZONE,
    rating INTEGER DEFAULT 0 CHECK (rating >= 0 AND rating <= 5),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, artist_id),
    INDEX idx_user_artists_user_id (user_id),
    INDEX idx_user_artists_artist_id (artist_id),
    INDEX idx_user_artists_starred (is_starred)
);
```

### User Pins Table (for pinned content)
```sql
CREATE TABLE user_pins (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    song_id BIGINT REFERENCES songs(id) ON DELETE CASCADE,
    album_id BIGINT REFERENCES albums(id) ON DELETE CASCADE,
    artist_id BIGINT REFERENCES artists(id) ON DELETE CASCADE,
    pinned_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_pins_user_id (user_id),
    INDEX idx_user_pins_song_id (song_id),
    INDEX idx_user_pins_album_id (album_id),
    INDEX idx_user_pins_artist_id (artist_id)
);
```

### Bookmarks Table (for user bookmarks)
```sql
CREATE TABLE bookmarks (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    song_id BIGINT NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    comment TEXT,
    position INTEGER NOT NULL, -- Position in milliseconds
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, song_id),
    INDEX idx_bookmarks_user_id (user_id),
    INDEX idx_bookmarks_song_id (song_id)
);
```

### Players Table (for tracking user players/devices)
```sql
CREATE TABLE players (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    user_agent TEXT,
    user_id INTEGER NOT NULL REFERENCES users(id),
    client VARCHAR(500) NOT NULL,
    ip_address VARCHAR(45), -- Support for IPv6 addresses
    last_seen_at TIMESTAMP WITH TIME ZONE NOT NULL,
    max_bitrate INTEGER, -- Maximum bitrate for this player
    scrobble_enabled BOOLEAN DEFAULT TRUE,
    transcoding_id VARCHAR(255),
    hostname VARCHAR(500),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_players_user_id (user_id),
    INDEX idx_players_client (client),
    INDEX idx_players_last_seen (last_seen_at)
);
```

### Play Queues Table (for managing play queues)
```sql
CREATE TABLE play_queues (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    song_id BIGINT NOT NULL REFERENCES songs(id),
    song_api_key UUID NOT NULL, -- To not expose internal song IDs to API consumers
    is_current_song BOOLEAN DEFAULT FALSE,
    changed_by VARCHAR(255) NOT NULL,
    position DOUBLE PRECISION NOT NULL DEFAULT 0,
    play_queue_id INTEGER NOT NULL, -- To manage order in the queue
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_play_queues_user_id (user_id),
    INDEX idx_play_queues_song_id (song_id),
    INDEX idx_play_queues_current_song (is_current_song),
    INDEX idx_play_queues_play_queue_id (play_queue_id)
);
```

### Search Histories Table (for tracking user search history)
```sql
CREATE TABLE search_histories (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    search_term VARCHAR(500) NOT NULL,
    search_type VARCHAR(50) NOT NULL CHECK (search_type IN ('artist', 'album', 'song', 'any')),
    results_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_search_histories_user_id (user_id),
    INDEX idx_search_histories_search_term_gin (search_term gin_trgm_ops),
    INDEX idx_search_histories_search_type (search_type)
);
```

### Shares Table (for shared content)
```sql
CREATE TABLE shares (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    name VARCHAR(255),
    description TEXT,
    expires_at TIMESTAMP WITH TIME ZONE,
    max_streaming_minutes INTEGER,
    max_streaming_count INTEGER,
    allow_streaming BOOLEAN DEFAULT TRUE,
    allow_download BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

### Share Activities Table (for tracking share usage)
```sql
CREATE TABLE share_activities (
    id SERIAL PRIMARY KEY,
    share_id INTEGER NOT NULL REFERENCES shares(id) ON DELETE CASCADE,
    user_id INTEGER REFERENCES users(id), -- User who accessed (null if anonymous)
    ip_address VARCHAR(45),
    accessed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    user_agent TEXT,
    INDEX idx_share_activities_share_id (share_id),
    INDEX idx_share_activities_accessed_at (accessed_at)
);
```

### Library Scan Histories Table (for tracking library scanning)
```sql
CREATE TABLE library_scan_histories (
    id SERIAL PRIMARY KEY,
    library_id INTEGER NOT NULL REFERENCES libraries(id),
    status VARCHAR(50) NOT NULL CHECK (status IN ('started', 'in_progress', 'completed', 'failed')),
    message TEXT,
    total_files INTEGER DEFAULT 0,
    processed_files INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_lib_scan_hist_lib_id (library_id),
    INDEX idx_lib_scan_hist_status (status),
    INDEX idx_lib_scan_hist_created_at (created_at)
);
```

### Settings Table (for application configuration)
```sql
CREATE TABLE settings (
    id SERIAL PRIMARY KEY,
    key VARCHAR(500) UNIQUE NOT NULL,
    value TEXT,
    category INTEGER, -- Enum for setting category
    comment TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_settings_key (key)
);
```

### Artist Relations Table (for artist relationships)
```sql
CREATE TABLE artist_relations (
    id SERIAL PRIMARY KEY,
    from_artist_id BIGINT NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    to_artist_id BIGINT NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    relation_type VARCHAR(100) NOT NULL, -- e.g., 'member', 'collaborator', 'influenced_by'
    relation_start DATE, -- When the relationship started
    relation_end DATE, -- When the relationship ended
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    is_locked BOOLEAN DEFAULT FALSE,
    sort_order INTEGER DEFAULT 0,
    api_key UUID UNIQUE DEFAULT gen_random_uuid(),
    tags JSONB,
    notes TEXT,
    description TEXT,
    UNIQUE(from_artist_id, to_artist_id, relation_type),
    INDEX idx_artist_relations_from (from_artist_id),
    INDEX idx_artist_relations_to (to_artist_id),
    INDEX idx_artist_relations_type (relation_type)
);
```

### Radio Stations Table
```sql
CREATE TABLE radio_stations (
    id SERIAL PRIMARY KEY,
    api_key UUID UNIQUE DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    stream_url TEXT NOT NULL,
    home_page_url TEXT,
    created_by_user_id INTEGER REFERENCES users(id),
    song_count INTEGER DEFAULT 0,
    is_enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_radio_stations_created_by (created_by_user_id),
    INDEX idx_radio_stations_enabled (is_enabled)
);
```

### Contributors Table (for song contributors like composers, performers, etc.)
```sql
CREATE TABLE contributors (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(100) NOT NULL, -- e.g., 'performer', 'composer', 'producer'
    sort_name VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_contributors_name (name),
    INDEX idx_contributors_type (type)
);
```

## Performance-Optimized Aggregated Views

### User Play Statistics View
```sql
-- Fast lookup for user play counts and statistics
CREATE MATERIALIZED VIEW user_play_statistics AS
SELECT
    u.id as user_id,
    u.username,
    COALESCE(SUM(us.played_count), 0) as total_plays,
    COUNT(DISTINCT CASE WHEN us.is_starred = TRUE THEN us.song_id END) as total_starred_songs,
    COUNT(DISTINCT CASE WHEN us.is_hated = TRUE THEN us.song_id END) as total_hated_songs,
    COALESCE(MAX(us.last_played_at), '1970-01-01'::TIMESTAMP WITH TIME ZONE) as last_played_at
FROM users u
LEFT JOIN user_songs us ON u.id = us.user_id
GROUP BY u.id, u.username;

CREATE INDEX idx_user_play_stats_user_id ON user_play_statistics(user_id);
```

## Data Integrity and Constraints

All original business logic constraints are maintained:
- Foreign key relationships ensure data integrity
- Check constraints on status fields
- Unique constraints on API keys
- Proper cascading rules for dependent records
- Comprehensive user interaction tracking for likes, plays, ratings, etc.

This schema is designed to support the extreme scale requirements mentioned in the PRD while maintaining the performance targets needed for an efficient music streaming system. It includes comprehensive tracking of user interactions including likes, plays, ratings, bookmarks, and play history.
