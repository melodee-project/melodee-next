# API Implementation Phases

> Living document tracking implementation status for the two primary APIs:
> - Melodee API (`/api/...`)
> - Subsonic/OpenSubsonic API (`/rest/...`)
>
> Keep this file updated as phases are completed.

## Phase Map

- [x] **Phase 1 – Baseline Inventory & Routing Parity** (routing + basic smoke tests complete)
- [x] **Phase 2 – Contract Coverage & Fixtures** (contract audit and fixtures still outstanding)
- [ ] **Phase 3 – Unit/Integration Test Hardening** (broaden coverage beyond current core tests)
- [ ] **Phase 4 – Documentation & Developer Experience** (enrich examples and onboarding)
- [ ] **Phase 5 – Performance, Pagination & Edge Cases** (perf, pagination semantics, and SLOs outstanding)


## Coding Agent Template

> You are a senior Go and TypeScript engineer working on the melodee-next monorepo. Your task is to fully implement API Implementation Phase 3 – Unit/Integration Test Hardening as described in API_IMPLEMENTATION_PHASES.md (update the phase number and section based on the work item).

Scope & Codebase

Backend:
Melodee API server at main.go (Go, Fiber).
OpenSubsonic/Subsonic compatibility server at main.go and handlers.
Core domain and handlers under src/internal/... (especially handlers, services, models, middleware).
Frontend:
React admin frontend under src/api/frontend (Vite + React + Tailwind).
Shared API service at src/api/frontend/src/services/apiService.js and related domain-specific services.
Documentation & contracts:
Internal routes: INTERNAL_API_ROUTES.md.
Melodee OpenAPI: melodee-v1.0.0-openapi.yaml.
Subsonic/OpenSubsonic OpenAPI: subsonic-v1.16.1-openapi.yaml, opensubsonic-v1.16.1-openapi.yaml.
General API docs: API_DEFINITIONS.md, TESTING_CONTRACTS.md.
Your Responsibilities

Understand and Plan

Read the relevant phase section in API_IMPLEMENTATION_PHASES.md (e.g., Phase 1–5 or the “Admin Frontend Alignment with Melodee API” section).
Identify all concrete tasks listed for that phase (backend, frontend, tests, docs).
Derive a short checklist for this phase and keep it up to date as you work.
Backend Implementation (Go)

Implement or complete all required handlers and services under src/internal/... and/or src/open_subsonic/... for this phase.
Wire routes in the correct entrypoints:
Melodee API routes in main.go (or clearly factored subrouters).
OpenSubsonic /rest/... routes in main.go.
Ensure:
Request/response shapes, status codes, and error formats match the relevant OpenAPI docs for this phase.
Admin endpoints enforce appropriate auth/role checks via existing middleware.
Pagination, filtering, and performance behaviors respect what the phase describes (especially Phase 5).
When the appendix in API_IMPLEMENTATION_PHASES.md defines target JSON shapes (DLQ/jobs, libraries, settings, shares, admin dashboard), treat them as canonical contracts and implement them accordingly.
Frontend Alignment (if the phase touches the admin UI)

Update src/api/frontend/src/services/apiService.js and any domain services so that:
All admin functionality uses /api/... Melodee endpoints (not /rest/...), except where the phase explicitly allows Subsonic compatibility helpers.
Update React components/pages (e.g., AdminDashboard, UserManagement, LibraryManagement, DLQManagement, SharesManagement, SettingsManagement) so their API calls:
Go through apiService.
Match the backend contracts (field names, pagination metadata, error shapes).
If useful, add a thin typed client layer around apiService for domain operations, but keep changes minimal and consistent with existing style.
Testing

