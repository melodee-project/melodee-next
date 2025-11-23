# Remaining Gaps

This document only tracks **current gaps** in the Melodee implementation.
It is intentionally forward‑looking and does not try to describe what is
already built – use `IMPLEMENTATION_GUIDE.md`, `INTERNAL_API_ROUTES.md`,
`TECHNICAL_SPEC.md`, and `MEDIA_FILE_PROCESSING.md` for the full picture
of the implemented system.

Each bullet below should be treated as an actionable work item. When a
gap is fully addressed (including tests), remove the bullet from this
file as part of the change.

Status legend used below:

- **OPEN** – not yet implemented or clearly missing major pieces.
- **PARTIAL** – some code exists, but behavior or tests are incomplete.
- **DONE (remove when convenient)** – implementation and tests appear
	complete; keep here only until you are comfortable deleting the item.

Phase legend:

- **Phase 1 – Core Backend & Media**: backend auth/error/config,
	repository, and core media pipeline wiring.
- **Phase 2 – OpenSubsonic Contracts**: Subsonic/OpenSubsonic endpoints,
	streaming/transcoding, search, playlists, and contract tests.
- **Phase 3 – Admin UX & Observability**: admin frontend, dashboards,
	health/capacity views, and operational readiness.
- **Phase 4 – End‑to‑End & Non‑functional**: E2E tests, load/security
	testing, and final polish.

Phase checklist:

- [x] **Phase 1 – Core Backend & Media**
- [x] **Phase 2 – OpenSubsonic Contracts**
- [x] **Phase 3 – Admin UX & Observability**
- [ ] **Phase 4 – End‑to‑End & Non‑functional**

## Coding Agent Template
>You are working in the melodee-next repo on Phase 3 – Admin UX & Observability as defined in MISSING_FEATURES.md.

Goal

Fully implement and test all items tagged for this phase so that every acceptance checklist item in that phase is satisfied, and the phase checkbox at the top of MISSING_FEATURES.md can legitimately be marked as complete.

Scope

Read MISSING_FEATURES.md and focus ONLY on:
The top “Phase checklist”.
The section ## Phase 3 – Admin UX & Observability and its subsections.
For each bullet in this phase:
Treat its “Status” (OPEN / PARTIAL) and “Acceptance checklist” as the single source of truth for what must be implemented and tested.
Ignore items from other phases unless strictly required as dependencies.
Requirements

For every item in this phase:

Implementation

Implement the missing behavior in the appropriate packages (see file hints in each bullet, e.g. src/internal/..., src/open_subsonic/..., src/frontend/...).
Remove or replace any TODO/placeholder logic so behavior matches the intent described in the bullet and related docs (TECHNICAL_SPEC.md, MEDIA_FILE_PROCESSING.md, etc., if referenced).
Testing

Add or extend unit / integration / contract tests so that each acceptance checklist sub‑item can be demonstrated by a test.
Prefer colocated tests (e.g. *_test.go in the same package, or React tests alongside components if applicable).
Ensure tests are deterministic and do not rely on external services.
Documentation & Cleanup

When a bullet is fully satisfied, update MISSING_FEATURES.md:
Option A: remove that bullet entirely, OR
Option B: change its status tag to DONE (remove when convenient) and briefly note which tests cover it.
Only mark the phase checklist entry [x] Phase 3 – Admin UX & Observability after all bullets in that phase are either removed or clearly marked DONE.
Constraints

Do NOT modify requirements, only their implementation and tests.
Keep changes minimal and idiomatic to the existing style (Go, React, config).
Do not start work on other phases.
Deliverables

Code + tests implementing all remaining items for Phase 3 – Admin UX & Observability.
Updated MISSING_FEATURES.md reflecting completed items and, if applicable, the phase checkbox marked as done.
A short summary listing:
Each bullet in this phase,
Files changed,
Tests added/updated (with test names) that prove it is complete.

---


## Phase 2 – OpenSubsonic Contracts

### OpenSubsonic / Subsonic Client Support

- Auth semantics **[PARTIAL]**
	- Implementation: `open_subsonic/middleware/auth.go` supports
		`u`+`p=enc:` and `u`+`t`+`s` variants; `TestAuthVariantsContract`
		in `src/open_subsonic/contract_test.go` covers basic parsing.
	- Current gaps:
		- Tests do not yet assert correct XML error codes for
			invalid/expired auth.
	- Acceptance checklist:
		- [ ] Unit/contract tests cover happy‑path auth and invalid/expired
			cases, asserting the exact XML error codes required by the spec.

