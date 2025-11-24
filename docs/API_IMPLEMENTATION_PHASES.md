# API Implementation Phases

> Living document tracking implementation status for the two primary APIs:
> - Melodee API (`/api/...`)
> - Subsonic/OpenSubsonic API (`/rest/...`)
>
> Keep this file updated as phases are completed.

## Phase Map

- [X] **Phase 5 – Performance, Pagination & Edge Cases**
- [X] **Admin Frontend Alignment with Melodee API**

This document tracks only **remaining** work; completed items are removed.

---

## Phase 5 – Performance, Pagination & Edge Cases

Focus on the quality of the API under real‑world load and data sizes, ensuring that pagination, filtering, and streaming behave well.

**Current Status (for context)**
- Basic load testing and monitoring infrastructure exists (`load-tests/basic-load-test.js`, `monitoring/dashboards/*`, `monitoring/prometheus/*`).
- Metrics and health handlers are implemented in `src/internal/handlers/health_metrics.go` and `src/internal/handlers/metrics.go`.
- Pagination helpers and metadata are implemented and used by key Melodee endpoints (for example users, libraries, search), but coverage is not yet systematic across **all** list endpoints.

**Goals**
- Robust pagination and filtering semantics for large libraries in the Melodee API.
- Reasonable behavior for OpenSubsonic clients even on large datasets.
- Clear handling of edge cases (empty libraries, deleted media, permission changes).

**Remaining Tasks – Melodee API**
- [X] **Systematically enforce pagination metadata on all list endpoints**
  - Inventory all Melodee list endpoints from `docs/melodee-v1.0.0-openapi.yaml` (for example `GET /api/users`, `/api/playlists`, `/api/libraries`, `/api/search`, `/api/shares`, any others marked as paginated).
  - For each endpoint, ensure the handler uses the shared `pagination` package (`GetPaginationParams`, `Calculate`, `CalculateWithOffset`) and returns a `pagination` object shaped according to the OpenAPI doc.
  - Add or update handler tests to assert that the response includes `pagination` with correct `page/size/total` (or `offset/limit/total`) semantics for at least: users, playlists, libraries, shares, search.

- [X] **Broaden large-offset / large-limit tests beyond search**
  - Add performance- or behavior-focused tests for representative list endpoints **other than** search, such as:
    - `GET /api/playlists` (many playlists, high page numbers),
    - `GET /api/users` (large user base),
    - `GET /api/shares` (many shares).
  - For each, simulate large `offset` / `page` values and maximum `limit` / `pageSize` and assert:
    - queries complete within an acceptable time bound in test (no unexpected timeouts/panics),
    - returned `pagination` metadata matches the requested parameters, and
    - results are capped at the configured maximum page size.

- [X] **Implement and wire rate‑limiting middleware**
  - Implement Fiber-compatible rate‑limiting middleware based on the configuration described in `docs/CONFIG_ENTRY_POINT_PLAN.md` and `docs/TECHNICAL_SPEC.md` (per-user or per-IP windows, tiers per endpoint type, 429 responses with JSON error body).
  - Wire this middleware into the Melodee API server in `src/api/main.go` (and any other binaries that expose `/api/...`), applying stricter limits to expensive endpoints (for example search, library stats) and lighter/global limits elsewhere.
  - Add tests that:
    - configure a very low limit (for example 2 requests per window),
    - issue multiple requests to a protected endpoint, and
    - assert that subsequent requests return HTTP 429 with the configured error payload.
  - Update `docs/API_DEFINITIONS.md` to confirm the implemented limits (numbers and tiers) match the documented behavior.

**Remaining Tasks – Subsonic/OpenSubsonic API**

- [X] **Characterize performance of high-volume OpenSubsonic endpoints**
  - Identify the core "heavy" endpoints (`getIndexes.view`, `getArtists.view`, `getAlbum.view`, `search3.view`) in `src/open_subsonic/handlers`.
  - Add targeted benchmarks or load-style tests (for example in `src/open_subsonic/handlers/*_test.go` or a dedicated `*_performance_test.go`) that:
    - seed the database with a large synthetic dataset (many artists/albums/songs),
    - exercise each endpoint with parameters typical of large libraries, and
    - record/query latency and memory characteristics to ensure they stay within agreed thresholds.
  - Summarize results briefly in `docs/OPENSUBSONIC_PERFORMANCE_EVALUATION.md` or a new short section in `docs/API_DEFINITIONS.md`.

- [X] **Document and enforce server‑side limits for `/rest` where allowed by spec**
  - For endpoints where OpenSubsonic permits server-imposed limits, implement reasonable caps on result counts and/or maximum offsets in the handlers.
  - Mirror those limits in `docs/API_DEFINITIONS.md` under a dedicated "OpenSubsonic limits" subsection (separate from Melodee API limits), including any differences from canonical Subsonic behavior.

