# Melodee Admin Frontend

The Melodee Admin Frontend is a React-based administration interface that provides UI for managing the Melodee music streaming service. 

## Environment Variables

The frontend uses the following environment variables:

### Required Variables

- `REACT_APP_API_BASE_URL`: The base URL for the API backend (e.g., `http://localhost:8080/api`, `/api`)
  - Default: `/api` (relative path to be used when frontend and backend are served from the same domain)
  - Used for all Melodee API endpoints

### Optional Variables

- `REACT_APP_OPEN_SUBSONIC_ENABLED`: Whether OpenSubsonic compatibility features are enabled
  - Values: `true` or `false`
  - Default: `false`
  - When enabled, allows access to Subsonic-compatible browsing and streaming features (for third-party client support)
  - These features use `/rest` endpoints and are flagged as compatibility features, not core admin functionality

- `REACT_APP_SUBSONIC_USERNAME`: Username for OpenSubsonic authentication (when compatibility features are enabled)
  - Used for authenticating to Subsonic-compatible endpoints
  - Default: `admin`

- `REACT_APP_SUBSONIC_PASSWORD`: Encoded password for OpenSubsonic authentication
  - Note: This is sent in encoded format to Subsonic endpoints
  - Default: Encoded default password

## Frontend Architecture

The frontend is built with React and follows these architectural patterns:

### Service Layer
- `apiService.js`: Centralized API service that handles all communication with backend services
- Uses JWT authentication with automatic token refresh
- All admin operations use the Melodee API (`/api/...`) endpoints
- OpenSubsonic compatibility features (flagged as such) use `/rest/...` endpoints

### Feature Separation
1. **Admin Operations** (primary): All administrative functions use Melodee API endpoints under `/api/...`
2. **Compatibility Features** (optional): Subsonic/OpenSubsonic client support under `/rest/...` (only when enabled)

### Components
- Admin dashboard with monitoring and system health
- User management (create, update, delete users)
- Library management (scanning, processing, quarantine)
- DLQ management (dead letter queue for failed jobs)
- Settings management
- Shares management
- Playlist management

## Configuration

The frontend can be configured with environment variables as described above. Configuration is typically done via a `.env` file in the project root:

```
REACT_APP_API_BASE_URL=/api
REACT_APP_OPEN_SUBSONIC_ENABLED=true
REACT_APP_SUBSONIC_USERNAME=admin
REACT_APP_SUBSONIC_PASSWORD=your_encoded_password
```

## Subsonic Compatibility Mode

When `REACT_APP_OPEN_SUBSONIC_ENABLED` is set to `true`, the frontend enables Subsonic-compatible browsing and streaming features. These features are clearly flagged as compatibility features and are intended for:

- Supporting existing Subsonic/OpenSubsonic clients
- Providing compatibility for third-party music player apps
- Allowing existing mobile apps to connect to Melodee

These features should NOT be used for core administrative operations, which should use the Melodee API (`/api/...`) endpoints exclusively.

## API Endpoints Used

### Melodee API (Primary)
All admin functionality uses these endpoints:
- `/api/auth/...` - Authentication
- `/api/users/...` - User management
- `/api/playlists/...` - Playlist management
- `/api/admin/...` - Administrative functions
- `/api/settings` - Settings management
- `/api/shares` - Shares management
- `/api/libraries/...` - Library management
- `/api/search` - Search functionality

### OpenSubsonic API (Optional - when enabled)
Compatibility features use these endpoints:
- `/rest/getMusicFolders.view` - Get music folders
- `/rest/getArtists.view` - Get artists
- `/rest/getAlbum.view` - Get album details
- `/rest/stream.view` - Stream tracks
- `/rest/getCoverArt.view` - Get cover art

## Development

To run the frontend in development mode:

```bash
cd src/frontend
npm install
npm run dev
```

## Building for Production

```bash
cd src/frontend
npm run build
```