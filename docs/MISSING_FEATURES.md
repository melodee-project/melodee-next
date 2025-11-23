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

---

## Backend Core

- Auth flows **[PARTIAL]**
	- Implementation: lockout semantics and password‑reset plumbing exist
		in `src/internal/services/auth.go` and related models/handlers.
	- Current gaps:
		- `AuthService` still contains TODO‑style behavior for password
			reset token validation.
		- End‑to‑end tests exist for reset and lockout, but they do not yet
			fully exercise the final production semantics described in
			`TECHNICAL_SPEC.md`.
	- Acceptance checklist:
		- [ ] `ResetPassword` and related flows validate tokens and
			expiry without placeholder errors.
		- [ ] Tests cover successful reset, invalid/expired token, and
			lockout interactions via `/api/auth/*`.
		- [ ] Behavior matches `TECHNICAL_SPEC.md` and
			`INTERNAL_API_ROUTES.md`.

- Error model **[PARTIAL]**
	- Implementation: `utils.SendError` is used by some handlers.
	- Current gaps:
		- Not all handlers consistently use the shared helper for JSON
			errors.
		- Limited tests assert common JSON error shapes.
	- Acceptance checklist:
		- [ ] All public/internal HTTP handlers use a single shared helper
			for JSON error responses.
		- [ ] Tests assert status code + error body shape for common
			failure cases (validation, not‑found, auth, conflict).

- Security middleware **[PARTIAL]**
	- Implementation:
		- `RateLimiterForPublicAPI` and `RateLimiterForAuth` are wired in
			`src/main.go`, `src/api/main.go`, and
			`src/open_subsonic/main.go`.
		- Unit tests exist in `src/internal/middleware/rate_limiter_test.go`.
	- Current gaps:
		- `/api/images/avatar` and related upload endpoints lack hardened
			size/MIME checks with tests.
		- Fixtures under `docs/fixtures/internal` are not yet exercised
			for these uploads.
	- Acceptance checklist:
		- [ ] Avatar/image upload handlers enforce strict size/MIME limits
			and reject invalid content.
		- [ ] Tests cover large payloads, invalid MIME types, and allowed
			cases using existing fixtures.

- Config & validation **[OPEN]**
	- Implementation: FFmpeg config and `FFmpegProcessor`/
		`TranscodeService` are wired in `src/main.go` and
		`src/internal/media`.
	- Current gaps:
		- No explicit startup‑time validation that fails fast when the
			FFmpeg binary is missing/misconfigured or when profiles are
			invalid.
		- No optional validation hooks for external metadata tokens.
	- Acceptance checklist:
		- [ ] Startup returns a clear error when FFmpeg is missing or
			profiles are invalid, with focused unit tests.
		- [ ] External metadata integrations (when enabled) validate
			configured tokens/URLs on startup.

- Repository tests **[PARTIAL]**
	- Implementation: `repository_db_test.go` and
		`repository_pagination_test.go` cover playlists and pagination
		(e.g., `GetPlaylistsWithUser`).
	- Current gaps:
		- Search‑related filters and ordering semantics are not fully
			covered in DB‑backed tests.
	- Acceptance checklist:
		- [ ] DB tests exist for playlist listing with filters,
			pagination, and ordering used by API handlers.
		- [ ] DB tests cover search‑oriented repository queries used by
			OpenSubsonic/search endpoints.

---

## Media Processing Pipeline

- Wire FFmpeg transcoding into OpenSubsonic **[PARTIAL]**
	- Implementation:
		- `FFmpegProcessor`/`TranscodeService` are wired in `src/main.go`
			and injected into `open_subsonic/handlers.MediaHandler`.
		- `MediaHandler.Stream` uses `maxBitRate` and `format` parameters
			and delegates to `transcodeFile`, which calls
			`TranscodeService.TranscodeWithCache`.
		- Range handling and content‑type inference are implemented in
			`handleRangeRequest`/`getContentType`.
	- Current gaps:
		- Some paths still use hard‑coded storage locations and
			"demo"‑style behavior.
		- No focused tests validate transcoding and Range behavior against
			`MEDIA_FILE_PROCESSING.md`.
	- Acceptance checklist:
		- [ ] `Stream` and related code use configured storage paths and
			finalized pipeline behavior (no demo placeholders).
		- [ ] Tests cover `maxBitRate`, `format`, caching/idempotency,
			Range requests, and content‑type correctness.

- Inbound / staging / production exposure **[PARTIAL]**
	- Implementation: internal media/directory code models pipeline
		states; admin APIs and `LibraryManagement.jsx` expose some
		controls and status.
	- Current gaps:
		- Not all pipeline states (inbound, staging, production,
			quarantine) are exposed in a single coherent internal API
			surface tailored for the admin UI.
	- Acceptance checklist:
		- [ ] Internal APIs enumerate pipeline states and basic controls in
			a way that lets the admin UI reflect current status and actions
			for each library.
		- [ ] Admin UI consumes those APIs and clearly surfaces each
			stage.