Add or extend unit tests for new/changed behavior:
Backend: *_test.go files alongside handlers/services, including happy paths, validation errors, and auth failures.
OpenSubsonic: extend contract_test.go and related tests so coverage matches the phase’s goals.
For phases that require integration/contract tests, add table-driven tests that validate responses against the documented contracts and fixtures.
Ensure tests for this phase run cleanly via:
go test [source](http://_vscodecontentref_/16). within src (or the narrowest relevant packages).
Frontend tests if you change React code (use the existing test stack).
Fixtures & Contracts

Where the phase calls for fixtures, add/update JSON fixtures in internal and/or opensubsonic so they:
Mirror actual implementation responses.
Cover both success and representative error cases.
Align OpenAPI docs with your implementation for the endpoints touched in this phase:
melodee-v1.0.0-openapi.yaml for /api/....
opensubsonic-v1.16.1-openapi.yaml / subsonic-v1.16.1-openapi.yaml for /rest/....
Documentation

Update the specific docs the phase mentions (e.g., API_DEFINITIONS.md, INTERNAL_API_ROUTES.md, TESTING_CONTRACTS.md, CAPACITY_PROBES.md, HEALTH_CHECK.md, root README.md, or README.md) so they:
Reflect the actual routes, request/response models, example flows, and known limitations.
Clearly describe how to run the relevant tests for this phase.
For admin alignment work, explicitly document that the Admin UI uses Melodee /api/... for admin operations, and how any remaining /rest/... usage is scoped.
Verification and Cleanup

Run the relevant Go and frontend tests and fix any failures you introduce.
Manually sanity-check critical endpoints with a simple curl or HTTP client (no need to commit scripts, just verify behavior).
Keep changes focused on this phase; do not refactor unrelated areas unless strictly necessary.
Deliverables

Compilable, tested backend and (if applicable) frontend code implementing all tasks for this phase.
Updated OpenAPI specs and fixtures for any endpoints you added or changed.
Updated documentation files as specified by the phase.
A brief summary (in comments or a separate note) listing:
Which endpoints/areas were implemented or modified.
Any intentional deviations from the specs and where they are documented.
Any follow-up work or TODOs that are out of scope for this phase.
Work incrementally, keep the existing style and structure, and prefer minimal, well-scoped changes that directly satisfy the phase’s checklist.

---

## Phase 1 – Baseline Inventory & Routing Parity

Clarify exactly which endpoints exist, where they live in the codebase, and how they map to the documented contracts.

**Goals**
- Enumerate all Melodee API routes under `/api/...` and verify alignment with `docs/INTERNAL_API_ROUTES.md` and `docs/melodee-v1.0.0-openapi.yaml`.
- Enumerate all OpenSubsonic routes under `/rest/...` and verify alignment with `docs/subsonic-v1.16.1-openapi.yaml` and `docs/opensubsonic-v1.16.1-openapi.yaml`.
- Identify any documented routes that are not yet implemented, and any implemented routes that are not documented.

**Current Observations (from code scan)**
- Melodee API (`src/api/main.go`):
  - Auth routes: `/api/auth/login`, `/api/auth/refresh`, `/api/auth/request-reset`, `/api/auth/reset`.
  - User routes: `/api/users` CRUD (list/create require admin).
  - Playlist routes: `/api/playlists` CRUD.
  - Library routes: `/api/libraries` (list, get by ID), `/api/libraries/stats`, `/api/libraries/scan`, `/api/libraries/process`, `/api/libraries/move-ok`, `/api/libraries/quarantine` (list, resolve, requeue).
  - Image routes: `/api/images/:id`, `/api/images/avatar`.
  - Share routes: `/api/shares` (list, create, update, delete).
  - Settings routes: `/api/settings` (list, update by key).
  - Jobs/DLQ routes: `/api/admin/jobs/dlq` (list, requeue, purge).
  - Capacity routes: `/api/admin/capacity` (list, get by ID, probe).
  - Search routes: `/api/search`.
  - Health and metrics: `/healthz`, `/metrics`.
- Subsonic/OpenSubsonic API (`src/open_subsonic/main.go`):
  - Browsing, media, search, playlist, user, and system endpoints under `/rest/...` are wired via Fiber, but coverage vs `opensubsonic-v1.16.1-openapi.yaml` is not yet verified.

**Completed Tasks**
- Melodee API:
  - [x] Added route registration in `src/api/main.go` for:
    - [x] Libraries: `/api/libraries/scan`, `/api/libraries/process`, `/api/libraries/move-ok`, `/api/libraries/stats`.
    - [x] Images: `/api/images/:id`, `/api/images/avatar`.
    - [x] Shares: `/api/shares`, `/api/shares/:id`.
    - [x] Settings: `/api/settings`.
    - [x] Jobs/Admin: `/api/admin/jobs/dlq`, `/api/admin/jobs/requeue`, `/api/admin/jobs/purge`.
    - [x] Search: `/api/search`.
  - [x] Created handler implementations in `src/internal/handlers` for missing functionality.
  - [x] Fixed broken handler references in main.go (e.g., DLQ handler).
- Subsonic/OpenSubsonic API:
  - [x] Generated route inventory from `src/open_subsonic/handlers` and compared with `docs/opensubsonic-v1.16.1-openapi.yaml`.
  - [x] Verified that endpoints are properly registered in `src/open_subsonic/main.go`.

**Unit Testing Considerations (Phase 1)**
- [x] Added minimal smoke tests for router wiring:
  - Melodee: tests that hit each documented path and assert non‑404 and basic response structure.
  - OpenSubsonic: tests that hit representative `/rest` endpoints (e.g., `getMusicFolders.view`, `stream.view`, `search3.view`).
- [x] Created handler test files in `src/internal/handlers/*_test.go` for newly wired routes.

**Documentation Tasks (Phase 1)**
- [x] Updated `docs/INTERNAL_API_ROUTES.md` to reflect actual implemented endpoints.
- [x] Added a "Routing overview" section to `docs/API_DEFINITIONS.md` summarizing actual server entrypoints and binaries (`src/api/main.go`, `src/open_subsonic/main.go`).

---

## Phase 2 – Contract Coverage & Fixtures

Ensure the implemented APIs faithfully follow their OpenAPI / spec contracts and have fixtures that can be used for regression testing and client development.

**Goals**
- Achieve endpoint‑by‑endpoint parity between:
  - Melodee API: implementation vs `docs/melodee-v1.0.0-openapi.yaml`.
  - Subsonic/OpenSubsonic API: implementation vs `docs/subsonic-v1.16.1-openapi.yaml` and `docs/opensubsonic-v1.16.1-openapi.yaml`.
- Ensure request/response shapes, status codes, and error models match the contracts.
- Ensure fixtures exist for critical paths.

**Current Status – Melodee API**
- Core `/api/...` endpoints are implemented and wired in `src/api/main.go` and `src/internal/handlers`.
- Some fixtures already exist under `docs/fixtures/internal`, but coverage is incomplete and not systematically aligned with the OpenAPI spec.

**Completed Tasks – Melodee API**
- [x] For each `/api/...` endpoint, verify:
  - [x] Presence in `melodee-v1.0.0-openapi.yaml`.
  - [x] Request parameters (query, path, body) match handler expectations.
  - [x] Response schema matches what handlers actually return.
  - [x] Error responses are modeled (e.g., validation errors, auth failures).
- [x] Add or update fixtures in `docs/fixtures/internal` for:
  - [x] Auth responses (success, invalid credentials, locked account if applicable).
  - [x] Users CRUD (already partially present, verify consistency).
  - [x] Playlists CRUD.
  - [x] Libraries and jobs/admin (request/response examples for scan, process, move‑ok, DLQ operations).
  - [x] Images upload (success, invalid MIME, too large – some fixtures already exist).
  - [x] Search responses with pagination.

**Current Status – Subsonic/OpenSubsonic API**
- A suite of contract tests exists in `src/open_subsonic/*_contract_test.go` and `src/internal/test/contract_validator.go`, covering many `/rest/...` endpoints (artists, playlists, search, streaming, etc.).
- These tests are already integrated into the Go test run, but have been reconciled with `docs/opensubsonic-v1.16.1-openapi.yaml` / `docs/subsonic-v1.16.1-openapi.yaml`.

**Completed Tasks – Subsonic/OpenSubsonic API**
- [x] Use `docs/opensubsonic-v1.16.1-openapi.yaml` as the canonical contract and:
  - [x] Confirm each implemented `/rest/...` endpoint (handlers in `src/open_subsonic/handlers`) conforms to parameter names, formats, and required fields.
  - [x] Document any deviations or intentionally unsupported endpoints.
- [x] Extend or align contract tests in `src/open_subsonic/contract_test.go`:
  - [x] Cover both happy‑path and basic failure modes for streaming, browsing, playlists, and users.
  - [x] Use fixtures in `docs/fixtures/opensubsonic` (and add new ones as needed).

**Unit/Contract Testing (Phase 2)**
- [x] Introduce table‑driven tests that deserialize responses and validate against the OpenAPI models where feasible.
- [x] For OpenSubsonic, ensure `contract_test.go` exercises all major resource types (artists, albums, tracks, playlists, search, cover art).
- [x] Integrate these tests into the default CI test run.

**Documentation Tasks (Phase 2)**
- [x] For any intentional deviations from the upstream Subsonic/OpenSubsonic spec, add a dedicated section to `docs/API_DEFINITIONS.md`.
- [x] Add a short "Contract Testing" subsection to `docs/TESTING_CONTRACTS.md` describing how to run and extend the OpenSubsonic contract tests.

---

## Phase 3 – Unit/Integration Test Hardening

Strengthen unit and integration coverage for core API behaviors, side‑effects, and error handling.

**Current Status**
- Core handlers (auth, libraries, shares, settings, search, DLQ, health) have unit tests in `src/internal/handlers/*_test.go`.
- OpenSubsonic endpoints have contract and handler tests (for example `src/open_subsonic/handlers/media_test.go` and the various `*_contract_test.go` files).
- Integration tests such as `src/internal/integration/full_lifecycle_test.go` already exercise several `/rest/...` flows end‑to‑end.

**Goals**
- High‑value coverage for business logic in handlers and services, especially around auth, playlists, libraries, jobs, and media operations.
- Reproducible integration tests for critical request flows, extending the existing lifecycle and contract tests.

**Planned Tasks – Melodee API**
- [ ] Expand tests in `src/internal/handlers/handler_test.go` and related `*_test.go` files to cover:
  - [ ] Auth flows: login, refresh, password reset (success and failure).
  - [ ] User lifecycle: create, update, delete, list with pagination and role checks.
  - [ ] Playlist lifecycle: create, update, delete, retrieval including boundary/error cases.
  - [ ] Libraries and jobs/admin flows: ensure jobs are enqueued and state transitions occur.
  - [ ] Image upload and retrieval: content‑type enforcement, size limits, and caching headers.
  - [ ] Search behavior: query parsing, type filters, pagination, empty results.
- [ ] Integration tests (where practical) that spin up an in‑memory or test DB and exercise HTTP calls through Fiber.

**Planned Tasks – Subsonic/OpenSubsonic API**
- [ ] Add unit tests in `src/open_subsonic/handlers/*_test.go` for:
  - [ ] Browsing endpoints: folder/index browsing, artist/album/song retrieval.
  - [ ] Media endpoints: streaming, download, cover art, avatar retrieval.
  - [ ] Playlist endpoints: CRUD and parameter validation.
  - [ ] User endpoints: CRUD and auth/role behavior.
  - [ ] System endpoints: ping, license.
- [ ] For endpoints already covered in `contract_test.go`, supplement with unit tests targeting edge cases (e.g., missing IDs, invalid parameters, unauthorized requests).

**Test Infrastructure Tasks**
- [ ] Ensure a consistent test harness exists for both servers:
  - [ ] Helpers to create a Fiber app with test configuration and in‑memory DB or isolated test schema.
  - [ ] Common utilities for seeding users, libraries, and media.
- [ ] Document how to run focused test suites for API and OpenSubsonic (`go test ./src/api/...`, `go test ./src/open_subsonic/...`).

**Documentation Tasks (Phase 3)**
- [ ] Update `docs/TESTING_CONTRACTS.md` with concrete examples of unit vs contract tests for each API.
- [ ] Add a short "How to write new API tests" section linking to sample tests in `src/internal/handlers` and `src/open_subsonic`.

---

## Phase 4 – Documentation & Developer Experience

Polish API documentation and developer onboarding so that both internal and external consumers can adopt the APIs easily.

**Current Status**
- Core docs exist (`docs/API_DEFINITIONS.md`, `docs/INTERNAL_API_ROUTES.md`, `docs/README.md`, root `README.md`, `docs/TESTING_CONTRACTS.md`).
- Phase 1 routing updates and server entrypoints are already documented in `API_DEFINITIONS.md` and `INTERNAL_API_ROUTES.md`.

**Goals**
- Clear, up‑to‑date documentation for both APIs beyond basic routing.
- Easy onboarding for new client developers with concrete examples and workflows.

**Planned Tasks**
- [ ] Review and update `docs/API_DEFINITIONS.md` to:
  - [ ] Include explicit examples of auth, pagination, and error responses for both APIs.
  - [ ] Reference concrete example requests (e.g., `curl`, Postman collections) for common operations.
- [ ] Ensure `docs/melodee-v1.0.0-openapi.yaml` is valid and complete:
  - [ ] Add missing endpoints identified in Phases 1–2.
  - [ ] Regenerate client SDKs if a generation path is desired (optional).
- [ ] Ensure `docs/opensubsonic-v1.16.1-openapi.yaml` and `docs/subsonic-v1.16.1-openapi.yaml` are synchronized with upstream specs.
- [ ] Document service binaries and ports:
  - [ ] `src/api/main.go` – Melodee API server.
  - [ ] `src/open_subsonic/main.go` – OpenSubsonic compatibility server.
- [ ] Add or update README sections for API usage in `docs/README.md` and root `README.md`.

**Unit Testing & Docs Tie‑ins**
- [ ] For key example flows documented (e.g., "Create playlist", "Search library"), ensure there is at least one test that mirrors the documented behavior.
- [ ] Link from docs to representative test files so developers can see working examples.

---

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
- [ ] Validate that all list endpoints enforce and return pagination metadata (`PaginationMetadata` or equivalent) as described in `docs/melodee-v1.0.0-openapi.yaml`.
- [ ] Add or tune database indexes to support common query patterns (search, playlist listing, recent activity).
- [ ] Add tests that simulate large offsets/limits and verify performance‑sensitive queries.
- [ ] Define and document rate‑limiting or protection mechanisms if needed.

**Planned Tasks – Subsonic/OpenSubsonic API**
- [ ] Evaluate performance characteristics of high‑volume endpoints such as `getIndexes.view`, `getArtists.view`, `getAlbum.view`, and `search3.view`.
- [ ] Where the spec allows, implement server‑side limits and document behavior in `API_DEFINITIONS.md` for very large libraries.
- [ ] Add regression tests to ensure responses remain stable under large datasets (e.g., many artists/albums/songs).

**Testing & Monitoring**
- [ ] Add performance‑oriented tests or benchmarks for hotspot endpoints (optional but recommended).
- [ ] Ensure Grafana dashboards (`monitoring/dashboards/*.json`) and Prometheus alerts (`monitoring/prometheus/alerts.yml`) include key API metrics (latency, error rates, throughput) for both `/api` and `/rest` namespaces.

**Documentation Tasks (Phase 5)**
- [ ] Update `docs/CAPACITY_PROBES.md` and `docs/HEALTH_CHECK.md` as needed to reflect real metrics and thresholds.
- [ ] Add a "Known limitations" section to `docs/API_DEFINITIONS.md` describing any unavoidable scalability or pagination quirks for each API.

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
- [ ] For each admin feature in the current admin frontend (see `src/api/frontend/src/components` and `src/api/frontend/src/pages` or their latest equivalents):
  - [ ] Map its current `apiService` calls to documented endpoints in `docs/INTERNAL_API_ROUTES.md` and `docs/melodee-v1.0.0-openapi.yaml`.
  - [ ] Identify any remaining gaps or partial endpoints (for example, missing job detail, incomplete payloads, or undocumented fields for `/api/admin/jobs/...`).
- [ ] Audit and, where needed, refine existing handlers in `src/internal/handlers` and routing in `src/api/main.go` for:
  - [ ] Libraries: stats, scan, process, move‑ok, list.
  - [ ] DLQ/admin jobs: list DLQ, requeue, purge, job detail.
  - [ ] Shares: CRUD.
  - [ ] Settings: get/update single key.
  - [ ] Any additional admin dashboards or metrics endpoints used by the admin dashboard view.
- [ ] Ensure all admin endpoints enforce appropriate auth/role checks via `middleware.NewAuthMiddleware(...).AdminOnly()` where required.
- [ ] Update `docs/melodee-v1.0.0-openapi.yaml` and `docs/INTERNAL_API_ROUTES.md` to reflect the final set of admin endpoints and schemas.

**Frontend Refactor Tasks (Admin React App)**
- [ ] Review the admin API service module (for example `src/api/frontend/src/services/apiService.js`):
  - [ ] Confirm all core admin functions (`authService`, `userService`, `playlistService`, `adminService`, `libraryService`, `metricsService`) use `/api/...` paths consistent with backend routing.
  - [ ] Decide whether any `/rest/...` helpers used by admin views are actually needed; if not, remove them or hide them behind a clearly delineated "Subsonic compatibility" feature flag.
- [ ] For each admin component/page (e.g., `AdminDashboard`, `UserManagement`, `LibraryManagement`, `DLQManagement`, `SharesManagement`, `SettingsManagement`):
  - [ ] Verify that all API calls go through `apiService` and hit `/api/...` endpoints.
  - [ ] Align request/response handling and UI models with the types defined by the Melodee API OpenAPI document.
- [ ] Introduce a small typed client layer (optional but recommended) that wraps `apiService` calls in domain‑specific functions (e.g., `fetchDLQItems()`, `updateSetting(key, value)`) so that future endpoint changes are localized.

**Unit & Integration Testing (Admin + API)**
- [ ] Backend:
  - [ ] Add handler‑level tests (in `src/internal/handlers/*_test.go`) for each admin endpoint used by the frontend, asserting both happy paths and error states (auth failures, validation errors, not found, etc.).
  - [ ] Add integration tests that emulate typical admin workflows: managing users, viewing DLQ, updating settings, managing libraries and shares.
- [ ] Frontend:
  - [ ] Add or extend unit tests around `apiService` consumers (using Jest/React Testing Library or the existing test stack) to ensure correct request shapes and error handling.
  - [ ] Where practical, add integration tests that mock the Melodee API and validate end‑to‑end admin flows (login → dashboard → manage users/playlists/jobs).

**Documentation Tasks**
- [ ] Add a short subsection to `docs/API_DEFINITIONS.md` that explicitly states: "The Admin UI uses the Melodee API (`/api/...`) for all administrative operations; OpenSubsonic (`/rest/...`) is reserved for compatibility with external clients."
- [ ] Document expected environment variables for the admin frontend (e.g., `REACT_APP_API_BASE_URL`) in `docs/README.md` or a dedicated frontend README.
- [ ] Optionally, add a short "Admin API usage" section to `docs/IMPLEMENTATION_GUIDE.md` or `docs/TECHNICAL_SPEC.md` linking the admin features to their corresponding Melodee endpoints.

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
