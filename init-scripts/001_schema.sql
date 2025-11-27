-- Melodee Database Schema
-- This file is managed manually - recreate containers to apply changes
-- PostgreSQL 17 compatible

-- Required Extensions (already created in init_db.sh)
-- uuid-ossp, pg_trgm, btree_gin

-- Note: GORM uses "melodee_" prefix for all tables

-- Users Table
CREATE TABLE IF NOT EXISTS melodee_users (
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
CREATE INDEX IF NOT EXISTS idx_melodee_users_api_key ON melodee_users (api_key);

-- Libraries Table
CREATE TABLE IF NOT EXISTS melodee_libraries (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    path TEXT NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('inbound', 'staging', 'production')),
    is_locked BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    song_count INTEGER DEFAULT 0,
    album_count INTEGER DEFAULT 0,
    duration BIGINT DEFAULT 0,
    base_path VARCHAR(512) NOT NULL
);

-- Settings Table
CREATE TABLE IF NOT EXISTS melodee_settings (
    id SERIAL PRIMARY KEY,
    key VARCHAR(500) UNIQUE NOT NULL,
    value TEXT,
    category INTEGER,
    comment TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Log Entries Table
CREATE TABLE IF NOT EXISTS melodee_log_entries (
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
CREATE INDEX IF NOT EXISTS idx_log_entries_timestamp ON melodee_log_entries (timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_log_entries_level ON melodee_log_entries (level);
CREATE INDEX IF NOT EXISTS idx_log_entries_module ON melodee_log_entries (module);
CREATE INDEX IF NOT EXISTS idx_log_entries_request_id ON melodee_log_entries (request_id);
CREATE INDEX IF NOT EXISTS idx_log_entries_user_id ON melodee_log_entries (user_id);
CREATE INDEX IF NOT EXISTS idx_log_entries_library_id ON melodee_log_entries (library_id);
CREATE INDEX IF NOT EXISTS idx_log_entries_level_timestamp ON melodee_log_entries (level, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_log_entries_message_search ON melodee_log_entries USING gin(to_tsvector('english', message));
CREATE INDEX IF NOT EXISTS idx_log_entries_error_search ON melodee_log_entries USING gin(to_tsvector('english', coalesce(error, '')));

-- Artists Table (simplified - GORM will add remaining columns as needed)
CREATE TABLE IF NOT EXISTS melodee_artists (
    id BIGSERIAL PRIMARY KEY,
    api_key UUID UNIQUE NOT NULL DEFAULT gen_random_uuid(),
    is_locked BOOLEAN DEFAULT FALSE,
    name VARCHAR(255) NOT NULL,
    name_normalized VARCHAR(255) NOT NULL,
    directory_code VARCHAR(20),
    sort_name VARCHAR(255),
    alternate_names TEXT[],
    song_count INTEGER DEFAULT 0,
    album_count INTEGER DEFAULT 0,
    duration BIGINT DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_artists_name_normalized ON melodee_artists USING gin(name_normalized gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_artists_directory_code ON melodee_artists (directory_code);

-- Albums Table (simplified)
CREATE TABLE IF NOT EXISTS melodee_albums (
    id BIGSERIAL PRIMARY KEY,
    api_key UUID UNIQUE NOT NULL DEFAULT gen_random_uuid(),
    artist_id BIGINT REFERENCES melodee_artists(id) ON DELETE CASCADE,
    library_id INTEGER REFERENCES melodee_libraries(id) ON DELETE CASCADE,
    is_locked BOOLEAN DEFAULT FALSE,
    name VARCHAR(255) NOT NULL,
    name_normalized VARCHAR(255) NOT NULL,
    directory VARCHAR(500),
    album_type VARCHAR(50),
    album_status VARCHAR(50) DEFAULT 'New',
    song_count INTEGER DEFAULT 0,
    duration BIGINT DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_albums_artist_id ON melodee_albums (artist_id);
CREATE INDEX IF NOT EXISTS idx_albums_library_id ON melodee_albums (library_id);
CREATE INDEX IF NOT EXISTS idx_albums_name_normalized ON melodee_albums USING gin(name_normalized gin_trgm_ops);

-- Songs Table (simplified)
CREATE TABLE IF NOT EXISTS melodee_songs (
    id BIGSERIAL PRIMARY KEY,
    api_key UUID UNIQUE NOT NULL DEFAULT gen_random_uuid(),
    album_id BIGINT REFERENCES melodee_albums(id) ON DELETE CASCADE,
    artist_id BIGINT REFERENCES melodee_artists(id) ON DELETE CASCADE,
    library_id INTEGER REFERENCES melodee_libraries(id) ON DELETE CASCADE,
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
CREATE INDEX IF NOT EXISTS idx_songs_album_id ON melodee_songs (album_id);
CREATE INDEX IF NOT EXISTS idx_songs_artist_id ON melodee_songs (artist_id);
CREATE INDEX IF NOT EXISTS idx_songs_library_id ON melodee_songs (library_id);
CREATE INDEX IF NOT EXISTS idx_songs_title_normalized ON melodee_songs USING gin(title_normalized gin_trgm_ops);

-- Playlists Table
CREATE TABLE IF NOT EXISTS melodee_playlists (
    id SERIAL PRIMARY KEY,
    api_key UUID UNIQUE DEFAULT gen_random_uuid(),
    user_id BIGINT REFERENCES melodee_users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    comment TEXT,
    public BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    changed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    duration BIGINT DEFAULT 0,
    song_count INTEGER DEFAULT 0,
    cover_art_id INTEGER
);
CREATE INDEX IF NOT EXISTS idx_playlists_user_id ON melodee_playlists (user_id);
CREATE INDEX IF NOT EXISTS idx_playlists_api_key ON melodee_playlists (api_key);
CREATE INDEX IF NOT EXISTS idx_playlists_public_where ON melodee_playlists (public) WHERE public = true;

-- Playlist Songs (Junction Table)
CREATE TABLE IF NOT EXISTS melodee_playlist_songs (
    id SERIAL PRIMARY KEY,
    playlist_id INTEGER REFERENCES melodee_playlists(id) ON DELETE CASCADE,
    song_id BIGINT REFERENCES melodee_songs(id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(playlist_id, song_id)
);
CREATE INDEX IF NOT EXISTS idx_playlist_songs_playlist_id ON melodee_playlist_songs (playlist_id);
CREATE INDEX IF NOT EXISTS idx_playlist_songs_song_id ON melodee_playlist_songs (song_id);

-- Shares Table
CREATE TABLE IF NOT EXISTS melodee_shares (
    id SERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES melodee_users(id) ON DELETE CASCADE,
    resource_type VARCHAR(50) NOT NULL,
    resource_id BIGINT NOT NULL,
    description TEXT,
    expires_at TIMESTAMP,
    visit_count INTEGER DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_shares_user_id ON melodee_shares (user_id);

-- Capacity Status Table  
CREATE TABLE IF NOT EXISTS melodee_capacity_statuses (
    id SERIAL PRIMARY KEY,
    library_id INTEGER REFERENCES melodee_libraries(id) ON DELETE CASCADE,
    total_space BIGINT NOT NULL,
    used_space BIGINT NOT NULL,
    available_space BIGINT NOT NULL,
    is_healthy BOOLEAN DEFAULT TRUE,
    last_check_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_capacity_statuses_library_id ON melodee_capacity_statuses (library_id);

-- Library Scan History
CREATE TABLE IF NOT EXISTS melodee_library_scan_histories (
    id BIGSERIAL PRIMARY KEY,
    library_id INTEGER REFERENCES melodee_libraries(id) ON DELETE CASCADE,
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
CREATE INDEX IF NOT EXISTS idx_scan_histories_library_id ON melodee_library_scan_histories (library_id);
CREATE INDEX IF NOT EXISTS idx_scan_histories_started_at ON melodee_library_scan_histories (started_at DESC);

-- Quarantine Records Table (for media processing errors)
CREATE TABLE IF NOT EXISTS melodee_quarantine_records (
    id BIGSERIAL PRIMARY KEY,
    file_path TEXT NOT NULL,
    original_path TEXT NOT NULL,
    reason TEXT NOT NULL,
    message TEXT,
    library_id INTEGER REFERENCES melodee_libraries(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_quarantine_library_id ON melodee_quarantine_records (library_id);
CREATE INDEX IF NOT EXISTS idx_quarantine_detected_at ON melodee_quarantine_records (detected_at DESC);

-- Grant all privileges on tables and sequences to melodee_user
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO melodee_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO melodee_user;
