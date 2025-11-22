# Missing Features Analysis & Implementation Plan

This document summarizes remaining gaps in the current Melodee implementation, with a focus on:

- Serving OpenSubsonic clients in a production‑ready way
- Providing a robust, full‑featured admin frontend for operating the system

It reflects the current codebase (Go services + Vite admin UI) and the requirements described in `PRD.md`, `TECHNICAL_SPEC.md`, `INTERNAL_API_ROUTES.md`, `MEDIA_FILE_PROCESSING.md`, `DIRECTORY_ORGANIZATION_PLAN.md`, and `IMPLEMENTATION_GUIDE.md`.

Coding agents should treat the checklists below as **actionable work items**. For each bullet:

- Reference the mentioned files and docs.
- Implement or adjust code and tests in the indicated locations.
- Keep changes minimal and consistent with the existing style.
- When a gap is fully addressed (including tests), mark the checkbox as completed in this file.

## Phase Map
- [ ] Phase 1: Core Backend Services & Infrastructure
- [ ] Phase 2: Media Processing Pipeline
- [ ] Phase 3: Frontend Integration & UI
- [ ] Phase 4: Service Integration & Orchestration
- [ ] Phase 5: Deployment & Environment Configuration
- [ ] Phase 6: Final Testing & Documentation

---

## Coding agent template
> Implement Phase <n> from MISSING_FEATURES.md: read the referenced specs/fixtures for that phase, implement the items under “Remaining Gaps” for that phase (code + tests), keep changes minimal and consistent with existing style, then (1) mark the Phase <n> checkbox and any completed bullets in MISSING_FEATURES.md, (2) update the OpenSubsonic endpoint status matrix rows you’ve fully implemented, and (3) summarize what you completed and what’s left for later phases.


---

## Phase 1: Core Backend Services & Infrastructure

### Status Summary

- Core service layout (`api`, `web`, `worker`) and Go modules exist and compile.
- Internal REST routes for auth, users, playlists, libraries, jobs, settings, and shares are wired in `src/main.go` against the internal repository.
- Database connection manager, migrations, and partition helpers are implemented per `DATABASE_SCHEMA.md`/`DB_CONNECTION_PLAN.md`.
- JWT authentication middleware and login/refresh flows exist; password reset and lockout are partially represented but not fully wired through UX and tests.
- Basic health endpoint and metrics stubs exist.

### Remaining Gaps (Backend Core)

- [ ] **Auth flows completeness**
	- Implement and test password reset + account lockout per `INTERNAL_API_ROUTES.md` and `TECHNICAL_SPEC.md`:
		- Implement `/api/auth/request-reset` and `/api/auth/reset` handlers in `src/internal/handlers` (or appropriate package) including email‑agnostic 202 behavior, token verification, and password policy errors.
		- Implement lockout tracking (failed login counters, lockout window) in the auth service/repository.
		- Add unit tests under `src/internal/tests` to cover success and error paths.
- [ ] **Error model + consistency**
	- Standardize internal REST error responses according to `TESTING_CONTRACTS.md`:
		- Introduce a shared error response helper (similar to OpenSubsonic’s `SendOpenSubsonicError`) in `src/internal/utils`.
		- Update handlers in `src/internal/handlers` to use this helper, and add/adjust tests to assert on error JSON shape.
- [ ] **Security middleware**
	- Add rate‑limiting/IP throttling middleware in `src/internal/middleware` and wire it in `src/main.go` for public APIs.
	- Implement stronger upload guards (size and MIME) for `/api/images/avatar` based on fixtures under `docs/fixtures/internal` once they exist, with tests.
- [ ] **Config + env validation coverage**
	- Extend `src/internal/config` to:
		- Validate FFmpeg binary path and required profiles from `MEDIA_FILE_PROCESSING.md` at startup.
		- Optionally check presence of external metadata service tokens if/when those integrations are enabled.
	- Add tests in `config_test.go` for missing/invalid FFmpeg configuration.