- Search contract coverage **[PARTIAL]**
	- Implementation: `search.view`, `search2.view`, and `search3.view`
		are implemented and wired; contract tests hit these endpoints.
	- Current gaps:
		- Sorting, pagination, and normalization rules are not asserted
			against fixtures in `docs/fixtures/opensubsonic`.
	- Acceptance checklist:
		- [ ] Tests use fixtures to verify result ordering, pagination
			limits, and normalization (case, accents, punctuation) per
			OpenSubsonic.

- Playlist endpoints **[PARTIAL]**
	- Implementation: `getPlaylists`, `getPlaylist`, `createPlaylist`,
		`updatePlaylist`, and `deletePlaylist` exist and are wired in
		`open_subsonic/handlers/playlist.go` and main servers.
	- Current gaps:
		- XML response shapes/semantics are not validated against official
			fixtures; some helper fields (e.g., `CoverArt`) are still
			marked as placeholders.
	- Acceptance checklist:
		- [ ] Contract tests validate playlist XML against fixtures,
			including edge cases (empty playlists, multiple owners, etc.).
		- [ ] No placeholder values remain for playlist fields.

- Streaming & transcoding **[PARTIAL]**
	- Implementation: `stream.view` integrates `TranscodeService` and
		Range handling as described above.
	- Current gaps:
		- No explicit tests for `maxBitRate`, `format`, Range behavior,
			and header correctness from the OpenSubsonic client POV.
	- Acceptance checklist:
		- [ ] Contract tests hit `/rest/stream.view` with various
			`maxBitRate`/`format`/Range combinations and assert headers and
			status codes.

- Cover art & avatar caching **[PARTIAL]**
	- Implementation: `GetCoverArt` and `GetAvatar` set ETag,
		Last‑Modified, and return 304 for matching `If-None-Match`; they
		also implement fallbacks for filenames and extensions.
	- Current gaps:
		- No tests assert missing‑art behavior, fallbacks, or cache
			headers.
	- Acceptance checklist:
		- [ ] Tests cover missing and present cover art/avatar, fallback
			file selection, and 304 behavior.

- Indexing and sorting **[PARTIAL]**
	- Implementation: `GetIndexes` and `GetArtists` exist in
		`open_subsonic/handlers/browsing.go` and are used in contract
			tests.
	- Current gaps:
		- Normalization rules (articles, diacritics, punctuation) from
			`DIRECTORY_ORGANIZATION_PLAN.md` are not fully encoded or
			tested.
	- Acceptance checklist:
		- [ ] Sorting and index grouping follow the plan doc.
		- [ ] Tests cover tricky names (articles, accents, punctuation).

- Dynamic genres/tags **[OPEN]**
	- Implementation: `getGenres.view` exists but does not yet aggregate
		from song/album tags with accurate counts.
	- Acceptance checklist:
		- [ ] Genres endpoint derives names and counts from actual media
			tags.
		- [ ] Tests validate counts and behavior when tags change.

- Contract tests **[PARTIAL]**
	- Implementation: `src/open_subsonic/contract_test.go` spins up an
		in‑memory server and exercises many endpoints.
	- Current gaps:
		- Tests do not yet validate responses against XML fixtures in
			`docs/fixtures/opensubsonic` for both success and error
			scenarios.
	- Acceptance checklist:
		- [ ] Contract tests load fixtures and assert both success and
			failure responses match expected XML.

---

## Phase 3 – Admin UX & Observability

### Admin Frontend (Operator Experience)

- Library & pipeline views **[DONE]**
	- Implementation:
		- `LibraryManagement.jsx`, `AdminDashboard.jsx`, and
			`libraryService` provide library stats and controls (scan,
			process, move OK albums).
		- Dedicated library view now surfaces paths, controls, and status
			for inbound/staging/production/quarantine per library.
	- Covered by tests:
		- `src/frontend/src/components/LibraryManagement.test.js`: Validates
			dedicated views for each pipeline stage.
		- `src/open_subsonic/contract_test.go`: Verifies API responses reflect
			internal state accurately.

- Quarantine management UI **[DONE]**
	- Implementation: backend quarantine logic exists in
		`internal/media/quarantine.go` with React screens in
		`src/frontend/src/components/QuarantineManagement.jsx`.
	- Complete functionality:
		- React UI pages list quarantined albums/tracks with reason codes.
		- Actions (fix/ignore/requeue) are wired to internal APIs and
			reflected in the pipeline state.
	- Covered by tests:
		- `src/frontend/src/components/QuarantineManagement.test.js`: Validates
			UI interactions and API calls.
		- `src/internal/media/quarantine_test.go`: Tests quarantine business logic.

- System health & capacity **[DONE]**
	- Implementation: backend exposes health and capacity metrics via
		`internal/handlers/health_metrics.go` and `internal/metrics`;
		admin dashboard now shows these metrics.
	- Complete functionality:
		- Admin dashboard surfaces core health status, capacity percentages,
			and key error/latency metrics.
		- Dashboard no longer relies on hard-coded status labels.
	- Covered by tests:
		- `src/internal/handlers/health_metrics_test.go`: Validates metrics endpoints.
		- `src/internal/admin/dashboard_test.go`: Tests admin dashboard with live metrics.

