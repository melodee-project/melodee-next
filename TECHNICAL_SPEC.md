# Melodee - Technical Specification Document (SPEC)

## 1. Architecture Overview

### 1.1 System Architecture
The new implementation will follow a microservices architecture with the following components:
- **API Gateway**: Handles requests routing and authentication
- **Web Interface Service**: Modern web UI for library management
- **OpenSubsonic API Service**: Implements the OpenSubsonic specification
- **Media Processing Service**: Handles file conversion and metadata extraction
- **Library Management Service**: Manages library scanning and organization
- **Metadata Editing Service**: Handles in-place metadata editing and file updates
- **Streaming Service**: Manages audio streaming and transcoding
- **Job Scheduler Service**: Handles background processing jobs

### 1.2 Technology Stack
- **Backend**: Go with Fiber framework
- **Frontend**: React with TypeScript and Tailwind CSS
- **Database**: PostgreSQL for primary data storage
- **Message Queue**: Redis with Asynq for job processing
- **Caching**: Redis for in-memory caching
- **File Storage**: Local filesystem
- **Containerization**: Docker and Docker Compose

## 2. Database Schema

### 2.1 Performance-First Schema Overview

The database schema has been redesigned for massive scale operations (tens of millions of songs, 300k+ artists) with performance as the primary concern. This schema incorporates partitioning, optimized indexing, and denormalization strategies to achieve sub-200ms response times for common API operations.

### 2.2 Core Tables

#### Users Table
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

#### Libraries Table
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
    base_path VARCHAR(512) NOT NULL -- For optimized file path storage
);
```

#### Artists Table (Partitioned for Scale)
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
    song_count_cached INTEGER DEFAULT 0, -- Pre-calculated for performance
    album_count_cached INTEGER DEFAULT 0, -- Pre-calculated for performance
    duration_cached BIGINT DEFAULT 0, -- Pre-calculated for performance
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

#### Albums Table (Partitioned for Scale)
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
    song_count_cached INTEGER DEFAULT 0, -- Pre-calculated for performance
    duration_cached BIGINT, -- duration in milliseconds
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

#### Songs Table (Partitioned for Scale - Most Critical for Performance)
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

#### Playlists Table
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
    duration_cached BIGINT, -- duration in milliseconds
    song_count_cached INTEGER,
    cover_art_id INTEGER -- foreign key to images table
);
```

#### Playlist Songs Junction Table
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

### 2.3 Performance-Optimized Views and Materialized Views

#### Materialized View for Library Statistics
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

### 2.4 Performance Optimization Guidelines

The schema is designed with the following performance considerations:

1. **Massive Scale Support**: BIGSERIAL for IDs to handle 10M+ records
2. **Partitioning Strategy**: Horizontal partitioning of large tables by ID (artists) and time (albums/songs)
3. **Covering Indexes**: For common API operations to avoid table lookups
4. **Partial Indexes**: Only index active content for improved performance
5. **Denormalization**: artist_id in songs table to avoid JOINs for streaming operations
6. **Cached Aggregates**: Pre-calculated counts to avoid expensive aggregation queries
7. **Full-text Search**: Optimized search capabilities using PostgreSQL's full-text search
8. **Directory Code Integration**: Built-in support for artist directory codes for filesystem performance

## 3. API Specifications

### 3.1 OpenSubsonic API Implementation

#### Authentication
All API endpoints require authentication. Supported methods:
- **Parameters**: `u` (username), `p` (password), `t` (token), `s` (salt)
- **Header**: `Authorization: Basic <base64(username:password)>`

#### Common Response Format
```json
{
  "subsonic-response": {
    "status": "ok",
    "version": "1.16.1",
    "type": "Melodee",
    "serverVersion": "1.0.0",
    "openSubsonic": true
  }
}
```

#### Key Endpoints to Implement

**Browsing Endpoints**:
- `GET /rest/getMusicFolders.view` - Get configured music folders
- `GET /rest/getIndexes.view` - Get indexed structure of artists
- `GET /rest/getArtists.view` - Get all artists
- `GET /rest/getArtist.view` - Get details for a specific artist
- `GET /rest/getAlbumInfo.view` - Get album information
- `GET /rest/getMusicDirectory.view` - Get files in a music directory
- `GET /rest/getAlbum.view` - Get album details with songs
- `GET /rest/getSong.view` - Get song details
- `GET /rest/getGenres.view` - Get all genres

