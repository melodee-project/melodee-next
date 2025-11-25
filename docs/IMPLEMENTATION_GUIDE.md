# Implementation Guide (Phase Plan)

**Audience:** Engineers and coding agents implementing Melodee

**Purpose:** Sequence of implementation phases with links to canonical specs.

**Source of truth for:** Implementation order and phase deliverables (not detailed behavior).

Use this guide to sequence work for coding agents. Check off phases as completed.

## Phase Map
- [x] Phase 1: Bootstrapping & Contracts
- [x] Phase 2: Core Services (API/Web/Worker) Foundations
- [x] Phase 3: Media Pipeline & Directory Code Integration
- [x] Phase 4: OpenSubsonic Compatibility & Clients
- [x] Phase 5: Admin, Jobs, and Observability
- [x] Phase 6: Final Hardening & QA

---

## Coding agent template
> Implement Phase <n> from docs/IMPLEMENTATION_GUIDE.md: read the referenced specs/fixtures for that phase; build the listed deliverables; add/adjust unit tests for new logic; update docs/readmes as needed; mark the Phase <n> checkbox as checked in docs/IMPLEMENTATION_GUIDE.md when done; stop once the phase checklist would be satisfied; summarize what you completed and what’s left for later phases.


---

## Phase 1: Bootstrapping & Contracts
- Align on contracts and configs:
  - Read canonical specs: `TECHNICAL_SPEC.md`, `INTERNAL_API_ROUTES.md`, `METADATA_MAPPING.md`, `HEALTH_CHECK.md`, `CAPACITY_PROBES.md`, `DATABASE_SCHEMA.md`.
  - Use fixtures in `docs/fixtures/` and mapping in `TESTING_CONTRACTS.md`.
  - Envs: `CONFIG_ENTRY_POINT_PLAN.md` (env matrix, per-service samples).
- Deliverables:
  - Generate OpenAPI for internal APIs from `INTERNAL_API_ROUTES.md` and fixtures.
  - Validate health endpoint implementation plan (`HEALTH_CHECK.md`).
  - Confirm FFmpeg path and external tokens availability.

## Phase 2: Core Services (API/Web/Worker) Foundations
- Backend setup:
  - Initialize Go modules per `GO_MODULE_PLAN.md`.
  - Wire DB connection + migrations per `DB_CONNECTION_PLAN.md` and `DATABASE_SCHEMA.md`.
  - Add auth flows (JWT/refresh/reset/lockout) per `TECHNICAL_SPEC.md`.
  - Implement internal routes skeleton (`INTERNAL_API_ROUTES.md`) with contract tests (`TESTING_CONTRACTS.md`).
- Frontend setup:
  - Scaffold React/Vite app with auth + basic navigation; consume internal API OpenAPI client.
- Deliverables:
  - Passing contract tests for auth, users, playlists CRUD (internal fixtures).
  - Running `/healthz` and metrics stubs.

## Phase 3: Media Pipeline & Directory Code Integration
- Implement directory codes and path templates:
  - Follow `DIRECTORY_ORGANIZATION_PLAN.md` and `MEDIA_FILE_PROCESSING.md` (idempotency, capacity thresholds).
- Media pipeline:
  - Inbound -> staging -> production flows, quarantine reasons from `METADATA_MAPPING.md`.
  - FFmpeg profiles and checksum/idempotency rules from `MEDIA_FILE_PROCESSING.md`.
- Deliverables:
  - Worker jobs for scan/process/promote using Asynq queues (`TECHNICAL_SPEC.md` job payloads).
  - Quarantine handling with reason codes.

## Phase 4: OpenSubsonic Compatibility & Clients
- Implement endpoints per `TECHNICAL_SPEC.md` with fixtures in `docs/fixtures/opensubsonic/`.
- Sorting/normalization rules in `TECHNICAL_SPEC.md` and directory article handling in `DIRECTORY_ORGANIZATION_PLAN.md`.
- Streaming/download: range support (fixtures), error handling, transcoding hooks.
- Deliverables:
  - Contract tests for search, playlists, cover/art avatar, stream/download (success/error).
  - Cache/ETag support for art/avatar.

## Phase 5: Admin, Jobs, and Observability
- Admin endpoints:
  - DLQ list/inspect/requeue/purge (`INTERNAL_API_ROUTES.md`, fixtures).
  - Settings/shares/user admin.
- Observability:
  - Metrics/tracing/logging fields (`TECHNICAL_SPEC.md` section 9, `HEALTH_CHECK.md`, `CAPACITY_PROBES.md`).
- Capacity probes:
  - Implement cross-platform probes per `CAPACITY_PROBES.md`.
- Deliverables:
  - Observability dashboards/alerts wired; DLQ admin flows tested against fixtures.

## Phase 6: Final Hardening & QA
- Contract enforcement:
  - Bind `TESTING_CONTRACTS.md` fixtures into CI; fail on drift.
- Security:
  - Verify password rules, lockout/reset, API key rotation.
  - Pen-test uploads (size/MIME) per fixtures.
- Performance:
  - Verify index creation via GORM migrations; query performance tests.
  - Load test streaming/transcoding profiles.
- Deliverables:
  - All phase checkboxes marked, CI green on contract tests, deployment-ready configs (prod env matrix) validated.

## Quick Links
- Architecture/Specs: `TECHNICAL_SPEC.md`, `TECHNICAL_STACK.md`
- Data: `DATABASE_SCHEMA.md`, `DB_CONNECTION_PLAN.md`, `DIRECTORY_ORGANIZATION_PLAN.md`, `MEDIA_FILE_PROCESSING.md`, `METADATA_MAPPING.md`
- Config: `CONFIG_ENTRY_POINT_PLAN.md`, `CAPACITY_PROBES.md`
- APIs: `INTERNAL_API_ROUTES.md`, fixtures under `docs/fixtures/`, `TESTING_CONTRACTS.md`
- Ops: `HEALTH_CHECK.md`, `GAP_RESOLVEMENT_PLAN.md`
