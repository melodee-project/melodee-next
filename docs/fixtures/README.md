# API Fixtures

Authoritative sample requests/responses for OpenSubsonic and internal REST APIs. Use these to keep handlers, clients, and tests aligned with the documented contracts.

## Structure
- `opensubsonic/`: XML fixtures that follow Subsonic 1.16.1 + OpenSubsonic extensions as documented in `../TECHNICAL_SPEC.md`.
- `internal/`: JSON fixtures for REST APIs used by web/admin UI.
- `notes.md`: Any deviations, pending decisions, or edge-case clarifications.

## Naming Convention
- `{endpoint}-{case}.{xml|json}`
- Prefix auth context when relevant: `auth-`, `anon-`, `admin-`.
- Include pagination where relevant, e.g., `getArtists-page2.xml`.

## Required Fields (OpenSubsonic)
- `subsonic-response` root with `status`, `version`, `type`, `serverVersion`, `openSubsonic`.
- On error, `status="failed"` and `<error code="" message=""/>`; keep HTTP 200 as per compatibility.
- Pagination: include `offset` and `size` attributes in list responses.
- Dates: ISO8601 UTC; strings UTF-8 NFC.

## Required Fields (Internal)
- Envelope-less JSON with snake_case fields unless otherwise documented.
- Errors: `{ "error": { "code": "<string>", "message": "<string>", "fields": { ... } } }`.
- Pagination: `{ "data": [...], "pagination": { "offset": 0, "limit": 50, "total": 123 } }`.

## Examples (placeholders)
- `opensubsonic/getArtists-ok.xml`
- `opensubsonic/getSong-not-found.xml`
- `opensubsonic/getArtists-page2.xml`
- `opensubsonic/stream-ok.xml`
- `opensubsonic/stream-error.xml`
- `opensubsonic/search3-ok.xml`
- `opensubsonic/search2-ok.xml`
- `opensubsonic/playlist-get-ok.xml`
- `opensubsonic/coverArt-not-found.xml`
- `opensubsonic/download-not-found.xml`
- `opensubsonic/share-create-ok.xml`
- `opensubsonic/share-delete-ok.xml`
- `opensubsonic/stream-range-example.txt`
- `opensubsonic/playlist-not-found.xml`
- `opensubsonic/avatar-ok.headers`
- `opensubsonic/avatar-not-found.xml`
- `opensubsonic/download-ok.headers`
- `opensubsonic/search-ok.xml`
- `opensubsonic/playlist-create-ok.xml`
- `opensubsonic/playlist-update-ok.xml`
- `internal/auth-login-ok.json`
- `internal/libraries-scan-request.json`
- `internal/libraries-scan-response.json`
- `internal/metadata-update-request.json`
- `internal/metadata-update-response.json`
- `internal/playlist-create-request.json`
- `internal/playlist-create-response.json`
- `internal/user-create-request.json`
- `internal/user-create-response.json`
- `internal/settings-update-request.json`
- `internal/settings-update-response.json`
- `internal/admin-partition-trigger-request.json`
- `internal/admin-partition-trigger-response.json`
- `internal/cover-art-fetch-response.headers`
- `internal/playlist-update-request.json`
- `internal/playlist-update-response.json`
- `internal/playlist-delete-response.json`
- `internal/user-update-request.json`
- `internal/user-update-response.json`
- `internal/user-delete-response.json`
- `internal/search-results-page2.json`
- `internal/jobs-dlq-requeue-request.json`
- `internal/jobs-dlq-requeue-response.json`
- `internal/jobs-dlq-purge-request.json`
- `internal/jobs-dlq-purge-response.json`
- `internal/cover-art-upload-too-large.json`
- `internal/admin-forbidden-response.json`

## Fixtures to Add Next
- OpenSubsonic: avatar upload (if supported).
- Internal: cover art upload success example (binary), error cases for invalid MIME.

## Reference
- Align with OpenSubsonic 1.16.1 (see upstream docs) and the conventions in `../TECHNICAL_SPEC.md`.
