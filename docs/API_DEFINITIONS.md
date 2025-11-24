# Overview

Melodee exposes two APIs that share the same data and capabilities but serve very different audiences and compatibility requirements:
- **Melodee API** – the preferred, first-class API for Melodee-native clients.
- **Subsonic/OpenSubsonic API** – a compatibility layer for third‑party Subsonic/OpenSubsonic clients.

At a high level:
- If you are building a **new Melodee‑aware client or service**, use the **Melodee API**.
- If you are **integrating an existing Subsonic/OpenSubsonic client**, use the **Subsonic/OpenSubsonic API**.

The sections below describe each API in more detail and when to choose one over the other.

## Melodee API

The Melodee API is the primary API for Melodee. It is designed for modern clients that want:
- predictable, required pagination on list operations,
- consistent, typed models (Artists, Albums, Songs, Playlists, etc.), and
- access to newer Melodee features that have no equivalent in the Subsonic/OpenSubsonic spec.

This matters especially for very large collections, where a lack of proper pagination in legacy Subsonic clients can cause severe performance issues.

- **Base path**: `https://<host>/api` (versioned per endpoint, for example `https://<host>/api/v1/{endpoint}` or `https://<host>/api/v2/{endpoint}` depending on the API version being called)
- **Authentication**: JWT bearer token obtained from `POST /api/v1/Users/authenticate` (see `melodee-v1-openapi.yaml`).
- **Schema**: `docs/melodee-v1-openapi.yaml` is the source of truth for endpoints, request/response models, and pagination metadata.
- **Primary clients**: Melodee Admin, Melodee‑Amp, Melodee‑Droid, and future native/first‑party applications.
- **Notable capabilities**: first‑class pagination on every list endpoint, typed models for Artists/Albums/Songs/Playlists, richer search APIs, system stats/info, scrobbling, and per‑user actions (ratings, starred state).

**When to use it**

Use the Melodee API when:
- you control the client and can adopt Melodee‑specific concepts and pagination;
- you are building admin or operational tooling (user management, libraries, jobs, health, metrics);
- you need features that are not present in the Subsonic/OpenSubsonic spec (for example, richer metadata or server stats);
- you want a stable, versioned contract (`/api/v1`) with explicit OpenAPI documentation.

When updating or adding endpoints in this namespace, keep the OpenAPI document in sync and consider adding fixtures under `docs/fixtures/internal` for new request/response examples.

### Versioning strategy

The Melodee API uses explicit version segments in the path:
- **v1**: `/api/v1/{endpoint}`
- **v2**: `/api/v2/{endpoint}` (and so on for future major versions)

Guidelines:
- Breaking changes (such as removing fields, changing response shapes, or altering semantics) are introduced only in a new major version (`v2`, `v3`, ...), not within an existing one.
- Non‑breaking, backwards‑compatible changes (adding optional fields, expanding enum values, adding new endpoints) may be added within an existing version.
- The OpenAPI document for each version (for example, `melodee-v1-openapi.yaml`) is the contract for that version and should not change in a way that breaks existing clients.

## Subsonic/OpenSubsonic API

The Subsonic/OpenSubsonic API exists to support the ecosystem of existing Subsonic/OpenSubsonic clients. It presents Melodee as a standards‑compliant Subsonic/OpenSubsonic server so that those clients can connect without changes.

> This API is generated from the official OpenSubsonic OpenAPI specification for version X.Y. Implementations that conform to this version of the specification should be broadly compatible. Some servers or clients may not implement all optional endpoints or extensions, so callers should be prepared to handle missing endpoints and extra fields gracefully.

