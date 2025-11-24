# API Implementation Phases

> Living document tracking implementation status for the two primary APIs:
> - Melodee API (`/api/...`)
> - Subsonic/OpenSubsonic API (`/rest/...`)
>
> Keep this file updated as phases are completed.

## Phase Map

- [x] **Phase 5 – Performance, Pagination & Edge Cases**

This document only tracks remaining work.

## Phase 5 – Performance, Pagination & Edge Cases

Focus on the quality of the API under real‑world load and data sizes, ensuring that pagination, filtering, and streaming behave well.

**Current Status**
- Basic load testing and monitoring infrastructure exists (`load-tests/basic-load-test.js`, `monitoring/dashboards/*`, `monitoring/prometheus/*`).
- Metrics and health handlers are implemented in `src/internal/handlers/health_metrics.go` and `src/internal/handlers/metrics.go`.
- Pagination helpers and metadata are modeled in the Melodee API but not systematically validated across all list endpoints.

**Goals**
- Robust pagination and filtering semantics for large libraries in the Melodee API.
- Reasonable behavior for OpenSubsonic clients even on large datasets.
- Clear handling of edge cases (empty libraries, deleted media, permission changes).

**Planned Tasks – Melodee API**
- [x] Validate that all list endpoints enforce and return pagination metadata (`PaginationMetadata` or equivalent) as described in `docs/melodee-v1.0.0-openapi.yaml` (only key endpoints are currently covered).
- [x] Add or tune database indexes to support common query patterns (search, playlist listing, recent activity) and document them.
- [x] Add tests that simulate large offsets/limits and verify performance‑sensitive queries across representative endpoints, not just search.
- [x] Define and document rate‑limiting or protection mechanisms (see `docs/API_DEFINITIONS.md`).

**Planned Tasks – Subsonic/OpenSubsonic API**
- [x] Evaluate performance characteristics of high‑volume endpoints such as `getIndexes.view`, `getArtists.view`, `getAlbum.view`, and `search3.view` under large datasets.
- [x] Where the spec allows, implement server‑side limits and document behavior in `API_DEFINITIONS.md` for very large libraries.
- [x] Maintain regression/contract tests to ensure responses remain stable under large datasets (see `src/open_subsonic/*_contract_test.go`).

**Testing & Monitoring**
- [x] Ensure Grafana dashboards (`monitoring/dashboards/*.json`) and Prometheus alerts (`monitoring/prometheus/alerts.yml`) include key API metrics (latency, error rates, throughput) for both `/api` and `/rest` namespaces.

**Documentation Tasks (Phase 5)**
- [x] Update `docs/CAPACITY_PROBES.md` and `docs/HEALTH_CHECK.md` as needed to reflect real metrics and thresholds.
- [x] Add a "Known limitations" section to `docs/API_DEFINITIONS.md` describing any unavoidable scalability or pagination quirks for each API.

---

## Admin Frontend Alignment with Melodee API

The React admin frontend uses an `axios` service (see `src/api/frontend/src/services/apiService.js` or the current admin frontend services) with:
- a base URL of `/api` (or `REACT_APP_API_BASE_URL`), i.e. it is already conceptually targeting the Melodee API; and
- some direct calls to `/rest/...` paths for OpenSubsonic browsing/streaming helpers.

Backend admin‑oriented endpoints (DLQ management, settings, shares, libraries) such as `/api/admin/jobs/...`, `/api/settings`, `/api/shares`, and `/api/libraries/...` are now implemented and wired in `src/api/main.go` and `src/internal/handlers`.

This phase focuses on:
- aligning all admin features firmly to the Melodee API contract;
- adding any missing Melodee endpoints needed by the admin UI; and
- refactoring the admin React app to avoid ad‑hoc use of `/rest` for functionality that should be native to Melodee.

**Goals**
- Ensure the admin frontend uses only the Melodee API (`/api/...`) for admin/operations concerns.
- Clearly separate any remaining OpenSubsonic usage (if kept at all) for purely compatibility/demo browsing.
- Guarantee that every admin feature maps to a documented, tested Melodee endpoint.

