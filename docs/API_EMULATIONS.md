# API Emulations for Melodee

## Overview

This document outlines the API emulation strategies for popular streaming music servers that will allow Melodee to serve as a single streaming platform while maintaining compatibility with existing clients built for other services.

The goal of these API emulation layers is to provide a unified backend that can serve multiple client ecosystems, allowing users to connect their preferred music streaming applications to Melodee without requiring separate server instances for different client types.

### Scope & Non‑Goals
- Focus on Subsonic/OpenSubsonic full compatibility first.
- Jellyfin: start with auth and basic item browsing/streaming.
- Plex: do not implement Plex.tv claim/association or GDM (UDP discovery) in the initial phase; keep experimentation behind feature flags.

## Implemented Standards

### 1. Subsonic & OpenSubsonic
*Includes compatibility for: Navidrome, Airsonic-Advanced, Gonic, and original Subsonic clients.*

Melodee currently implements the OpenSubsonic API specification, which is a superset of the original Subsonic API. This provides immediate compatibility with a vast ecosystem of existing clients (DSub, Symfonium, Ultrasonic, etc.).

- **API Version**: Emulates Subsonic v1.16.1 + OpenSubsonic Extensions
- **Authentication**:
  - Required params: `u` (user), `v` (API version), `c` (client name)
  - Optional params: `p` (password or `enc:`-prefixed hex), or `t` (token) + `s` (salt)
  - Supported methods:
    - **Legacy**: Username and password (plaintext or hex-encoded with `enc:` prefix)
    - **Token-based**: Username and token (MD5 of password + salt)
- **Primary Use Case**: Personal music streaming with offline caching support
- **Key Endpoints**:
  - System: `/rest/ping`, `/rest/getLicense`, `/rest/getOpenSubsonicExtensions`
  - Browsing: `/rest/getMusicFolders`, `/rest/getIndexes`, `/rest/getArtist`, `/rest/getAlbum`
  - Streaming: `/rest/stream`, `/rest/download`, `/rest/getCoverArt`
  - Lists: `/rest/getAlbumList2`, `/rest/getRandomSongs`, `/rest/getNowPlaying`
  - User: `/rest/getUser`, `/rest/scrobble`, `/rest/star`
- **Response Format**: XML (default for many clients) and JSON.
  - Clients can request JSON with `f=json` or `Accept: application/json`.
- **Status**: **Active** (See `docs/OPENSUBSONIC_IMPLEMENTATION_PLAN.md` for detailed coverage)

References: `docs/opensubsonic-v1.16.1-openapi.yaml`, `docs/subsonic-v1.16.1-openapi.yaml`.

#### Navidrome Nuances (within Subsonic/OpenSubsonic)
Navidrome aims to fully support Subsonic 1.16.1 and adopts OpenSubsonic extensions, but it introduces a few behavioral differences and extras on existing endpoints. These are not new endpoints, but deviations or additional parameters/fields:
- **IDs**: Always strings (MD5/UUID). Some clients incorrectly cast IDs to integers; avoid this.
- **Scan endpoints**: `getScanStatus` includes extra fields `lastScan` and `folderCount`. `startScan` accepts an extra `fullScan` boolean to force a full rescan.
- **Download**: `/rest/download` accepts IDs for Songs, Albums, Artists and Playlists, and accepts transcoding options similar to `/rest/stream`.
- **Users**: `getUsers` returns only the authenticated user. `getUser` ignores the `username` param and returns the authenticated user; roles reflect actual server capabilities (e.g., download/jukebox depends on server settings).
- **Search**: `search2`/`search3` do not support Lucene; only simple autocomplete queries are supported.
- **Play tracking**: `stream` does not mark as played; only `scrobble` with `submission=true` does.
- **Browse-by-folder**: Not implemented; directory endpoints simulate a tree (e.g., `/Artist/Album/01 - Song.mp3`).
- **Video**: Not implemented (music-only focus).
- **PlayQueue**: In `getPlayQueue`, the `current` field is a string ID (not int as some clients assume).

