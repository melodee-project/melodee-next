# API Emulations for Melodee

## Overview

This document outlines the API emulation strategies for popular streaming music servers that will allow Melodee to serve as a single streaming platform while maintaining compatibility with existing clients built for other services.

The goal of these API emulation layers is to provide a unified backend that can serve multiple client ecosystems, allowing users to connect their preferred music streaming applications to Melodee without requiring separate server instances for different client types.

**Note**: Melodee already includes implementations for OpenSubsonic and Subsonic APIs, providing compatibility with a wide range of existing music streaming clients that support these standards.

## Popular Streaming Music Services

### 1. Jellyfin
- **API Type**: RESTful API
- **Authentication**: Multiple methods:
  - API key via `X-MediaBrowser-Token` header
  - JWT tokens for session management
  - User credentials via `/Users/AuthenticateByName` endpoint
- **Primary Use Case**: Media server for personal use
- **Client Support**: Web, Android, iOS, TV platforms
- **Key Endpoints**:
  - Authentication: `/Users/AuthenticateByName`, `/Sessions/Logout`
  - Media: `/Items`, `/Users/{UserId}/Items`, `/Items/{Id}`
  - System: `/System/Info`, `/System/Configuration`
  - Streaming: `/Audio/{Id}/stream`, `/Videos/{Id}/stream`
- **Response Format**: JSON
- **Authorization**: Claims-based with `AuthorizeAttribute` and various permission levels

### 2. Navidrome
- **API Type**: Subsonic API compatible (API v1.16.1)
- **Authentication**: Username/password + token-based
- **Primary Use Case**: Personal music streaming server
- **Client Support**: Any Subsonic-compatible client
- **Key Endpoints**:
  - Authentication: `/auth/login` (Navidrome native) and Subsonic auth methods
  - Media: `/rest/getMusicFolders`, `/rest/getArtists`, `/rest/getAlbumList`
  - Streaming: `/rest/stream`, `/rest/download`
  - Playlists: `/rest/getPlaylists`, `/rest/createPlaylist`
  - Search: `/rest/search2`, `/rest/search3`
- **Response Format**: XML/JSON (configurable via `f` parameter)
- **Authorization**: User sessions with tokens

### 3. Subsonic
- **API Type**: RESTful API with specific XML/JSON responses
- **Authentication**: Two methods:
  - Traditional: Username and password (clear text or hex-encoded with "enc:" prefix)
  - Token-based (API 1.13.0+): Username and token (MD5 hash of password + salt)
- **Primary Use Case**: Music streaming server (original implementation)
- **Client Support**: Wide range of dedicated clients
- **Key Authentication Parameters**:
  - `u` (required): Username
  - `p` (required*): Password or `t` (token) + `s` (salt)
  - `v` (required): Protocol version
  - `c` (required): Client identifier
  - `f` (optional): Response format ("xml", "json", "jsonp")
- **Key Endpoints**:
  - System: `/rest/ping`, `/rest/getLicense`
  - Media Browsing: `/rest/getMusicFolders`, `/rest/getArtists`, `/rest/getAlbumList`
  - Streaming: `/rest/stream`, `/rest/download`, `/rest/getCoverArt`
  - Playlists: `/rest/getPlaylists`, `/rest/createPlaylist`
  - Search: `/rest/search2`, `/rest/search3`
  - User: `/rest/getUser`, `/rest/createUser`
- **Response Format**: XML (default) or JSON
- **Error Handling**: Standardized error codes (40=wrong credentials, 50=not authorized, etc.)

### 4. Airsonic-Advanced
- **API Type**: Subsonic API compatible
- **Authentication**: Username/password + token-based (same as Subsonic)
- **Primary Use Case**: Enhanced Subsonic fork
- **Client Support**: Subsonic-compatible clients
- **Key Endpoints**: Same as Subsonic API
- **Response Format**: XML/JSON (configurable)
- **Additional Features**: More robust media organization and streaming options

### 5. Plex
- **API Type**: RESTful API
- **Authentication**: Multiple methods:
  - JWT tokens (recommended)
  - Username/password with OAuth flow
  - API tokens via Authorization header
- **Primary Use Case**: Comprehensive media server
- **Client Support**: Extensive native client ecosystem
- **Key Endpoints**:
  - Authentication: `/api/v2/auth`, `/auth#signin`
  - Media: `/library/sections`, `/library/metadata/{id}`
  - User: `/api/users`, `/api/users/{id}`
  - System: `/api/info`, `/status/sessions`
  - Streaming: Various transcode endpoints
- **Response Format**: XML/JSON (depends on endpoint)
- **Authorization**: Token-based with user permissions

## Technical Implementation Considerations

### Architecture Overview
Each API emulation layer will be implemented as a separate router/handler within Melodee that intercepts requests, translates them to internal representations, and converts responses back to the expected format of the target API.

```
[Client Request] → [API Router] → [Translator] → [Melodee Core] → [Response Builder] → [Client Response]
```

### Request/Response Mapping
Each API emulation layer needs to translate between the client's expected API format and Melodee's internal data structures. This involves:
- Mapping request parameters from the target API to Melodee's internal format
- Converting Melodee's responses to match the expected API format
- Handling authentication differently per API standard
- Maintaining metadata compatibility across systems

### Authentication Layer
Each service has its own authentication mechanism, so the emulation layers will need to:
- Implement the specific authentication method for each API
- Map external authentication to internal user management
- Handle token generation and validation according to each API's specifications
- Maintain separate authentication state for different API types

### Data Model Mapping
Different services organize their metadata differently. The emulation layers must map:
- Artist, album, track, and playlist structures
- Genre, year, and other metadata fields
- File format and codec information
- User ratings, play counts, and other dynamic data

### Implementation Patterns

#### 1. Jellyfin API Emulation
```go
// Example structure for Jellyfin API emulation
type JellyfinHandler struct {
    core *MelodeeCore
}

func (j *JellyfinHandler) AuthenticateUser(c *gin.Context) {
    // Handle Jellyfin-specific authentication
    // Convert to internal user representation
    // Return Jellyfin-formatted response
}

func (j *JellyfinHandler) GetItems(c *gin.Context) {
    // Translate Jellyfin API parameters to internal query
    // Call Melodee core
    // Format response as Jellyfin expects
}
```

#### 2. Subsonic/Navidrome API Emulation
For Subsonic-compatible APIs, we need to handle both the traditional and token-based authentication methods:

```go
type SubsonicHandler struct {
    core *MelodeeCore
}

func (s *SubsonicHandler) Authenticate(c *gin.Context) {
    // Parse authentication parameters (u, p, t, s, v, c)
    // Validate credentials
    // Return XML/JSON response as appropriate
}

func (s *SubsonicHandler) StreamMedia(c *gin.Context) {
    // Handle Subsonic streaming parameters
    // Map to internal media representation
    // Stream content with Subsonic-compatible headers
}
```

#### 3. Plex API Emulation
```go
type PlexHandler struct {
    core *MelodeeCore
}

func (p *PlexHandler) GetLibrarySections(c *gin.Context) {
    // Handle Plex authentication
    // Map internal library structure to Plex format
    // Return XML response as expected by Plex clients
}
```

### Middleware Considerations
Each API will need specific middleware to handle:
- Authentication validation
- Request parameter parsing
- Response formatting
- Error handling with API-specific error codes

### Configuration
API emulation layers should be configurable to:
- Enable/disable specific API emulations
- Customize authentication methods
- Adjust response formats
- Configure compatibility modes