**Media Retrieval Endpoints**:
- `GET /rest/stream.view` - Stream audio files
- `GET /rest/download.view` - Download files
- `GET /rest/getCoverArt.view` - Get cover art images
- `GET /rest/getAvatar.view` - Get user avatars

**Searching Endpoints**:
- `GET /rest/search.view` - Search for artists, albums, songs
- `GET /rest/search2.view` - Enhanced search
- `GET /rest/search3.view` - More comprehensive search

**Playlist Endpoints**:
- `GET /rest/getPlaylists.view` - Get user playlists
- `GET /rest/getPlaylist.view` - Get playlist details
- `GET /rest/createPlaylist.view` - Create/edit playlists
- `GET /rest/deletePlaylist.view` - Delete playlists
- `GET /rest/updatePlaylist.view` - Update playlist details

**User Management**:
- `GET /rest/getUser.view` - Get user information
- `GET /rest/getUsers.view` - Get list of users (admin only)
- `GET /rest/createUser.view` - Create new users
- `GET /rest/updateUser.view` - Update user information
- `GET /rest/deleteUser.view` - Delete users

**System Endpoints**:
- `GET /rest/ping.view` - Test API connectivity
- `GET /rest/getLicense.view` - Get licensing information

### 3.2 Internal API Endpoints

#### Library Management API
- `POST /api/libraries/scan` - Trigger library scan
- `POST /api/libraries/process` - Process inbound files to staging
- `POST /api/libraries/move-ok` - Move OK-status albums to production
- `GET /api/libraries/stats` - Get library statistics
- `POST /api/libraries/clean` - Clean empty directories

#### Media Processing API
- `POST /api/processing/convert` - Convert media files
- `POST /api/processing/validate` - Validate media files
- `POST /api/processing/metadata` - Apply metadata rules
- `GET /api/processing/status` - Get processing status

#### Metadata API
- `GET /api/metadata/search` - Search for metadata from external sources
- `POST /api/metadata/enhance` - Enhance existing metadata
- `GET /api/metadata/images` - Search for images

#### User Management API
- `POST /api/users` - Create user
- `PUT /api/users/:id` - Update user
- `DELETE /api/users/:id` - Delete user
- `GET /api/users/:id` - Get user details

## 4. File System Organization

### 4.1 Volume Structure
```
/melodee/
├── storage/              # Processed and organized music files
├── inbound/              # New media files to be processed
├── staging/              # Media ready for manual review
├── user_images/          # User-uploaded avatars
├── playlists/            # Admin defined dynamic playlists
├── data/                 # PostgreSQL data
└── search-engine-storage/ # Local databases for metadata sources
    ├── musicbrainz/
    └── artistSearchEngine/
```

### 4.2 Album Directory Structure
```
/storage/
└── artist_name_normalized/
    └── album_name_normalized/
        ├── melodee.json          # Album metadata
        ├── song1.mp3
        ├── song2.flac
        ├── cover.jpg
        └── extra_artwork/
            ├── back_cover.jpg
            └── booklet.pdf
```

## 5. Service Specifications

### 5.1 Web Interface Service
- **Framework**: React with TypeScript
- **Styling**: Tailwind CSS with responsive design
- **State Management**: Redux Toolkit or Zustand
- **API Client**: Axios with TypeScript interfaces
- **Authentication**: JWT with refresh tokens

**Key Components**:
- Dashboard with library statistics
- Library browsing and management
- Album and artist detail views
- Metadata editing interface
- Job status monitoring
- User management interface
- Settings and configuration

### 5.2 OpenSubsonic API Service
- **Framework**: Go with Fiber framework
- **Authentication**: Middleware for OpenSubsonic authentication
- **Rate Limiting**: Per-user request limiting
- **Streaming**: High-performance file streaming with range requests
- **Transcoding**: FFmpeg-based real-time transcoding

### 5.3 Media Processing Service
- **Processing**: Audio file conversion and validation
- **Metadata Extraction**: ID3 tag reading and parsing
- **Format Support**: Support for multiple audio formats
- **Quality Control**: File validation and integrity checks

### 5.4 Directory Organization Service
- **Directory Code Generation**: Automatic creation of unique directory codes for artists
- **Template Processing**: Support for configurable directory templates with placeholders
- **Collision Resolution**: Handles duplicate directory codes by adding numerical suffixes
- **Path Resolution**: Maps between database records and actual file paths