- [ ] **Repository coverage**
	- Extend the internal repository in `src/internal/services/repository.go` (and related files) to support:
		- All filters and pagination options required by `GET /api/search` and playlist endpoints.
	- Add repository tests that exercise these new methods.

## Phase 2: Media Processing Pipeline

### Description
Build the core media processing functionality following the three-stage workflow (inbound → staging → production) with proper file handling and organization.

### Checklist
- [ ] Implement inbound directory scanning service
- [ ] Build file validation and metadata extraction
- [ ] Create staging area with proper quarantine handling
- [ ] Implement directory code generation with collision handling
- [ ] Build FFmpeg transcoding integration
- [ ] Create checksum validation and idempotency checks
- [ ] Implement file normalization and organization logic
- [ ] Build three-stage workflow: inbound → staging → production
- [ ] Create quarantine system with reason codes
- [ ] Build job queue processing for media tasks
- [ ] Implement capacity monitoring and probes
- [ ] Create path template resolution with directory codes
- [ ] Implement album/song/cue sheet processing
- [ ] Build metadata mapping and conflict resolution

### Notes vs Current Implementation

- Core scan/process/promote jobs exist in `internal/jobs` and are wired via Asynq; quarantine reasons in `internal/media/quarantine.go` match the codes in `METADATA_MAPPING.md`.
- Directory codes and path templates are implemented in `internal/directory` but collision and normalization behavior is only lightly tested.
- FFmpeg integration is intentionally stubbed: handlers in `open_subsonic/handlers/media.go` call a placeholder transcoder that currently just returns the original file path.

**Remaining Gaps (Pipeline)**

- [ ] Implement FFmpeg‑based transcoding and caching per `MEDIA_FILE_PROCESSING.md` (profiles, max bit‑rate, idempotent outputs).
- [ ] Add tests/fixtures that validate directory code generation and path template behavior for the collision and normalization scenarios described in `DIRECTORY_ORGANIZATION_PLAN.md`.
- [ ] Expose inbound/staging/production and quarantine state through internal APIs and the admin UI (per `PRD.md` user flows).

## Phase 3: Frontend Integration & UI

### Description
Connect React/Vite frontend components to backend services, implement admin UI, and create user interfaces for all required functionality.

### Checklist
- [ ] Complete API service integration with backend endpoints
- [ ] Implement authentication context and flow
- [ ] Build user dashboard with library statistics
- [ ] Create library browsing interface (folders, indexes, artists, albums, songs)
- [ ] Develop media streaming and download functionality
- [ ] Build search interface with filters and pagination
- [ ] Create playlist management UI
- [ ] Implement user management interface (admin only)
- [ ] Build admin dashboard with system monitoring
- [ ] Create DLQ management interface
- [ ] Implement settings management UI
- [ ] Build shares management interface
- [ ] Add proper navigation and routing
- [ ] Create responsive and accessible UI components

### Notes vs Current Implementation

- A Vite/React admin app exists under `src/frontend` with routes for `/login`, `/admin`, `/admin/dlq`, `/admin/users`, `/admin/settings`, and a placeholder `/admin/shares`.
- Components `DLQManagement`, `UserManagement`, and `SettingsManagement` implement basic tables/forms, but they currently call non‑versioned paths like `/users`, `/admin/jobs/dlq`, `/admin/settings` instead of the `/api/...` contracts in `INTERNAL_API_ROUTES.md`.
- `AdminDashboard` exists twice (static component in `components/AdminDashboard.jsx` and a more dynamic inline version inside `App.jsx`), indicating an unfinished refactor.

**Remaining Gaps (Admin UI)**

- [ ] Align all admin API calls with `INTERNAL_API_ROUTES.md` and fixtures:
	- Use `/api/users`, `/api/admin/jobs/dlq`, `/api/admin/jobs/requeue`, `/api/admin/jobs/purge`, `/api/settings`, `/api/shares`, etc.
	- Adjust request/response shapes to match JSON fixtures in `docs/fixtures/internal`.
