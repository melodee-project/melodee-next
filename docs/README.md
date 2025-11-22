# Melodee System Documentation

## Table of Contents
1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Installation](#installation)
4. [Configuration](#configuration)
5. [API Endpoints](#api-endpoints)
6. [Deployment](#deployment)
7. [Monitoring & Health](#monitoring--health)
8. [Capacity Management](#capacity-management)
9. [Troubleshooting](#troubleshooting)
10. [FAQ](#faq)

## Overview

Melodee is a high-performance music streaming server that implements the OpenSubsonic API specification. It provides a robust platform for organizing, streaming, and sharing music libraries at scale.

### Key Features
- Compatible with Subsonic/OpenSubsonic clients
- Advanced library organization with artist directory codes
- Media transcoding and streaming
- User management and sharing capabilities
- Admin dashboard and monitoring
- Scalable architecture with job processing queues

## Architecture

The Melodee system consists of several interconnected services:

### Core Services
- **API Gateway**: Handles OpenSubsonic and internal API requests
- **Worker Service**: Processes background jobs for scanning, processing, and promoting media files
- **Media Service**: Handles file processing, transcoding, and metadata extraction
- **Authentication Service**: Manages user authentication and authorization

### Data Stores
- **PostgreSQL**: Primary database for metadata
- **Redis**: Job queue and caching
- **File System**: Media file storage organized by artist directory codes

### Job Processing Architecture
The system uses Asynq for job processing with the following workflow:
1. **Inbound**: Newly added files are placed in inbound directories
2. **Staging**: Files are processed and organized with directory codes
3. **Production**: Approved files are moved to production libraries

## Installation

### Prerequisites
- Docker and Docker Compose
- At least 4GB RAM
- Sufficient disk space for media files
- FFmpeg installed for transcoding

### Quick Start
```bash
# Clone the repository
git clone https://github.com/your-org/melodee.git

# Navigate to the project directory
cd melodee

# Create environment file
cp .env.example .env
# Edit .env with your settings

# Start services
docker-compose up -d

# Visit the admin dashboard
open http://localhost:3000
```

## Configuration

### Environment Variables
The system is configured using environment variables in the `.env` file:

```bash
# Database configuration
MELODEE_DATABASE_HOST=localhost
MELODEE_DATABASE_PORT=5432
MELODEE_DATABASE_USER=melodee_user
MELODEE_DATABASE_PASSWORD=your_secure_password
MELODEE_DATABASE_DBNAME=melodee
MELODEE_DATABASE_SSLMODE=disable

# Redis configuration
MELODEE_REDIS_ADDRESS=localhost:6379
MELODEE_REDIS_PASSWORD=

# JWT configuration
MELODEE_JWT_SECRET=your_very_secure_jwt_secret_here

# FFmpeg configuration
FFMPEG_PATH=/usr/bin/ffmpeg

# Storage paths
STORAGE_MOUNT=/path/to/your/music/files
INBOUND_MOUNT=/path/to/inbound/files
STAGING_MOUNT=/path/to/staging/files
QUARANTINE_MOUNT=/path/to/quarantine/files
USER_IMAGES_MOUNT=/path/to/user/images
```

### Directory Code Configuration
Artist directory codes follow a standardized format to ensure efficient filesystem organization:
- Format: Consonant-vowel pattern from artist name (e.g., "Led Zeppelin" → "LZ")
- Length: 2-8 characters
- Collisions are handled with numeric suffixes (e.g., "LZ-2", "LZ-3")

## API Endpoints

### Public Endpoints (OpenSubsonic Compatible)
- `GET /rest/ping.view` - Health check
- `GET /rest/getLicense.view` - License information
- `GET /rest/getMusicFolders.view` - Available libraries
- `GET /rest/getIndexes.view` - Artist indexes organized by directory code
- `GET /rest/getArtists.view` - All artists
- `GET /rest/getArtist.view` - Artist details
- `GET /rest/getAlbum.view` - Album details
- `GET /rest/getMusicDirectory.view` - Directory contents
- `GET /rest/stream.view` - Stream media files
- `GET /rest/download.view` - Download media files
- `GET /rest/getCoverArt.view` - Album cover art
- `GET /rest/getAvatar.view` - User avatars
- `GET /rest/search.view`, `search2.view`, `search3.view` - Search functionality
- `GET /rest/getPlaylists.view`, `getPlaylist.view` - Playlist management

### Internal Endpoints
- `POST /api/auth/login` - User login
- `POST /api/auth/refresh` - Token refresh
- `POST /api/auth/request-reset` - Password reset request
- `POST /api/auth/reset` - Password reset
- `GET /api/libraries` - Library management
- `POST /api/libraries/scan` - Initiate library scan
- `POST /api/libraries/process` - Process inbound files to staging
- `POST /api/libraries/move-ok` - Promote staging files to production
- `GET /api/admin/jobs/dlq` - Dead letter queue items
- `POST /api/admin/jobs/requeue` - Requeue DLQ items
- `POST /api/admin/jobs/purge` - Purge DLQ items
- `GET /api/settings` - System settings
- `PUT /api/settings` - Update settings
- `GET /api/users` - User management
- `POST /api/users` - Create user
- `PUT /api/users/:id` - Update user
- `DELETE /api/users/:id` - Delete user
- `GET /api/shares` - Share management
- `POST /api/shares` - Create share
- `PUT /api/shares/:id` - Update share
- `DELETE /api/shares/:id` - Delete share

## Deployment

### Docker Compose Deployment
For production deployment, use the production compose file:

```bash
# Create production environment
docker-compose -f docker-compose.prod.yml up -d
```

### Kubernetes Deployment
TODO: Add Helm chart and Kubernetes manifests

### Service Scaling
The system can be scaled horizontally:

```yaml
# Example scaling configuration
services:
  api:
    replicas: 3
    resources:
      limits:
        cpus: '0.5'
        memory: 512M
      reservations:
        cpus: '0.25'
        memory: 256M

  worker:
    replicas: 2
    resources:
      limits:
        cpus: '1.0'
        memory: 1G
      reservations:
        cpus: '0.5'
        memory: 512M
```

## Monitoring & Health

### Health Check Endpoints
- `GET /healthz` - General health check
- `GET /metrics` - Prometheus metrics endpoint

### Health Check Response Format
```json
{
  "status": "ok",
  "db": {
    "status": "ok",
    "latency_ms": 15
  },
  "redis": {
    "status": "ok",
    "latency_ms": 8
  }
}
```

### Monitoring Dashboard
Access Grafana monitoring at `http://localhost:3001` with credentials from your environment file.

### Alerting Rules
The system includes predefined alerting rules for:
- Database connectivity issues
- High capacity usage (warning at 80%, critical at 90%)
- Dead letter queue items accumulating
- High error rates
- Slow response times

## Capacity Management

### Directory Organization
The system uses artist directory codes to efficiently organize large music libraries:
- Artists are assigned 2-8 character directory codes
- Files are stored in hierarchical structure using directory codes
- This prevents filesystem performance issues with deep directories

### Path Templates
The system supports configurable path templates:
```
{artist_dir_code}/{artist}/{year} - {album}
```

### Capacity Probes
The system automatically monitors storage capacity:
- Default check every 10 minutes
- Warning at 80% usage
- Critical alert at 90% usage
- Quarantine mechanism for full disks

## Troubleshooting

### Common Issues

#### Database Connection Issues
1. Check database service status:
   ```bash
   docker-compose logs db
   ```
2. Verify database configuration in environment variables
3. Ensure PostgreSQL is listening on the configured port

#### Job Processing Issues
1. Check the dead letter queue:
   - Visit the admin DLQ management page
   - Look for any failed jobs
2. Review worker logs:
   ```bash
   docker-compose logs worker
   ```

#### Transcoding Issues
1. Ensure FFmpeg is properly installed and in PATH:
   ```bash
   which ffmpeg
   ffmpeg -version
   ```
2. Check worker service logs for transcoding errors

#### Slow Performance
1. Check database performance:
   - Ensure proper indexing
   - Monitor connection pool usage
2. Check Redis performance:
   - Ensure sufficient memory allocation
   - Monitor connection count

### Log Files
- API logs: `/var/log/melodee/api.log`
- Worker logs: `/var/log/melodee/worker.log`
- Database logs: `/var/log/postgresql/`

### Debug Mode
To enable debug logging, set the LOG_LEVEL environment variable to "debug":

```bash
LOG_LEVEL=debug docker-compose up
```

## FAQ

Q: How do I add music to my Melodee library?
A: You can add music by placing files in the inbound directory specified in your configuration. The system will automatically scan and process these files through the inbound → staging → production workflow.

Q: What file formats are supported?
A: The system supports all formats that FFmpeg supports for transcoding, including MP3, FLAC, OGG, OPUS, M4A, and many others.

Q: Can I access my Melodee instance from mobile apps?
A: Yes! Melodee implements the OpenSubsonic specification, so it's compatible with all Subsonic-compatible mobile apps like DSub, Subsonic, and others.

Q: How does the directory code system work?
A: The system generates efficient directory codes for artists (e.g., "Led Zeppelin" → "LZ") to prevent deep directory structures that can cause performance issues with large libraries.

Q: What are the system requirements?
A: Minimum 4GB RAM, sufficient disk space for your music library, and a modern CPU for transcoding. For large libraries, consider SSD storage for better performance.

## Contact

For support, please open an issue on the GitHub repository or contact the development team.