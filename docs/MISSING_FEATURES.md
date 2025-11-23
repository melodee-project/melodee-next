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

Phase checklist (outstanding only):

- [ ] **Phase 2 – OpenSubsonic Contracts**

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
			ags.
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