For an up-to-date list of OpenSubsonic extensions supported by Navidrome, see: https://github.com/navidrome/navidrome/issues/2695. Navidrome’s Subsonic compatibility page: https://www.navidrome.org/docs/developers/subsonic-api/

## Planned Emulations

### 1. Jellyfin
- **API Type**: RESTful API
- **Authentication**: Multiple methods:
  - API token via `X-MediaBrowser-Token` or `X-Emby-Token` header
  - Client identity via `X-Emby-Authorization` (device, version, client)
  - JWT tokens for session management
  - User credentials via `/Users/AuthenticateByName` endpoint
- **Primary Use Case**: Media server for personal use (Video & Audio)
- **Client Support**: Web, Android, iOS, TV platforms (Roku, Android TV)
- **Key Endpoints**:
  - Authentication: `/Users/AuthenticateByName`, `/Sessions/Logout`
  - Media: `/Items`, `/Users/{UserId}/Items`, `/Items/{Id}`
  - System: `/System/Info`, `/System/Configuration`
  - Streaming: `/Audio/{Id}/stream`, `/Videos/{Id}/stream`
- **Response Format**: JSON
- **Authorization**: Claims-based with `AuthorizeAttribute` and various permission levels
 - **Headers Example**:
   - `X-Emby-Authorization: MediaBrowser Client="Melodee", Device="Linux", DeviceId="<id>", Version="<ver>"`
   - `X-Emby-Token: <token>`

### 2. Plex
*Note: Plex emulation is significantly more complex than REST-based alternatives due to its proprietary nature, strict XML schemas, and discovery protocols (GDM).*

- **API Type**: RESTful API (Internal/Private)
- **Authentication**:
  - `X-Plex-Token` header (typically acquired via Plex.tv or local auth)
- **Primary Use Case**: Comprehensive media server with centralized auth
- **Client Support**: Extensive native client ecosystem (Smart TVs, Consoles, Mobile)
- **Key Endpoints (common local server routes)**:
  - Media library: `/library/sections`, `/library/metadata/{id}`
  - Sessions: `/status/sessions`
  - Transcode: `/video/:/transcode/universal/start` (parameters vary)
- **Discovery/Auth Caveats**:
  - GDM (UDP discovery) and Plex.tv claim/association are not planned for phase 1.
  - Some clients require the server to be claimed; treat Plex emulation as experimental behind a feature flag.