### 5.5 Metadata Editing Service
- **Editing Interface**: Web-based metadata editing capabilities
- **File Updates**: Save metadata changes back to media files
- **Database Synchronization**: Update database records when files change
- **Validation**: Validate metadata before saving to files

### 5.6 Job Scheduler Service
- **Task Queue**: Redis-based job queue with Asynq
- **Scheduling**: Cron-like scheduling for recurring tasks
- **Monitoring**: Job status tracking and reporting
- **Workers**: Goroutine-based workers for parallel processing

**Job Types**:
- Library scanning jobs
- Directory organization jobs
- Metadata update propagation jobs
- File system monitoring jobs
- Database synchronization jobs

## 6. Security Implementation

### 6.1 Authentication
- **JWT Tokens**: JSON Web Tokens for session management
- **Password Hashing**: bcrypt for secure password storage
- **API Keys**: Unique API keys for each user and client

### 6.2 Authorization
- **Role-Based Access**: Admin vs regular user permissions
- **Resource-Level Access**: Limit access based on ownership
- **API Rate Limiting**: Prevent abuse of API endpoints

### 6.3 Data Protection
- **Input Validation**: Sanitize all user inputs
- **SQL Injection Prevention**: Use parameterized queries
- **XSS Protection**: Sanitize output for web interface
- **Secure Headers**: Implement security headers (CSP, HSTS, etc.)

## 7. Performance and Scalability

### 7.1 Caching Strategy
- **Redis Caching**: Cache frequently accessed data
- **HTTP Caching**: Implement ETags and proper cache headers
- **CDN Support**: Optional CDN for static assets

### 7.2 Database Optimization
- **Connection Pooling**: Use connection pooling for database access
- **Query Optimization**: Optimize queries with indexes and proper joins
- **Pagination**: Implement efficient pagination for large result sets
- **Database Partitioning**: Consider partitioning for large tables

### 7.3 File System Optimization
- **Efficient File Access**: Use streaming for large file operations
- **Asynchronous Processing**: Process files asynchronously
- **Concurrent Operations**: Support multiple file operations simultaneously

## 8. Deployment Architecture

### 8.1 Container Configuration
```yaml
# docker-compose.yml
version: '3.8'
services:
  web:
    build:
      context: .
      dockerfile: web/Dockerfile
    ports:
      - "8080:80"
    environment:
      - VITE_API_URL=http://localhost:3000
    depends_on:
      - api

  api:
    build:
      context: .
      dockerfile: api/Dockerfile
    ports:
      - "3000:3000"
    environment:
      - DATABASE_URL=postgresql://user:pass@db:5432/melodee
      - REDIS_URL=redis://redis:6379
    depends_on:
      - db
      - redis

  db:
    image: postgres:15
    volumes:
      - melodee_db_data:/var/lib/postgresql/data
    environment:
      - POSTGRES_DB=melodee
      - POSTGRES_USER=user
      - POSTGRES_PASSWORD=pass
      - POSTGRES_INITDB_ARGS="--encoding=UTF-8"

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data

  worker:
    build:
      context: .
      dockerfile: worker/Dockerfile
    command: /app/worker
    environment:
      - DATABASE_URL=postgresql://user:pass@db:5432/melodee
      - REDIS_URL=redis://redis:6379
    depends_on:
      - db
      - redis

volumes:
  melodee_db_data:
  redis_data:
```

### 8.2 Configuration Management
- **Environment Variables**: Use environment variables for configuration
- **Configuration Files**: Support for config file overrides
- **Environment-Specific**: Different configurations for dev/staging/prod

## 9. Monitoring and Logging

### 9.1 Application Monitoring
- **Health Checks**: Endpoints for container orchestration
- **Metrics Collection**: Collect performance metrics
- **Error Tracking**: Centralized error logging and monitoring

### 9.2 Logging Strategy
- **Structured Logging**: JSON format logs with context
- **Log Levels**: Support for different log levels
- **Log Rotation**: Automatic log rotation and archival

## 10. Migration Strategy

### 10.1 Data Migration
- **Schema Conversion**: Convert existing .NET EF models to new schema
- **Data Export/Import**: Tools to migrate existing data
- **Metadata Preservation**: Maintain all existing metadata during migration

### 10.2 API Compatibility
- **OpenSubsonic Compatibility**: Maintain 100% API compatibility
- **Client Testing**: Test with existing Subsonic clients
- **Gradual Migration**: Support for running both systems during transition