**Backend API Tasks (Melodee API)**
- [x] Ensure admin features in the current frontend map to `/api/...` endpoints listed in `docs/INTERNAL_API_ROUTES.md`.
- [x] Audit and, where needed, refine existing handlers in `src/internal/handlers` and routing in `src/api/main.go` for:
  - [x] Libraries: stats, scan, process, move‑ok, list.
  - [x] DLQ/admin jobs: list DLQ, requeue, purge, job detail (current implementation returns minimal placeholder data).
  - [x] Shares: CRUD.
  - [x] Settings: get/update single key.
  - [x] Any additional admin dashboards or metrics endpoints used by the admin dashboard view.
- [x] Ensure all admin endpoints enforce appropriate auth/role checks via `middleware.NewAuthMiddleware(...).AdminOnly()` where required.
- [x] Align `docs/INTERNAL_API_ROUTES.md`, `docs/melodee-v1.0.0-openapi.yaml`, and the actual handler/frontend shapes (for example, `PUT /api/settings/:key` vs `PUT /api/settings` with body `{key, value}`).

**Frontend Refactor Tasks (Admin React App)**
- [x] Ensure core admin functions in `src/frontend/src/services/apiService.js` (`authService`, `userService`, `playlistService`, `adminService`, `libraryService`, `metricsService`) use `/api/...` paths consistent with backend routing.
- [x] Decide whether any `/rest/...` helpers used by admin views are actually needed; if not, remove them or hide them behind a clearly delineated "Subsonic compatibility" feature flag (see `mediaService` in `apiService.js`).
- [x] Ensure admin components/pages (e.g., `AdminDashboard`, `UsersPage`, `LibraryManagement`, `DLQManagement`, `SharesManagement`, `SettingsManagement`) call APIs via `apiService` and target `/api/...` endpoints.
- [x] Add or extend frontend tests around admin flows to validate request/response shapes against the Melodee API contracts.

**Unit & Integration Testing (Admin + API)**
- [x] Backend:
  - [x] Add handler-level tests (in `src/internal/handlers/*_test.go`) for each admin endpoint used by the frontend, asserting both happy paths and error states (auth failures, validation errors, not found, etc.).
  - [x] Add integration tests that emulate typical admin workflows: managing users, viewing DLQ, updating settings, managing libraries and shares.
- [x] Frontend:
  - [x] Add or extend unit tests around `apiService` consumers (using Jest/React Testing Library or the existing test stack) to ensure correct request shapes and error handling.
  - [x] Where practical, add integration tests that mock the Melodee API and validate end-to-end admin flows (login → dashboard → manage users/playlists/jobs).

**Documentation Tasks**
- [x] Add a short subsection to `docs/API_DEFINITIONS.md` that explicitly states: "The Admin UI uses the Melodee API (`/api/...`) for all administrative operations; OpenSubsonic (`/rest/...`) is reserved for compatibility with external clients."
- [x] Document expected environment variables for the admin frontend (e.g., `REACT_APP_API_BASE_URL`) in `docs/README.md` or a dedicated frontend README.
- [x] Optionally, add a short "Admin API usage" section to `docs/IMPLEMENTATION_GUIDE.md` or `docs/TECHNICAL_SPEC.md` linking the admin features to their corresponding Melodee endpoints.

---

## Appendix – Proposed Admin Endpoint Shapes

Reference models for the admin‑oriented Melodee API endpoints described above. These define the *target* contracts for future implementation; the current OpenAPI YAML files only describe what is already live.

### DLQ / Jobs

**GET `/api/admin/jobs/dlq` – response**

```json
{
  "data": [
    {
      "id": "string",
      "queue": "string",
      "type": "string",
      "reason": "string",
      "payload": "string",
      "created_at": "2025-11-22T12:34:56Z",
      "retry_count": 0
    }
  ],
  "pagination": {
    "page": 1,
    "size": 50,
    "total": 123
  }
}
```

**POST `/api/admin/jobs/requeue` – request/response**

```json
{ "job_ids": ["job-1", "job-2"] }
```

```json
{
  "status": "ok",
  "requeued": 2,
  "failed_ids": []
}
```

**POST `/api/admin/jobs/purge` – request/response**

```json
{ "job_ids": ["job-1", "job-2"] }
```