- **Response Format**: XML (primary) / JSON
- **Challenges**:
  - **GDM (G'Day Mate)**: Plex's UDP discovery protocol.
  - **Plex.tv Association**: Clients often expect the server to be "claimed" by a Plex account.
  - **XML Schema**: The response structure is deeply nested and strict.

## Other Protocols (Consideration)

### 1. DLNA / UPnP
While not a "client API" in the traditional sense, DLNA support is critical for hardware compatibility.

- **Primary Use Case**: Streaming to "dumb" receivers, Hi-Fi equipment, Smart TVs, and Sonos (via UPnP).
- **Components**:
  - **Content Directory Service (CDS)**: Exposes the library hierarchy (Artists -> Albums -> Tracks).
  - **Connection Manager**: Negotiates protocols and formats.
  - **Media Renderer**: (Optional) Allows Melodee to control playback on other devices.
- **Format**: SOAP-based XML.

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

### Discovery
Client discovery expectations differ by ecosystem:
- **Subsonic/OpenSubsonic**: No special discovery; clients use a configured base URL.
- **Jellyfin**: Some clients support network discovery via SSDP/UPnP, but base URL entry is common.
- **Plex**: GDM (UDP) discovery and Plex.tv association are common; both out of scope for initial phase.

### ID Stability & Mapping
**Critical Challenge**: Different clients expect different ID formats.
- **Melodee**: Uses UUIDs or Integer IDs internally.
- **Subsonic**: Often expects string IDs (can be paths or hashes).
- **Plex**: Expects Integer IDs.
- **Jellyfin**: Expects UUIDs.

*Strategy*: Treat IDs as opaque strings in responses and maintain stable, per-emulation mappings. If an emulation requires integers, use a persistent mapping table or deterministic hash (with collision monitoring) to ensure `Song A` always has `ID X` for that emulation.

### Versioning & Paths
- **Subsonic/OpenSubsonic**: Routes under `/rest/...`.
- **Jellyfin**: Mirror official paths (`/Users`, `/Items`, etc.) under a dedicated prefix (e.g., `/jellyfin`) to avoid collisions.
- **Plex**: Mirror common paths under a feature-flagged prefix (e.g., `/plex`).

### Streaming Parameters (Subsonic)
Support and document key query params for `/rest/stream`:
- `format`, `maxBitRate`, `timeOffset`, `size`, `estimateContentLength`, `converted`, `transcoding` behavior.
- Content headers (e.g., `Content-Type`, `Content-Length` when known), byte-range support, and caching directives.

### Authentication Layer
Each service has its own authentication mechanism, so the emulation layers will need to:
- Implement the specific authentication method for each API
- Map external authentication to internal user management
- Handle token generation and validation according to each API's specifications
- Maintain separate authentication state for different API types

### Error Model Mapping
- **Subsonic**: Map internal errors to standard error codes (e.g., `10` auth failed, `40` not found). Preserve the `status="ok"|"failed"` envelope.
- **Jellyfin**: Use appropriate HTTP status codes with JSON problem details where applicable.
- **Plex**: Return expected XML structures and status codes; include error attributes consistently.

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

### Middleware Considerations
Each API will need specific middleware to handle:
- Authentication validation
- Request parameter parsing
- Response formatting
- Error handling with API-specific error codes

### Security & Logging
- Redact sensitive query parameters (Subsonic `p`, `t`, `s`) in access logs.
- CORS: Align with client expectations per emulation; consider opt-in domain restrictions.
- Rate limiting: Optional, configurable per emulation.

### Configuration
API emulation layers should be configurable to:
- Enable/disable specific API emulations
- Customize authentication methods
- Adjust response formats
- Configure compatibility modes

Example `config.yaml` excerpt:

```yaml
emulations:
  subsonic:
    enabled: true
    default_format: xml   # xml|json
    compatibility_modes:
      symfonium: true
      dsub: true
  jellyfin:
    enabled: false
    prefix: "/jellyfin"
  plex:
    enabled: false
    prefix: "/plex"
    experimental: true
```

### Testing Approach
- OpenSubsonic: Contract tests against `opensubsonic-v1.16.1-openapi.yaml`; client-flow tests for DSub/Symfonium happy paths.
- Jellyfin: Start with `/Users/AuthenticateByName`, `/Users/{UserId}/Items`, `/Items/{Id}` using fixtures.
- Use the Manual SQLite Schema test pattern to avoid Postgres/SQLite conflicts.

### Compatibility Modes
- Provide toggles for known client quirks (e.g., star/rating field expectations, index date handling) per emulation.

### Compatibility Matrix (initial)

| Client       | Emulation           | Browse | Stream | Star/Rate | Notes |
|--------------|---------------------|--------|--------|-----------|-------|
| DSub         | OpenSubsonic        | Yes    | Yes    | Yes       | XML default; `f=json` works |
| Symfonium    | OpenSubsonic        | Yes    | Yes    | Yes       | Prefers JSON; large lists pagination |
| Ultrasonic   | OpenSubsonic        | Yes    | Yes    | Partial   | Some endpoints optional |
| Jellyfin Web | Jellyfin (planned)  | WIP    | WIP    | N/A       | Start with auth + items |
| Plex Mobile  | Plex (experimental) | No     | No     | N/A       | Requires claim/discovery |

### Licensing / Disclaimer
Melodee is not affiliated with Plex, Jellyfin, Emby, Subsonic, or related projects. Emulations are provided solely for compatibility with existing clients.