- Checksum & idempotency **[OPEN]**
	- Implementation: transcoding cache provides idempotent outputs for
		transcodes.
	- Current gaps:
		- No checksum calculation/validation for source media files across
			pipeline stages.
	- Acceptance checklist:
		- [ ] A stable checksum (e.g., SHA‑256) is computed for each media
			file and stored.
		- [ ] Pipeline steps use the checksum to avoid duplicate work and
			to guarantee idempotent processing.

---

## Admin Frontend (Operator Experience)

- Library & pipeline views **[PARTIAL]**
	- Implementation:
		- `LibraryManagement.jsx`, `AdminDashboard.jsx`, and
			`libraryService` provide library stats and controls (scan,
			process, move OK albums).
	- Current gaps:
		- Not all pipeline paths (inbound/staging/production/quarantine)
			are surfaced with the level of detail envisioned in
			`MEDIA_FILE_PROCESSING.md`.
	- Acceptance checklist:
		- [ ] Dedicated library view surfaces paths, controls, and status
			for inbound/staging/production/quarantine per library.
		- [ ] UI reflects underlying internal API state accurately.

- Quarantine management UI **[OPEN]**
	- Implementation: backend quarantine logic exists in
		`internal/media/quarantine.go`.
	- Current gaps:
		- No React screens or services for listing quarantine items,
			showing reason codes, or performing fix/ignore/requeue actions.
	- Acceptance checklist:
		- [ ] UI page(s) list quarantined albums/tracks with reason codes.
		- [ ] Actions (fix/ignore/requeue) are wired to internal APIs and
			reflected in the pipeline state.

- System health & capacity **[PARTIAL]**
	- Implementation: backend exposes health and capacity metrics via
		`internal/handlers/health_metrics.go` and `internal/metrics`.
	- Current gaps:
		- Admin UI does not yet show `/healthz`, capacity probe output,
			or key SLO metrics directly.
	- Acceptance checklist:
		- [ ] Admin dashboard surfaces core health status, capacity
			percentages, and key error/latency metrics.
		- [ ] Dashboard no longer relies on hard‑coded status labels.

- Playlist & search UX **[PARTIAL]**
	- Implementation: playlist and search APIs are exposed via
		`apiService`; backend playlist/search endpoints exist.
	- Current gaps:
		- No dedicated admin‑oriented UI for advanced search/browse flows
			and playlist management as described in `PRD.md`.
	- Acceptance checklist:
		- [ ] Admin tools allow searching/browsing artists, albums, and
			songs using internal search APIs.
		- [ ] Admins can create/update/delete playlists with a UX that
			matches PRD expectations.

- Auth UX completeness **[PARTIAL]**
	- Implementation: `AuthContext.jsx`, `LoginPage.jsx`, and related
		components implement login/logout/password‑reset flows.
	- Current gaps:
		- Lockout states and detailed error mappings from `/api/auth/*`
			are not fully surfaced in the UI.
	- Acceptance checklist:
		- [ ] UI shows appropriate messages for invalid credentials,
			lockout (including expiry), and password‑reset errors.
		- [ ] UX behavior matches the backend error model and
			`/api/auth/*` semantics.

---

## OpenSubsonic / Subsonic Client Support

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

## Testing & Quality

- Unit testing **[PARTIAL]**
	- Implementation: substantial unit tests exist for services, media,
		middleware, and handlers.
	- Current gaps:
		- Coverage for some error paths and edge cases remains limited;
			no coverage reports are tracked here.
	- Acceptance checklist:
		- [ ] Measured coverage (e.g., via `go test -cover`) shows healthy
			coverage for auth, repository, media, capacity, and admin
			handlers.
		- [ ] Key error paths are covered with targeted tests.

- Integration tests **[PARTIAL]**
	- Implementation: integration/contract tests exist for internal
		services and some API endpoints.
	- Current gaps:
		- Not all key flows (auth, search, playlists, media processing
			jobs) are covered end‑to‑end across the monolith.
	- Acceptance checklist:
		- [ ] Integration tests exercise full request lifecycles for auth,
			search, playlists, and representative media processing flows.

- End‑to‑end testing **[OPEN]**
	- Implementation: none for API + OpenSubsonic + admin frontend as a
		stack.
	- Acceptance checklist:
		- [ ] An automated E2E suite brings up API, OpenSubsonic, and the
			admin UI and verifies at least: library scan, playback via
			`/rest/stream.view`, and basic admin operations.

- Load & security testing **[OPEN]**
	- Implementation: not present in this repo (no load‑test scripts or
		security test harnesses).
	- Acceptance checklist:
		- [ ] Load tests are defined and results (plus any tuning) are
			documented.
		- [ ] Basic security checks cover auth hardening, rate limits, and
			obvious injection/IDOR issues, with findings captured.

---

## Operational Readiness

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