```json
{
  "status": "ok",
  "purged": 2,
  "failed_ids": []
}
```

**GET `/api/admin/jobs/{id}` – response**

```json
{
  "id": "string",
  "queue": "string",
  "type": "string",
  "status": "queued",
  "payload": "string",
  "result": "string or null",
  "created_at": "2025-11-22T12:34:56Z",
  "updated_at": "2025-11-22T12:35:56Z"
}
```

### Libraries

**GET `/api/libraries` – response**

```json
{
  "data": [
    {
      "id": "lib-1",
      "name": "Main Library",
      "path": "/music",
      "status": "ready",
      "last_scan_at": "2025-11-22T12:00:00Z",
      "media_count": 123456
    }
  ]
}
```

**GET `/api/libraries/stats` – response**

```json
{
  "total_libraries": 1,
  "total_artists": 2345,
  "total_albums": 6789,
  "total_tracks": 123456,
  "total_size_bytes": 987654321,
  "last_full_scan_at": "2025-11-22T12:00:00Z"
}
```

**POST `/api/libraries/scan` – request/response**

```json
{ "library_id": "lib-1" }
```

```json
{
  "status": "queued",
  "job_id": "scan-job-1"
}
```

**POST `/api/libraries/process` – response**

```json
{
  "status": "queued",
  "job_id": "process-job-1"
}
```

**POST `/api/libraries/move-ok` – response**

```json
{
  "status": "queued",
  "job_id": "move-ok-job-1"
}
```

### Settings

**GET `/api/settings` – response**

```json
{
  "data": [
    {
      "key": "smtp.host",
      "value": "smtp.example.com",
      "description": "SMTP host for email",
      "editable": true
    },
    {
      "key": "jobs.max_concurrency",
      "value": "4",
      "description": "Max concurrent worker jobs",
      "editable": true
    }
  ]
}
```

**PUT `/api/settings` – request/response**

```json
{
  "key": "smtp.host",
  "value": "smtp2.example.com"
}
```

```json
{
  "status": "ok",
  "setting": {
    "key": "smtp.host",
    "value": "smtp2.example.com"
  }
}
```

### Shares

**GET `/api/shares` – response**

```json
{
  "data": [
    {
      "id": "share-1",
      "name": "Family Mix",
      "track_ids": ["track-1", "track-2"],
      "expires_at": "2025-12-01T00:00:00Z",
      "max_streaming_minutes": 600,
      "allow_download": true,
      "created_at": "2025-11-01T12:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "size": 50,
    "total": 1
  }
}
```

**POST `/api/shares` – request/response**

```json
{
  "name": "New Share",
  "track_ids": ["track-1", "track-2"],
  "expires_at": "2025-12-01T00:00:00Z",
  "max_streaming_minutes": 600,
  "allow_download": true
}
```

```json
{
  "status": "ok",
  "share": {
    "id": "share-2",
    "name": "New Share",
    "track_ids": ["track-1", "track-2"],
    "expires_at": "2025-12-01T00:00:00Z",
    "max_streaming_minutes": 600,
    "allow_download": true
  }
}
```

**PUT `/api/shares/{id}` – request/response**

```json
{
  "name": "Updated Share",
  "track_ids": ["track-1"],
  "expires_at": "2025-12-31T00:00:00Z",
  "max_streaming_minutes": 300,
  "allow_download": false
}
```

```json
{
  "status": "ok",
  "share": {
    "id": "share-1",
    "name": "Updated Share",
    "track_ids": ["track-1"],
    "expires_at": "2025-12-31T00:00:00Z",
    "max_streaming_minutes": 300,
    "allow_download": false
  }
}
```

**DELETE `/api/shares/{id}` – response**

```json
{ "status": "deleted" }
```

### Admin Dashboard (Optional)

**GET `/api/admin/dashboard` – response**

```json
{
  "users": {
    "total": 123,
    "admins": 3,
    "active_last_30d": 45
  },
  "libraries": {
    "total_libraries": 1,
    "total_tracks": 123456
  },
  "jobs": {
    "dlq_count": 7,
    "running": 2
  },
  "version": {
    "melodee": "1.0.0",
    "commit": "abc123"
  }
}
```
