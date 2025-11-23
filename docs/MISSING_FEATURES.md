# Remaining Gaps

This document only tracks **current gaps** in the Melodee implementation.
It is intentionally forward‑looking and does not try to describe what is
already built – use `IMPLEMENTATION_GUIDE.md`, `INTERNAL_API_ROUTES.md`,
`TECHNICAL_SPEC.md`, and `MEDIA_FILE_PROCESSING.md` for the full picture
of the implemented system.

Each bullet below should be treated as an actionable work item. When a
gap is fully addressed (including tests), remove the bullet from this
file as part of the change.

---

## Backend Core

- Auth flows:
	- Add focused tests that cover password reset and account lockout
		semantics end‑to‑end (internal services + `/api/auth/*` handlers),
		aligned with `TECHNICAL_SPEC.md` and `INTERNAL_API_ROUTES.md`.
- Error model:
	- Ensure all internal handlers use the shared error helper in
		`src/internal/utils` and add/extend tests that assert the JSON
		error shape for common failure scenarios.
- Security middleware:
	- Verify rate‑limiting/IP throttling middleware from
		`src/internal/middleware` is wired for public APIs in the main
		application entrypoint.
	- Strengthen size and MIME checks for `/api/images/avatar` (and
		related upload endpoints) and add tests that exercise the fixtures
		under `docs/fixtures/internal`.
- Config & validation:
	- Extend config validation in `src/internal/config` so startup fails
		fast on invalid/missing FFmpeg binary or profiles, with explicit
		tests for these conditions.
	- Add optional validation hooks for external metadata service tokens
		if/when those integrations are enabled.
- Repository tests:
	- Add real DB‑backed tests for `src/internal/services/repository.go`
		that exercise filters, pagination, and ordering used by search and
		playlist endpoints.

---

## Media Processing Pipeline

- Wire FFmpeg transcoding into OpenSubsonic:
	- Replace the remaining placeholder `transcodeFile` logic in
		`src/open_subsonic/handlers/media.go` with the real
		`media.TranscodeService`/`FFmpegProcessor` pipeline (profiles,
		`maxBitRate`, formats, caching, idempotent outputs).
	- Add tests and fixtures that verify transcoding behavior per
		`MEDIA_FILE_PROCESSING.md` (including Range handling and
		content‑type correctness).
- Inbound / staging / production exposure:
	- Expose inbound, staging, production, and quarantine state (and
		basic controls) through internal APIs so the admin UI can reflect
		pipeline status and operations.
- Checksum & idempotency:
	- Implement checksum calculation/validation for media files and use
		it to enforce idempotent processing across the pipeline stages.

---

## Admin Frontend (Operator Experience)

- Library & pipeline views:
	- Ensure there is a dedicated view in `src/frontend` for libraries
		that surfaces inbound/staging/production paths, scan/process/
		promote controls, and current pipeline status.
- Quarantine management UI:
	- Add screens to list quarantine items (albums/tracks), show reason
		codes from `internal/media/quarantine.go`, and provide actions
		(fix/ignore/requeue) mapped to the corresponding internal APIs.
- System health & capacity:
	- Update the admin dashboard to surface health and capacity probe
		data (from `/healthz`, metrics, and capacity endpoints) instead of
		hard‑coded status.
- Playlist & search UX:
	- Provide admin‑oriented tools for searching/browsing artists,
		albums, and songs (using internal search APIs) and managing
		playlists as described in `PRD.md`.
- Auth UX completeness:
	- Ensure the login/logout/password‑reset/lockout UX in the React
		app exactly matches the behavior of `/api/auth/*` endpoints,
		including error states.

---

## OpenSubsonic / Subsonic Client Support

- Auth semantics:
	- Confirm and implement all supported auth variants (`u`+`p`/`enc:`,
		`u`+`t`+`s`) in `open_subsonic/middleware/auth.go` and add tests
		that verify correct XML error codes for expired/invalid auth.
- Search contract coverage:
	- Complete and test `search.view`, `search2.view`, and `search3.view`
		in `src/open_subsonic/handlers/search.go` for proper sorting,
		pagination, and normalization, using fixtures in
		`docs/fixtures/opensubsonic`.
- Playlist endpoints:
	- Finish `getPlaylists`, `getPlaylist`, `createPlaylist`,
		`updatePlaylist`, and `deletePlaylist` in
		`src/open_subsonic/handlers/playlist.go` so their XML shapes and
		semantics match the OpenSubsonic spec and playlists fixtures.
- Streaming & transcoding:
	- Fully integrate the FFmpeg transcoding/caching pipeline into
		`stream.view`, with tests for `maxBitRate`, `format`, Range
		behavior, and correct headers.
- Cover art & avatar caching:
	- Implement/verify ETag, Last‑Modified, and 304 behavior for
		`getCoverArt` and `getAvatar`, and add tests for missing art,
		fallbacks, and header handling.
- Indexing and sorting:
	- Expand normalization and sort rules for `getIndexes` and
		`getArtists` to match `DIRECTORY_ORGANIZATION_PLAN.md` and
		OpenSubsonic expectations; add edge‑case tests (articles,
		diacritics, punctuation).
- Dynamic genres/tags:
	- Replace any hard‑coded genre responses with aggregation from
		song/album tags, returning accurate counts.
- Contract tests:
	- Implement `src/open_subsonic/contract_test.go` as real contract
		tests that spin up an in‑memory server and validate XML responses
		against fixtures in `docs/fixtures/opensubsonic` for both success
		and error scenarios.

---

## Testing & Quality

- Unit testing:
	- Increase coverage for internal services (auth, repository, media,
		capacity, admin handlers) beyond basic compilation/placeholder
		tests; focus especially on error paths and edge cases.
- Integration tests:
	- Add integration tests that cover the full request lifecycle for
		key flows (auth, search, playlists, media processing jobs) across
		the monolith.
- End‑to‑end testing:
	- Add a small E2E suite that brings up the stack (API + OpenSubsonic
		+ admin frontend) and verifies a few representative scenarios
		(library scan, playback via `/rest/stream.view`, basic admin
		operations).
- Load & security testing:
	- Perform load tests against core endpoints and document any
		required performance tuning.
	- Add basic security testing (auth hardening checks, rate limit
		verification, obvious injection/IDOR checks) and capture findings.

---

## Operational Readiness

- Monitoring/dashboard polish:
	- Tighten Prometheus and Grafana dashboards in `monitoring/` so they
		expose the most important SLOs (availability, latency, error
		rates, queue depths, capacity).
- Runbooks & UAT:
	- Add brief runbooks for common operational tasks (onboarding new
		library, handling DLQ spikes, recovering from failed scans).
	- Capture outcomes from any user acceptance testing and link them
		back to specific defects or follow‑up work items.