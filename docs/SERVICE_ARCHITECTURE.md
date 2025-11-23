# Service Binaries and Ports

This document describes the various services in the Melodee application, their purposes, entry points, and runtime configuration.

## Service Overview

Melodee consists of multiple services that can run as a unified application or be deployed separately:

| Service | Purpose | Entry Point | Port | Environment |
|---------|---------|-------------|------|-------------|
| Melodee API Server | Primary API for Melodee-native clients | `src/api/main.go` | 8080 (default) | Production/Development |
| OpenSubsonic Compatibility Server | Subsonic/OpenSubsonic API compatibility layer | `src/open_subsonic/main.go` | 8080 (shared) | Production/Development |
| Worker Service | Background job processing | `src/worker/main.go` | N/A (background) | Production/Development |
| Web Server | Static file serving and frontend | `src/web/main.go` | 3000 (default) | Production/Development |

## Melodee API Server (`src/api/main.go`)

### Purpose
- Handles all `/api/...` endpoints
- Provides admin functions and native client APIs
- JWT-based authentication
- User management, libraries, jobs, and system operations

### Entry Point
- File: `src/api/main.go`
- Function: `main()`
- HTTP Routes:
  - `/api/auth/*` - Authentication endpoints
  - `/api/users/*` - User management
  - `/api/playlists/*` - Playlist operations
  - `/api/libraries/*` - Library management
  - `/api/admin/*` - Administrative functions (DLQ, jobs, capacity)
  - `/api/settings/*` - Application settings
  - `/api/shares/*` - Share management
  - `/api/search/*` - Search functionality
  - `/api/images/*` - Image handling
  - `/healthz` - Health check
  - `/metrics` - Prometheus metrics

### Configuration
- Port: Configured via `SERVER_PORT` environment variable (default: 8080)
- Database: PostgreSQL via connection string
- Redis: For job queue and session storage
- JWT Secret: For authentication token signing

## OpenSubsonic Compatibility Server (`src/open_subsonic/main.go`)

### Purpose
- Provides Subsonic/OpenSubsonic API compatibility
- Handles `/rest/...` endpoints for third-party clients
- Subsonic-style authentication (username/password/token)

### Entry Point
- File: `src/open_subsonic/main.go`
- Function: Integrated in the same binary as Melodee API
- HTTP Routes:
  - `/rest/getMusicFolders.view` - Music folder listing
  - `/rest/getArtists.view` - Artist listing
  - `/rest/getAlbum.view` - Album details
  - `/rest/getSong.view` - Song details
  - `/rest/stream.view` - Audio streaming
  - `/rest/getCoverArt.view` - Cover art retrieval
  - `/rest/search3.view` - Search functionality
  - `/rest/getPlaylists.view` - Playlist operations
  - `/rest/getUser.view` - User information
  - `/rest/ping.view` - API ping/availability check

### Configuration
- Shared with Melodee API server
- Authentication tokens stored in database

## Worker Service (`src/worker/main.go`)

### Purpose
- Background job processing
- Media file processing and transcoding
- Library scanning and synchronization
- DLQ (Dead Letter Queue) handling

### Entry Point
- File: `src/worker/main.go`
- Function: `main()`
- Runs as background service
- Uses Asynq for job queue management

### Configuration
- Redis connection for job queue
- Various processing profiles and paths

## Web Server (`src/web/main.go`)

### Purpose
- Serves static files and frontend assets
- SPA routing for React frontend
- API proxy for development

### Entry Point
- File: `src/web/main.go`
- Function: `main()`
- HTTP Routes: Serves static files and proxies API calls

## Environment Configuration

### Required Environment Variables

#### Database Configuration
```
DB_HOST=localhost
DB_PORT=5432
DB_USER=melodee
DB_PASSWORD=your_password
DB_NAME=melodee
DB_SSL_MODE=disable
```

#### Redis Configuration
```
REDIS_ADDRESS=localhost:6379
```

#### JWT Configuration
```
JWT_SECRET=your_jwt_secret_key_here
```

#### Server Configuration
```
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_CORS_ALLOW_ORIGINS=http://localhost:3000,https://yourdomain.com
SERVER_CORS_ALLOW_METHODS=GET,POST,PUT,DELETE,OPTIONS
SERVER_CORS_ALLOW_HEADERS=*
```

#### Processing Configuration
```
PROCESSING_FFMPEG_PATH=/usr/bin/ffmpeg
PROCESSING_PROFILES_DEFAULT=-c:a libmp3lame -b:a 320k
```

## Development vs Production

### Development
- Services run with hot-reload capabilities
- Debug logging enabled
- CORS configured for local frontend development

### Production
- Optimized for performance
- Structured logging
- Security headers enabled
- Resource limits enforced