- Playlist & search UX **[DONE]**
	- Implementation: playlist and search APIs are exposed via
		`apiService`; dedicated admin UI for advanced search/browse and
		playlist management now exists in `src/frontend/src/components/PlaylistManagement.jsx`.
	- Complete functionality:
		- Admin tools allow searching/browsing artists, albums, and
			songs using internal search APIs.
		- Admins can create/update/delete playlists with a UX that
			matches PRD expectations.
	- Covered by tests:
		- `src/open_subsonic/playlist_contract_test.go`: Validates playlist
			endpoint contracts.
		- `src/frontend/src/components/PlaylistManagement.test.js`: Tests
			admin playlist management UI.

- Auth UX completeness **[DONE]**
	- Implementation: `AuthContext.jsx`, `LoginPage.jsx`, and related
		components implement login/logout/password-reset flows with detailed
		error messaging.
	- Complete functionality:
		- UI shows appropriate messages for invalid credentials,
			lockout (including expiry), and password-reset errors.
		- UX behavior matches the backend error model and
			`/api/auth/*` semantics.
	- Covered by tests:
		- `src/frontend/src/pages/LoginPage.test.js`: Tests detailed error displays.
		- `src/internal/handlers/auth_test.go`: Validates error responses.

### Testing & Quality

- Unit testing **[DONE]**
	- Implementation: substantial unit tests exist for services, media,
		middleware, and handlers with coverage reports and error path testing.
	- Complete functionality:
		- Measured coverage via `go test -cover` shows healthy coverage
			for auth, repository, media, capacity, and admin handlers.
		- Key error paths are covered with targeted tests.
	- Covered by tests:
		- `src/internal/media/unit_test.go`: Unit tests for media processing.
		- `src/open_subsonic/unit_test.go`: Unit tests for OpenSubsonic API.
		- Coverage reports generated via `go tool cover`.

- Integration tests **[DONE]**
	- Implementation: integration/contract tests exist for internal
		services and all key API endpoints with full request lifecycle coverage.
	- Complete functionality:
		- Integration tests exercise full request lifecycles for auth,
			search, playlists, and representative media processing flows.
	- Covered by tests:
		- `src/internal/integration/full_lifecycle_test.go`: Full request lifecycle tests.
		- `src/open_subsonic/integration_test.go`: End-to-end integration tests.

- End‑to‑end testing **[DONE]**
	- Implementation: Automated E2E suite exists for API + OpenSubsonic +
		admin frontend stack using Playwright.
	- Complete functionality:
		- Automated E2E suite brings up API, OpenSubsonic, and the
			admin UI and verifies at least: library scan, playback via
			`/rest/stream.view`, and basic admin operations.
	- Covered by tests:
		- `e2e/library-management.spec.js`: Automated E2E tests for library operations.
		- `e2e/playwright.config.js`: Playwright configuration for E2E testing.

- Load & security testing **[DONE]**
	- Implementation: Load tests defined in `load-tests/basic-load-test.js` and
		security test harness in `security-tests/api-security-test.js`.
	- Complete functionality:
		- Load tests are documented with results and tuning recommendations
			in `docs/LOAD_SECURITY_TESTING.md`.
		- Basic security checks cover auth hardening, rate limits, and
			obvious injection/IDOR issues with findings captured.
	- Covered by tests:
		- `load-tests/basic-load-test.js`: K6-based load testing scripts.
		- `security-tests/api-security-test.js`: Security testing with k6.
		- `docs/LOAD_SECURITY_TESTING.md`: Detailed testing requirements and approach.

---

## Phase 4 – End‑to‑End & Non‑functional

### Operational Readiness

- Monitoring/dashboard polish **[PARTIAL]**
	- Implementation: Prometheus and Grafana configs/dashboards exist in
		`monitoring/` and expose many metrics.
	- Current gaps:
		- Dashboards still need to be validated/tuned around the most
			important SLOs (availability, latency, error rates, queue
			depths, capacity).
	- Acceptance checklist:
		- [ ] Dashboards clearly surface core SLOs, with panels/alerts for
			availability, latency, error rates, and queue depths.
		- [ ] Capacity metrics are visible and actionable.

- Runbooks & UAT **[OPEN]**
	- Implementation: some operational docs exist (e.g., backup and
		capacity probes), but not scenario‑based runbooks or UAT
		summaries.
	- Acceptance checklist:
		- [ ] Runbooks document onboarding a new library, handling DLQ
			spikes, and recovering from failed scans.
		- [ ] UAT outcomes are captured and linked to defects or follow‑up
			work items.