- [ ] Consolidate `AdminDashboard` into a single component that:
	- Shows library statistics from `GET /api/libraries/stats`.
	- Shows recent jobs from an internal admin jobs endpoint once implemented.
- [ ] Implement UI for:
	- Library configuration and status (inbound/staging/production paths, scan/process/promote controls).
	- Quarantine review and actions for albums/tracks.
	- Shares management (create/list/delete) matching `/api/shares` contracts.
- [ ] Implement auth context that uses `/api/auth/login` and `/api/auth/refresh` tokens/roles instead of only `localStorage` flags.

## Phase 4: Service Integration & Orchestration

### Description
Connect all backend services together, implement message queuing with Asynq, and ensure proper communication between services.

### Checklist
- [ ] Implement proper Asynq job processors for all required jobs
- [ ] Connect worker jobs for scan/process/promote workflows
- [ ] Implement job monitoring and DLQ management
- [ ] Create pub/sub messaging between services
- [ ] Implement cross-service authentication
- [ ] Build service discovery and health monitoring
- [ ] Create proper service-to-service communication protocols
- [ ] Implement distributed tracing
- [ ] Build service coordination for file processing
- [ ] Create failover and redundancy mechanisms
- [ ] Implement job scheduling and periodic tasks
- [ ] Build event-driven architecture for media changes

### Notes vs Current Implementation

- Asynq client and worker wiring exist; core media jobs (scan/process/promote) are implemented and invoked by internal routes.
- OpenSubsonic handlers directly query the internal GORM models rather than going through separate microservices, which is acceptable for the current monolith but should still follow the contracts in `TECHNICAL_SPEC.md`.

**Remaining Gaps (Integration)**

- [ ] Add job monitoring endpoints (`/api/admin/jobs/*`) with DLQ detail that matches fixtures and is consumable by `DLQManagement`.
- [ ] Implement capacity probes and health/metrics endpoints per `CAPACITY_PROBES.md` and `HEALTH_CHECK.md`, and surface them to the admin dashboard.

## Phase 5: Deployment & Environment Configuration

### Description
Complete the deployment infrastructure, environment configuration, and container orchestration setup.

### Checklist
- [ ] Finalize Docker container configurations for all services
- [ ] Complete docker-compose.yml with proper service links
- [ ] Implement environment variable configuration system
- [ ] Create production-ready configuration files
- [ ] Implement secrets management for sensitive data
- [ ] Build CI/CD pipeline configuration
- [ ] Implement health checks for container orchestration
- [ ] Create backup and disaster recovery procedures
- [ ] Implement monitoring and alerting configuration
- [ ] Build deployment scripts for different environments
- [ ] Implement SSL/HTTPS configuration
- [ ] Create database backup automation
- [ ] Implement service scaling configurations

## Phase 6: Final Testing & Documentation

### Description
Complete testing, documentation, and hardening to ensure production readiness.

### Checklist
- [ ] Implement comprehensive unit testing for all services
- [ ] Create integration tests for service communication
- [ ] Build contract tests for API endpoints
- [ ] Perform load testing and performance optimization
- [ ] Implement security testing and penetration testing
- [ ] Complete documentation for all features and APIs
- [ ] Build monitoring dashboards and alerting rules
- [ ] Perform end-to-end system testing
- [ ] Implement logging and monitoring in production
- [ ] Create deployment validation procedures
- [ ] Perform user acceptance testing
- [ ] Complete final documentation and README files
- [ ] Validate all Phase requirements from IMPLEMENTATION_GUIDE.md

---

## OpenSubsonic / Subsonic Client Support Gaps

This section calls out gaps specific to serving Subsonic/OpenSubsonic clients as described in `PRD.md` and `TECHNICAL_SPEC.md`.

### What Exists

