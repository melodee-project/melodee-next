# Melodee - Technical Specification Document (SPEC)

**Audience:** Engineers, contributors, tech leads

**Purpose:** Canonical specification for Melodee behavior: architecture, APIs, jobs, data model, normalization, and operational contracts.

**Source of truth for:** System architecture, API contracts (internal + OpenSubsonic), background jobs, filesystem layout, and cross-cutting rules.

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

The database schema is defined in detail in `DATABASE_SCHEMA.md`.

This section intentionally stays high-level and focuses on how the schema supports the behavior in this spec (performance targets, denormalization, partitioning) rather than repeating full DDL. When updating schema details, edit `DATABASE_SCHEMA.md` and ensure any behavioral implications are reflected here.

## 3. API Specifications

### 3.1 OpenSubsonic API Implementation
Reference upstream OpenSubsonic API (Subsonic 1.16.1) documentation; fixtures maintained under `docs/fixtures/opensubsonic/`.

#### Authentication
All API endpoints require authentication. Supported methods:
- **Parameters**: `u` (username), `p` (password), `t` (token), `s` (salt)
- **Header**: `Authorization: Basic <base64(username:password)>`

**Contract Conventions**
- Pagination: `size` (default 50, max 500) and `offset` with deterministic ordering by `name_normalized`, ties by `id`.
- Errors: XML body with `status="failed"` and `error.code`/`error.message`; HTTP 200 for protocol compliance, log real HTTP status in `X-Status-Code` header.
- Dates: ISO8601 UTC.
- Strings: UTF-8, normalized NFC.
- Booleans: `true`/`false` only.

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
Example error:
```xml
<subsonic-response status="failed" version="1.16.1">
  <error code="50" message="not authorized"/>
</subsonic-response>
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

**Endpoint Coverage (contracts)**
- Browsing/search: `getIndexes`, `getArtists`, `getArtist`, `getAlbum`, `getMusicDirectory`, `getAlbumInfo`, `search`, `search2`, `search3` — require `offset/size`, normalized sorting by `name_normalized`, error codes: `70 not found`, `40 missing param`, `50 not authorized`.
- Playlists: `getPlaylists`, `getPlaylist`, `createPlaylist`, `updatePlaylist`, `deletePlaylist` — required params: `playlistId` or `name`, `songId` list. Errors: `40 missing`, `50 auth`, `70 not found`.
- Media: `stream`, `download`, `getCoverArt`, `getAvatar` — params: `id`, optional `maxBitRate`, `format`. `stream` returns HTTP 200/206 with audio; XML response only for errors (`code 70/0 general`). Fixtures added for success/error states.
- Shares: `createShare`, `getShares`, `deleteShare` — params: `id` or `ids`, `expires`, `maxDownloads`; enforce admin role for creation/deletion.
- Users: `getUsers`, `getUser`, `createUser`, `updateUser`, `deleteUser` — admin only except `getUser` self. Password rules from auth section.
- Errors: always return HTTP 200 with `<error code="" message=""/>`; also emit `X-Status-Code` for observability (e.g., 404/401).
- Cover art/avatar: when not found, return XML error with `code=70`; include `ETag` and `Last-Modified` on success responses for cacheability. For avatar uploads (internal API), enforce 2MB max JPEG/PNG.
- Avatars/Cover upload (if exposed via OpenSubsonic extensions): accept `multipart/form-data` with field `file`, `Content-Type` `image/jpeg|png`, max 2MB; success returns XML `<status>ok</status>` with `coverArt` id.

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

**Contracts**
- Auth: `POST /api/auth/login` returns `{access_token, refresh_token, expires_in, user}` (see fixtures). `POST /api/auth/refresh` requires refresh token in body/Authorization; rotates tokens and revokes old refresh.
- Playlists: CRUD endpoints mirror OpenSubsonic data model but respond JSON (`id`, `name`, `song_ids`, `public`, timestamps).
- Cover art/avatar: `GET /api/images/:id` returns binary with ETag/Last-Modified; errors as JSON.
- Shares: `POST /api/shares` (`name`, `id(s)`, `expires_at`, `max_streaming_minutes`, `allow_download`), list, delete.
- Settings/admin: endpoints must require `admin` role and return `{data, pagination}` envelopes.

### 3.3 Auth Flows (All Services)
- **Bootstrap**: On first run (users table empty) a `MELODEE_BOOTSTRAP_ADMIN_PASSWORD` env var must be set; `melodee bootstrap-admin` seeds `admin` user and rotates the var immediately.
- **Password rules**: Min 12 chars, require upper/lower/number/symbol. Enforce via validation layer; return `422` with field errors for internal API.
- **JWT**: Access 15m, refresh 14d. Store signing key from `MELODEE_JWT_SECRET`; rotate via `melodee rotate-jwt` CLI. Refresh invalidation tracked in Redis set `jwt:revoked:<jti>`.
- **API Keys**: Per-user `api_key` UUID is returned only on creation; regeneration invalidates previous and clears active sessions.
- **OpenSubsonic token mapping**: Salted token must match password hash; after validation the request is mapped to the underlying user identity and inherits the same RBAC rules.
- **RBAC**: Roles `admin`, `editor`, `user`. Admin can mutate libraries/users/settings; editor can edit metadata and staging; user is read-only streaming/search. Unauthorized → HTTP 403 internal, OpenSubsonic error `code=50`.
- **Subsonic token example**: client sends `u=user&p=enc:5f4dcc3b...&t=<md5(password+salt)>&s=<salt>`; server recomputes and validates against bcrypt hash.
- **Account lock/reset**: 5 failed logins in 15m → lock 15m; `POST /api/auth/request-reset` issues signed token emailed; `POST /api/auth/reset` sets new password (honoring password rules) and revokes sessions.

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

**Queues & Payloads**
- Queues: `critical` (stream-serving support tasks), `default` (library scans, metadata write-backs), `bulk` (large backfills), `maintenance` (partition/index management).
- Job payload shape (JSON): `{ "type": "<job_type>", "id": "<entity id or batch key>", "args": {...} }`.
- Dedup keys: `queue:type:id`. Reject duplicates while in-flight.
- Retries: expo backoff base 2s, max 5 attempts, DLQ to Redis list `asynq:dlq`.
- Timeouts: streaming-adjacent jobs 30s, scans 5m, backfills 30m.
- Required instrumentation: wrap handlers with OpenTelemetry spans and log job id, queue, attempt.

**Job Types**:
- Library scanning jobs
- Directory organization jobs
- Metadata update propagation jobs
- File system monitoring jobs
- Database synchronization jobs

**Job Payload Schemas**
- `library.scan`: `{ "library_ids":[int], "force":bool }`, dedup `library.scan:<ids>`, timeout 5m.
- `partition.create-next-month`: `{}`, dedup `partition.create-next-month`, timeout 1m.
- `metadata.writeback`: `{ "song_ids":[int] }`, dedup `metadata.writeback:<song_ids hash>`, timeout 2m.
- `directory.recalculate`: `{ "artist_ids":[int] }`, dedup `directory.recalculate:<artist_ids hash>`, timeout 2m.
- `metadata.enhance`: `{ "album_id":int, "sources":["musicbrainz","lastfm"] }`, dedup `metadata.enhance:<album_id>`, timeout 3m.
- DLQ handling: handlers emit error context; operator endpoint `POST /api/admin/jobs/requeue` accepts job id(s) and target queue; `DELETE /api/admin/jobs/dlq/:id` purges.

## 6. Security Implementation

### 6.1 Authentication
- **JWT Tokens**: JSON Web Tokens for session management
- **Password Hashing**: bcrypt for secure password storage
- **API Keys**: Unique API keys for each user and client

### 6.2 Authorization
- **Role-Based Access**: Admin vs regular user permissions
- **Resource-Level Access**: Limit access based on ownership
- **API Rate Limiting**: Prevent abuse of API endpoints

### 6.3 Rate Limiting Strategy
- **General API Endpoints**: 100 requests per 15 minutes per IP/user
- **Authentication Endpoints**: 10 requests per 5 minutes per IP (to prevent brute force)
- **Search Endpoints**: 50 requests per 10 minutes per IP/user
- **Per-User vs Per-IP**: Rate limits apply per user when authenticated, per IP when not
- **Rate Limit Response**: HTTP 429 with JSON/XML error response including retry-after information
- **Bypass Mechanism**: Admin users may have elevated rate limits or bypass in some cases

### 6.4 Data Protection
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

### 7.3 Server-Side Limits for Large Libraries
- **Maximum Page Size**: 200 items per page (for all paginated endpoints)
- **Search Result Limits**: Maximum 500 results per search request
- **Large Offset Handling**: Implement cursor-based pagination for large result sets (> 10,000 items)
- **Query Timeout**: Database queries timeout after 30 seconds to prevent hung connections
- **Memory Limits**: Search and indexing operations limited to prevent memory exhaustion
- **Concurrent Request Limits**: Per-user request limits as defined in rate limiting section
- **Database Partitioning**: Consider partitioning for large tables

### 7.4 File System Optimization
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
- **Health Checks**: `/healthz` returns `{status, db, redis}`; fail if DB ping >200ms or Redis ping >100ms.
- **Metrics Collection**: Required Prometheus metrics:
  - `melodee_stream_requests_total{status,format}`
  - `melodee_stream_bytes_total`
  - `melodee_job_duration_seconds{queue,type,status}`
  - `melodee_metadata_drift_total`
  - `melodee_quarantine_total{reason}`
- **Tracing**: OpenTelemetry spans for `stream`, `metadata.writeback`, `library.scan` with baggage `user_id`, `song_id`, `job_id`.
- **Error Tracking**: Centralized error logging and monitoring

### 9.2 Logging Strategy
- **Structured Logging**: JSON format logs with context
- **Log Levels**: Support for different log levels
- **Log Rotation**: Automatic log rotation and archival
- **Required fields**: `ts`, `level`, `msg`, `req_id`, `user_id`, `ip`, `route`, `status`, `duration_ms`, `queue`, `job_type`, `attempt`.

## 10. Migration Strategy

### 10.1 Data Migration
- **Schema Conversion**: Convert existing .NET EF models to new schema
- **Data Export/Import**: Tools to migrate existing data
- **Metadata Preservation**: Maintain all existing metadata during migration

### 10.2 API Compatibility
- **OpenSubsonic Compatibility**: Maintain 100% API compatibility
- **Client Testing**: Test with existing Subsonic clients
- **Gradual Migration**: Support for running both systems during transition

## 11. Sorting and Normalization Rules
- Collation: normalize to lowercase ASCII (fold diacritics) for `name_normalized` fields; ordering uses `name_normalized` then `id`.
- Articles to ignore for sort: `the`, `a`, `an`, `le`, `la`, `les`, `el`, `los`, `las`; do not drop from display.
- Albums/songs: sort by `album.sort_name` then `song.sort_order` then `song.name_normalized`; ensure UI/API consistency.
- Search normalization: fold diacritics, strip punctuation, collapse whitespace; tokens matched with trigram + full-text per DB schema.