- **Base path**: `https://<host>/rest`
- **Authentication**: Subsonic token or salted password per the OpenSubsonic spec.
- **Schema reference**: `docs/subsonic-openapi.yaml` (canonical Subsonic schema) and `docs/opensubsonic-openapi.yaml` (OpenSubsonic superset generated from https://opensubsonic.netlify.app/docs/openapi/openapi.json). Keep both aligned with the corresponding upstream specs and mirror any changes in `src/open_subsonic` contract tests.
- **Primary clients**: third‑party Subsonic/OpenSubsonic apps.
- **Notable capabilities**: parity with the canonical Subsonic feature set (streaming, cover art, playlist management) required for external client compatibility. Pagination is limited to what the Subsonic spec provides and may not perform well on very large libraries.

**When to use it**

Use the Subsonic/OpenSubsonic API when:
- you are using an existing Subsonic/OpenSubsonic client and cannot modify it to speak the Melodee API;
- you need drop‑in compatibility with the standard Subsonic/OpenSubsonic endpoints for testing or migration purposes;
- you are validating Melodee against the upstream Subsonic/OpenSubsonic specifications or contract tests.

When we add or remove functionality here, we should mirror the change in the contract tests and, if applicable, the upstream spec documentation referenced by our release notes.

## Feature comparison

| Aspect | Melodee API | Subsonic/OpenSubsonic API |
| --- | --- | --- |
| Primary purpose | First‑party, modern Melodee clients | Compatibility with existing Subsonic/OpenSubsonic clients |
| Base path | `/api` | `/rest` |
| Auth | JWT bearer via `/Users/authenticate` | Subsonic token/password |
| Versioning | Versioned path segments (for example: `/api/v1/{endpoint}`, `/api/v2/{endpoint}`) | Version implied by Subsonic/OpenSubsonic spec used |
| Pagination | Mandatory on list endpoints with metadata (`PaginationMetadata`) | Limited / offset‑based per Subsonic spec |
| Schema reference | `docs/melodee-v1-openapi.yaml` | `docs/subsonic-openapi.yaml`, `docs/opensubsonic-openapi.yaml`, plus `src/open_subsonic` tests |
| Primary clients | Melodee‑native apps, Admin UI | Third‑party Subsonic/OpenSubsonic clients |
| Unique endpoints | System info, scrobble, server stats, user ratings/stars, richer search | Legacy Subsonic methods (e.g., `getIndexes.view`, `stream.view`) |

## Choosing an API for new work

For new features or integrations, default to the **Melodee API**. Only add new behavior to the Subsonic/OpenSubsonic API when it is required for:
- maintaining compatibility with existing clients, or
- matching changes in the upstream Subsonic/OpenSubsonic specifications.

If a capability must be available to both Melodee‑native and Subsonic/OpenSubsonic clients, add it to the Melodee API first, then provide a best‑effort mapping into the Subsonic/OpenSubsonic API where the legacy model allows.

## High‑level architecture

The two APIs sit side‑by‑side in front of the same core services and data store. They differ only in protocol, models, and compatibility requirements.

```mermaid
flowchart LR
	subgraph Native[Melodee‑native clients]
		A[Admin UI]
		B[Melodee‑Amp]
		C[Melodee‑Droid]
		D[Future native apps]
	end

	subgraph SubsonicClients[Subsonic/OpenSubsonic clients]
		E[Existing mobile/desktop/web apps]
	end

	subgraph APIs[API layer]
		M["Melodee API\n(/api/v1, JWT auth)"]
		S["Subsonic/OpenSubsonic API\n(/rest, Subsonic auth)"]
	end

	subgraph Core[Melodee core services]
		F["Library, metadata, jobs, streams"]
	end

	G[("Storage\n(DB, media)")]

	A --> M
	B --> M
	C --> M
	D --> M
	E --> S

	M --> F
	S --> F
	F --> G
```

## Routing overview

Melodee implements two distinct API servers in the same binary:

- **Melodee API server** (`src/api/main.go`): Handles endpoints under `/api/...` with JWT authentication, serving admin functions and native clients.
- **OpenSubsonic compatibility server** (`src/open_subsonic/main.go`): Handles endpoints under `/rest/...` with Subsonic-style authentication, providing compatibility with existing Subsonic clients.

Both servers share the same underlying data and services but expose different protocol and model contracts.

## Deviations from upstream Subsonic/OpenSubsonic spec

Melodee intentionally deviates from the upstream Subsonic/OpenSubsonic specifications in the following ways:

- **Pagination**: Melodee enforces server-side limits and provides consistent pagination metadata, unlike Subsonic which allows clients to request unlimited results.
- **Error responses**: Melodee returns structured JSON error responses, while Subsonic uses XML with error codes.
- **Authentication**: Extended JWT token support in addition to standard Subsonic authentication methods.
- **Job queue management**: Additional administration endpoints for managing the job queue (DLQ, requeue, purge operations) not present in the standard Subsonic spec.

## Authentication Examples

### Melodee API Authentication
To authenticate with the Melodee API, first obtain a JWT token:

```bash
curl -X POST https://your-melodee-instance.com/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "your-username",
    "password": "your-password"
  }'
```

Response:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "def50200...",
  "expires_in": 900,
  "user": {
    "id": "user-123",
    "username": "admin",
    "is_admin": true
  }
}
```

Then use the token in subsequent requests:
```bash
curl -X GET https://your-melodee-instance.com/api/users \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### Subsonic/OpenSubsonic Authentication
Subsonic API uses token or password-salt authentication:
```bash
curl "https://your-melodee-instance.com/rest/getMusicFolders.view?u=username&p=enc:hashed_password&t=token&s=salt&v=1.16.1&c=melodee"
```

## Pagination Examples

### Melodee API Pagination
All list endpoints in the Melodee API return consistent pagination metadata:

```bash
curl "https://your-melodee-instance.com/api/users?page=1&limit=10"
```

