-- Melodee Database Schema
-- This file is managed manually - recreate containers to apply changes
-- PostgreSQL 17 compatible

-- Required Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "btree_gin";

-- Users Table
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    api_key UUID UNIQUE NOT NULL DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255),
    password_hash VARCHAR(255) NOT NULL,
    is_admin BOOLEAN DEFAULT FALSE,
    failed_login_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMP,
    password_reset_token VARCHAR(255),
    password_reset_expiry TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_users_api_key ON users (api_key);

-- Libraries Table
CREATE TABLE IF NOT EXISTS libraries (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    path TEXT NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('inbound', 'staging', 'production')),
    is_locked BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    track_count INTEGER DEFAULT 0,
    album_count INTEGER DEFAULT 0,
    duration BIGINT DEFAULT 0
);

-- Settings Table
CREATE TABLE IF NOT EXISTS settings (
    id SERIAL PRIMARY KEY,
    key VARCHAR(500) UNIQUE NOT NULL,
    value TEXT,
    category INTEGER,
    comment TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Log Entries Table
CREATE TABLE IF NOT EXISTS log_entries (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    level VARCHAR(10) NOT NULL CHECK (level IN ('debug', 'info', 'warn', 'error', 'fatal', 'panic')),
    message TEXT NOT NULL,
    module VARCHAR(100),
    function VARCHAR(100),
    request_id VARCHAR(50),
    user_id BIGINT,
    trace_id VARCHAR(50),
    ip VARCHAR(50),
    route VARCHAR(255),
    status INTEGER,
    duration BIGINT,
    queue VARCHAR(50),
    job_type VARCHAR(100),
    library_id INTEGER,
    file_path VARCHAR(1024),
    error TEXT,
    stack TEXT,
    metadata JSONB
);

-- Log Entries Indexes
CREATE INDEX IF NOT EXISTS idx_log_entries_timestamp ON log_entries (timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_log_entries_level ON log_entries (level);
CREATE INDEX IF NOT EXISTS idx_log_entries_module ON log_entries (module);
CREATE INDEX IF NOT EXISTS idx_log_entries_request_id ON log_entries (request_id);
CREATE INDEX IF NOT EXISTS idx_log_entries_user_id ON log_entries (user_id);
CREATE INDEX IF NOT EXISTS idx_log_entries_library_id ON log_entries (library_id);
CREATE INDEX IF NOT EXISTS idx_log_entries_level_timestamp ON log_entries (level, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_log_entries_message_search ON log_entries USING gin(to_tsvector('english', message));
CREATE INDEX IF NOT EXISTS idx_log_entries_error_search ON log_entries USING gin(to_tsvector('english', coalesce(error, '')));

-- Artists Table (simplified - GORM will add remaining columns as needed)
CREATE TABLE IF NOT EXISTS artists (
    id BIGSERIAL PRIMARY KEY,
    api_key UUID UNIQUE NOT NULL DEFAULT gen_random_uuid(),
    is_locked BOOLEAN DEFAULT FALSE,
    name VARCHAR(255) NOT NULL,
    name_normalized VARCHAR(255) NOT NULL,
    directory_code VARCHAR(20),
    sort_name VARCHAR(255),
    alternate_names TEXT[],
    track_count INTEGER DEFAULT 0,
    album_count INTEGER DEFAULT 0,
    duration BIGINT DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_artists_name_normalized ON artists USING gin(name_normalized gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_artists_directory_code ON artists (directory_code);

-- Albums Table (simplified)
CREATE TABLE IF NOT EXISTS albums (
    id BIGSERIAL PRIMARY KEY,
    api_key UUID UNIQUE NOT NULL DEFAULT gen_random_uuid(),
    artist_id BIGINT REFERENCES artists(id) ON DELETE CASCADE,
    library_id INTEGER REFERENCES libraries(id) ON DELETE CASCADE,
    is_locked BOOLEAN DEFAULT FALSE,
    name VARCHAR(255) NOT NULL,
    name_normalized VARCHAR(255) NOT NULL,
    directory VARCHAR(500),
    album_type VARCHAR(50),
    track_count INTEGER DEFAULT 0,
    duration BIGINT DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_albums_artist_id ON albums (artist_id);
CREATE INDEX IF NOT EXISTS idx_albums_library_id ON albums (library_id);
CREATE INDEX IF NOT EXISTS idx_albums_name_normalized ON albums USING gin(name_normalized gin_trgm_ops);

-- Tracks Table (simplified)
CREATE TABLE IF NOT EXISTS tracks (
    id BIGSERIAL PRIMARY KEY,
    api_key UUID UNIQUE NOT NULL DEFAULT gen_random_uuid(),
    album_id BIGINT REFERENCES albums(id) ON DELETE CASCADE,
    artist_id BIGINT REFERENCES artists(id) ON DELETE CASCADE,
    library_id INTEGER REFERENCES libraries(id) ON DELETE CASCADE,
    is_locked BOOLEAN DEFAULT FALSE,
    title VARCHAR(255) NOT NULL,
    title_normalized VARCHAR(255) NOT NULL,
    relative_path TEXT,
    file_name VARCHAR(500),
    duration BIGINT DEFAULT 0,
    bit_rate INTEGER,
    sample_rate INTEGER,
    channels INTEGER,
    file_size BIGINT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_tracks_album_id ON tracks (album_id);
CREATE INDEX IF NOT EXISTS idx_tracks_artist_id ON tracks (artist_id);
CREATE INDEX IF NOT EXISTS idx_tracks_library_id ON tracks (library_id);
CREATE INDEX IF NOT EXISTS idx_tracks_title_normalized ON tracks USING gin(title_normalized gin_trgm_ops);

-- Playlists Table
CREATE TABLE IF NOT EXISTS playlists (
    id SERIAL PRIMARY KEY,
    api_key UUID UNIQUE DEFAULT gen_random_uuid(),
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    comment TEXT,
    public BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    changed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    duration BIGINT DEFAULT 0,
    track_count INTEGER DEFAULT 0,
    cover_art_id INTEGER
);
CREATE INDEX IF NOT EXISTS idx_playlists_user_id ON playlists (user_id);
CREATE INDEX IF NOT EXISTS idx_playlists_api_key ON playlists (api_key);
CREATE INDEX IF NOT EXISTS idx_playlists_public_where ON playlists (public) WHERE public = true;

-- Playlist Tracks (Junction Table)
CREATE TABLE IF NOT EXISTS playlist_tracks (
    id SERIAL PRIMARY KEY,
    playlist_id INTEGER REFERENCES playlists(id) ON DELETE CASCADE,
    track_id BIGINT REFERENCES tracks(id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(playlist_id, track_id)
);
CREATE INDEX IF NOT EXISTS idx_playlist_tracks_playlist_id ON playlist_tracks (playlist_id);
CREATE INDEX IF NOT EXISTS idx_playlist_tracks_track_id ON playlist_tracks (track_id);

-- Shares Table
CREATE TABLE IF NOT EXISTS shares (
    id SERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    resource_type VARCHAR(50) NOT NULL,
    resource_id BIGINT NOT NULL,
    description TEXT,
    expires_at TIMESTAMP,
    visit_count INTEGER DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_shares_user_id ON shares (user_id);

-- Capacity Status Table  
CREATE TABLE IF NOT EXISTS capacity_statuses (
    id SERIAL PRIMARY KEY,
    library_id INTEGER REFERENCES libraries(id) ON DELETE CASCADE,
    total_space BIGINT NOT NULL,
    used_space BIGINT NOT NULL,
    available_space BIGINT NOT NULL,
    is_healthy BOOLEAN DEFAULT TRUE,
    last_check_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_capacity_statuses_library_id ON capacity_statuses (library_id);

-- Library Scan History
CREATE TABLE IF NOT EXISTS library_scan_histories (
    id BIGSERIAL PRIMARY KEY,
    library_id INTEGER REFERENCES libraries(id) ON DELETE CASCADE,
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    status VARCHAR(50) NOT NULL,
    files_scanned INTEGER DEFAULT 0,
    files_added INTEGER DEFAULT 0,
    files_updated INTEGER DEFAULT 0,
    files_deleted INTEGER DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_scan_histories_library_id ON library_scan_histories (library_id);
CREATE INDEX IF NOT EXISTS idx_scan_histories_started_at ON library_scan_histories (started_at DESC);

-- Staging Items Table (for file-based staging workflow)
CREATE TABLE IF NOT EXISTS staging_items (
    id BIGSERIAL PRIMARY KEY,
    scan_id TEXT NOT NULL,
    staging_path TEXT NOT NULL UNIQUE,
    metadata_file TEXT NOT NULL,
    artist_name TEXT NOT NULL,
    album_name TEXT NOT NULL,
    track_count INTEGER DEFAULT 0,
    total_size BIGINT DEFAULT 0,
    processed_at TIMESTAMP NOT NULL,
    status VARCHAR(50) NOT NULL CHECK (status IN ('pending_review', 'approved', 'rejected')),
    reviewed_by BIGINT REFERENCES users(id),
    reviewed_at TIMESTAMP,
    notes TEXT,
    checksum TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_staging_status ON staging_items(status);
CREATE INDEX IF NOT EXISTS idx_staging_scan_id ON staging_items(scan_id);
CREATE INDEX IF NOT EXISTS idx_staging_artist_album ON staging_items(artist_name, album_name);

-- Grant all privileges on tables and sequences to melodee_user
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO melodee_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO melodee_user;