- [X] **Exercise contract tests against large datasets**
  - Extend or add to the existing contract tests in `src/open_subsonic/*_contract_test.go` to:
    - run against a fixture dataset that approximates a large real-world library, and
    - explicitly validate that responses remain stable (no truncation or shape changes) when result counts are high.
  - Where necessary, add dedicated large-dataset fixtures under `docs/fixtures/opensubsonic` and document how to run these tests in `docs/TESTING_CONTRACTS.md`.

**Remaining Tasks – Testing & Monitoring**

- [X] **Ensure request-level metrics for `/api` and `/rest` are recorded and visible**
  - Use the `api_requests_total` counter (or similar) from `src/internal/handlers/metrics.go` and add instrumentation in HTTP middleware so that every request to `/api/...` and `/rest/...` increments metrics labeled by method, route, status.
  - Update Grafana dashboards in `monitoring/dashboards/*.json` to break down latency, error rate, and throughput per namespace (`/api`, `/rest`) and per key endpoint group (search, playlists, libraries, streaming).
  - Confirm Prometheus alerts in `monitoring/prometheus/alerts.yml` reference these metrics and have thresholds tuned using data from at least one synthetic load test (`load-tests/basic-load-test.js`).

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

**Backend API Tasks (Melodee API) – Remaining**

- [X] **Align settings endpoint shape across docs, backend, and frontend**
  - Decide on the canonical contract for updating settings:
    - Option A: `PUT /api/settings/:key` with body `{ value }`.
    - Option B: `PUT /api/settings` with body `{ key, value }`.
  - Update `docs/INTERNAL_API_ROUTES.md`, `docs/melodee-v1.0.0-openapi.yaml`, and handler code in `src/internal/handlers/settings.go` (or equivalent) to use the chosen pattern.
  - Update `adminService.updateSetting` in `src/frontend/src/services/apiService.js` to match the canonical contract and add/adjust tests to lock this in.

**Frontend Refactor & Testing Tasks – Remaining**

- [X] **Limit `/rest/...` helpers to clearly flagged compatibility features**
  - Review usage of `mediaService` in `src/frontend/src/services/apiService.js` and any components that call it.
  - If admin UI does not strictly require Subsonic helpers, either:
    - remove those calls from admin views, or
    - gate them behind an explicit "Subsonic compatibility" or "demo browsing" feature flag (for example, an env variable or config toggle passed down to components).
  - Document this in a short comment near `mediaService` and, if appropriate, in a small "Subsonic compatibility" section of the frontend README.

- [X] **Add frontend tests for admin flows against Melodee contracts**
  - For each major admin area (users, libraries, DLQ, shares, settings), add Jest/React Testing Library tests that:
    - render the relevant component with mocked `apiService` responses,
    - assert that the component issues the expected `/api/...` calls with correct method, path, and payload, and
    - verify that error states (for example 401/403, validation errors) are handled in the UI as intended.
  - Optionally add higher-level tests that simulate simple flows (login → navigate to admin dashboard → perform one action in each admin area) using mocked APIs.

**Unit & Integration Testing (Admin + API) – Remaining**

- [X] **Backend admin endpoint coverage**
  - Ensure there are handler-level tests for **each** admin-facing endpoint used by the frontend, including:
    - DLQ: `/api/admin/jobs/dlq`, `/api/admin/jobs/requeue`, `/api/admin/jobs/purge`, `/api/admin/jobs/:id`,
    - Libraries admin actions: `/api/libraries/*` and `/api/admin/capacity*`,
    - Shares: `/api/shares` CRUD,
    - Settings: `/api/settings` (or `/api/settings/:key`, depending on final contract).
  - For each, cover at least one happy path and key error states (unauthorized/forbidden, invalid input, not found).

- [X] **End-to-end admin workflow tests (API level)**
  - Add integration-style tests (for example in `src/internal/tests` or a dedicated admin workflow test file) that:
    - create or authenticate an admin user,
    - perform a small but representative sequence for each admin area (for example: create user → list users; enqueue library scan → query library stats; create share → list shares; inspect DLQ item → requeue it; update a setting and verify it reads back), and
    - assert that responses conform to the Melodee API contracts.

- [X] **Frontend workflow tests (optional but recommended)**
  - Add a small number of high-level frontend tests that mock the Melodee API and validate basic admin flows:
    - login → view dashboard metrics,
    - navigate to DLQ management → list items → trigger requeue,
    - navigate to settings → update a setting and see confirmation,
    - navigate to shares → create/delete a share.
  - These tests should live alongside existing frontend tests and reuse shared test utilities where possible.

**Documentation Tasks – Remaining**

- [X] **Document admin frontend environment variables in one place**
  - Add or update a section (for example in `docs/README.md` or a dedicated `docs/FRONTEND_README.md`) that clearly lists environment variables used by the admin frontend, including `REACT_APP_API_BASE_URL` and any feature flags controlling Subsonic compatibility features.
  - Link this section from `docs/API_DEFINITIONS.md` so that API and frontend configuration guidance stay discoverable.

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