Response:
```json
{
  "data": [
    {
      "id": 1,
      "username": "user1",
      "email": "user1@example.com",
      "is_admin": false
    }
  ],
  "pagination": {
    "page": 1,
    "size": 10,
    "total": 150
  }
}
```

### Subsonic/OpenSubsonic Pagination
Subsonic API uses offset-based pagination:
```bash
curl "https://your-melodee-instance.com/rest/getArtists.view?u=username&p=enc:password&v=1.16.1&c=melodee&musicFolderId=1&offset=0&size=20"
```

## Error Response Examples

### Melodee API Error Responses
Melodee API returns structured JSON error responses:

```json
{
  "error": "Invalid credentials",
  "code": "AUTH_INVALID_CREDENTIALS",
  "details": {
    "field": "password",
    "reason": "Username or password is incorrect"
  }
}
```

### Subsonic/OpenSubsonic Error Responses
Subsonic API returns XML error responses:
```xml
<subsonic-response xmlns="http://subsonic.org/restapi" status="failed" version="1.16.1">
  <error code="40" message="Wrong username or password"/>
</subsonic-response>
```

## Common Operations Examples

### Create Playlist (Melodee API)
```bash
curl -X POST https://your-melodee-instance.com/api/playlists \
  -H "Authorization: Bearer JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My New Playlist",
    "comment": "A collection of favorite tracks",
    "public": false,
    "song_ids": [1, 5, 10]
  }'
```

### Search Library (Melodee API)
```bash
curl "https://your-melodee-instance.com/api/search?q=artist&type=artist" \
  -H "Authorization: Bearer JWT_TOKEN"
```

### Stream Track (Subsonic API)
```bash
curl "https://your-melodee-instance.com/rest/stream.view?u=username&p=enc:password&id=123&v=1.16.1&c=melodee"
```

### Admin API Usage
The Admin UI uses the Melodee API (`/api/...`) for all administrative operations; OpenSubsonic (`/rest/...`) is reserved for compatibility with external clients.

The frontend service layer clearly separates admin functionality from compatibility features:
- **Admin operations**: All administrative functions (users, libraries, DLQ, shares, settings, etc.) use Melodee API endpoints under `/api/...`
- **Compatibility helpers**: Any remaining `mediaService` endpoints that access `/rest/...` are clearly delineated as "Subsonic compatibility" features and are intended only for compatibility with external clients

## Related documentation

- Internal admin‑focused routes (user management, libraries, jobs, etc.) live under `/api/...` and are cataloged in `docs/INTERNAL_API_ROUTES.md`.

## Known limitations

### Pagination and scalability
- Melodee API enforces server-side limits with a maximum page size of 200 for list endpoints
- Pagination uses offset/limit for most endpoints; cursor-based pagination is not implemented
- Search endpoints enforce maximum 500 results per request to prevent performance issues with large datasets
- Very large offset values (over 100,000) may result in slower queries due to offset-based pagination

### Performance
- Rate limiting applies to expensive operations: search endpoints are limited to 30 requests per 10 minutes
- Public API rate limiting applies globally to prevent abuse: all endpoints are subject to rate limiting with different tiers based on endpoint type
- OpenSubsonic API endpoints maintain compatibility with existing clients but may not perform as well as Melodee API on very large libraries
- Some legacy OpenSubsonic endpoints do not include pagination metadata which may cause performance issues with large result sets in some client applications

### Admin functionality
- The `/api/admin/jobs/:id` endpoint provides job detail view but results field may be nil for processed jobs
- DLQ management endpoints are optimized for moderate DLQ sizes; very large queues may result in slower performance
- Service binaries and runtime configuration documented in `docs/SERVICE_ARCHITECTURE.md`.
- Baseline Subsonic spec lives in `docs/subsonic-openapi.yaml`; OpenSubsonic extensions (additional endpoints, auth mechanisms, form‑post variants) live in `docs/opensubsonic-openapi.yaml` and should be regenerated whenever upstream publishes a new JSON spec.
- Fixtures for request/response samples reside in `docs/fixtures/` and can be referenced from either API depending on the context.
- When exposing a capability via both APIs, explicitly call out any behavioral differences (pagination limits, response fields) in release notes to avoid client regressions.

## Test Examples

For implementation examples and test patterns, see:
- Authentication: `src/internal/handlers/auth_test.go`
- User management: `src/internal/handlers/user_test.go`
- Playlist operations: `src/internal/handlers/playlist_test.go`
- Library and job management: `src/internal/handlers/library_job_test.go`
- Image handling: `src/internal/handlers/image_test.go`
- Search functionality: `src/internal/handlers/search_test.go`
- OpenSubsonic compatibility: `src/open_subsonic/handlers/*_test.go`