- `/rest/*` routes are wired in `src/main.go` to handlers in `src/open_subsonic/handlers` with an auth middleware in `open_subsonic/middleware`.
- Browsing (`getMusicFolders`, `getIndexes`, `getArtists`, `getArtist`, `getAlbum`, `getMusicDirectory`, `getSong`, `getGenres`) and media (`stream`, `download`, `getCoverArt`, `getAvatar`) endpoints hit real DB models and return XML via `open_subsonic/utils` helpers.
- Error responses use `SendOpenSubsonicError` with numeric codes, and helper tests validate basic XML envelope shape and required attributes.

### Still Missing or Incomplete

- [ ] **Auth semantics**: Confirm and test all supported auth variants (`u`+`p`/`enc:`, `u`+`t`+`s`) as specified in the OpenSubsonic spec; add tests that prove failing/expired auth returns the right XML error codes.
- [ ] **Search contract coverage**: Implement and test `search.view`, `search2.view`, and `search3.view` logic with proper sorting, pagination, and normalization per `TECHNICAL_SPEC.md` and fixtures under `docs/fixtures/opensubsonic`.
- [ ] **Playlist endpoints**: Ensure `getPlaylists`, `getPlaylist`, `createPlaylist`, `updatePlaylist`, and `deletePlaylist` implement the correct semantics and XML shapes (static vs dynamic playlists, error handling) and are covered by contract tests.
- [ ] **Streaming & transcoding**: Replace the placeholder `transcodeFile` with real FFmpeg integration (profiles, caching, range + content‑type correctness) and add tests for `maxBitRate`, `format`, and HTTP Range behavior.
- [ ] **Cover art & avatar caching**: Verify ETag/Last‑Modified and 304 behavior for `getCoverArt`/`getAvatar` against fixtures; add tests for missing art and fallbacks.
- [ ] **Indexing and sorting rules**: Expand normalization (articles, diacritics, punctuation) so `getIndexes` and `getArtists` behavior matches `DIRECTORY_ORGANIZATION_PLAN.md` and OpenSubsonic expectations, with explicit tests for tricky artist names.
- [ ] **Dynamic genres/tags**: Replace the hard‑coded genre list with aggregation from song/album tags, returning accurate counts.
- [ ] **Contract tests**: Convert `open_subsonic/contract_test.go` from placeholders into real tests that spin up an in‑memory server and validate XML responses against fixtures for success and error scenarios.

### Endpoint Status Matrix (High Level)

| Area        | Endpoint / Feature              | Status       | Notes |
|------------|----------------------------------|-------------|-------|
| System     | `ping.view`                     | Partial      | Route and handler exist; basic XML envelope tested, but no full auth/error cases or fixture-based contract tests. |
| System     | `getLicense.view`               | Partial      | Handler stubbed; structure defined but not fully exercised by tests or fixtures. |
| Browsing   | `getMusicFolders.view`          | Implemented  | Queries `Library` models and returns folders; needs fixture-based verification. |
| Browsing   | `getIndexes.view`               | Partial      | Indexing and normalization implemented; needs edge-case coverage vs directory/article rules. |
| Browsing   | `getArtists.view`               | Implemented  | Paginates unlocked artists; relies on `ParsePaginationParams`; tests missing. |
| Browsing   | `getArtist.view`                | Implemented  | Returns albums for artist; depends on album status/filtering as per spec. |
| Browsing   | `getAlbum.view`                 | Implemented  | Returns album + songs; path/bitrate/year fields populated; needs fixtures. |
| Browsing   | `getMusicDirectory.view`        | Implemented  | Handles artist-or-album IDs; status OK but untested for complex trees. |
| Browsing   | `getAlbumInfo.view`             | Partial      | Basic struct returned; fields incomplete relative to spec. |
| Browsing   | `getGenres.view`                | Stub / Demo  | Returns static sample genres; must be replaced with real aggregation. |
| Media      | `stream.view`                   | Partial      | Streams real files, supports Range and basic transcoding hook; FFmpeg pipeline still stubbed. |
| Media      | `download.view`                 | Implemented  | Sends full file with ETag/Last-Modified; tests missing for error paths. |
| Media      | `getCoverArt.view`              | Partial      | Looks up common filenames; needs better directory integration and cache tests. |
| Media      | `getAvatar.view`                | Partial      | File-based lookup under `/melodee/user_images`; no linkage to user profiles yet. |
| Search     | `search.view` / `search2.view` / `search3.view` | Partial | Handlers exist and are wired; behavior not yet validated against fixtures and normalization rules. |
| Playlists  | `getPlaylists.view`             | Partial      | Handler and route exist; end-to-end contract tests and error handling not in place. |
| Playlists  | `getPlaylist.view`              | Partial      | Structured responses expected; verify IDs, song lists, and ACLs. |
| Playlists  | `createPlaylist.view`           | Partial      | Basic support; no fixtures or negative-tests wired. |
| Playlists  | `updatePlaylist.view`           | Partial      | Similar to create; need full coverage for adds/removes/renames. |
| Playlists  | `deletePlaylist.view`           | Partial      | Deletes playlist; error semantics not yet tested. |
| Users      | `getUser.view`                  | Partial      | Handler exists; ensure mapping from internal users and roles is correct. |
| Users      | `getUsers.view`                 | Partial      | Basic listing; needs pagination/role filtering per spec. |
| Users      | `createUser.view`               | Planned/Stub | Ensure Subsonic-style semantics are desired; currently aligned more with internal admin APIs. |
| Users      | `updateUser.view`               | Planned/Stub | Same as create; confirm supported fields vs OpenSubsonic spec. |
| Users      | `deleteUser.view`               | Planned/Stub | Ensure correct error codes when deleting self/last admin. |


---

## Admin Frontend Gaps (Operator Experience)

This section focuses on gaps preventing the React/Vite admin app from being a full‑featured operator console as described in `PRD.md` and `IMPLEMENTATION_GUIDE.md`.

### What Exists

- Auth‑gated React app under `src/frontend` with routes and skeleton components for dashboard, DLQ, user management, settings, and shares.
- Basic data tables and forms for users, DLQ items, and settings.

### Still Missing or Incomplete

- [ ] **Route & payload alignment**: Update all frontend API calls to use `/api/...` routes and JSON shapes from `INTERNAL_API_ROUTES.md` and `docs/fixtures/internal`.
- [ ] **Library & pipeline views**: Add pages for viewing libraries (inbound/staging/production), triggering scans/process/promote operations, and inspecting pipeline status.
- [ ] **Quarantine management**: Add UI to list quarantine items, show reason codes, and offer actions (fix/ignore/requeue) mapped to internal APIs.
- [ ] **Shares management**: Implement full CRUD UI for `/api/shares` (name, ids, expiry, max_streaming_minutes, allow_download) instead of the current placeholder text.
- [ ] **System health & capacity**: Surface `/healthz`, metrics, and capacity probe data in the dashboard with clear status and guidance.
- [ ] **Playlist & search UX**: Provide admin‑oriented tools for searching/browsing artists/albums/songs and managing playlists as per `PRD.md`.
- [ ] **Auth UX completeness**: Implement login, logout, password reset, and lockout flows in the UI that correspond to the internal auth endpoints.

## Key Infrastructure Gaps Identified

1. **Broken Package References:** Many components reference non-existent packages causing compilation errors
2. **Skeleton Code:** Large portions consist only of function signatures with no implementation
3. **Disconnected Components:** Frontend and backend components have no actual connection
4. **Missing Business Logic:** Core functionality like media processing exists only as placeholders
5. **Incomplete Job Processing:** Asynq queues are defined but not implemented
6. **Missing Configurations:** Environment variables and configuration files are incomplete
7. **Insufficient Error Handling:** No proper error handling or validation throughout
8. **No Service Communication:** Services don't communicate with each other properly

This document provides a roadmap to transform the current skeleton implementation into a fully functional, production-ready system.