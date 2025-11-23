# Testing and Contract Enforcement

Purpose: bind fixtures to automated tests to ensure API contracts remain stable.

## Harness Expectations
- Use contract tests that load fixtures from `docs/fixtures/` and hit local services in CI.
- Tests should be idempotent and not mutate production data; run against ephemeral DB/Redis.

## Fixture → Test Mapping
- OpenSubsonic:
  - `opensubsonic/search-ok.xml`, `search2-ok.xml`, `search3-ok.xml` → search endpoints return expected structure/order.
  - `playlist-*.xml` → playlist CRUD handlers.
  - `coverArt-not-found.xml`, `avatar-not-found.xml`, `download-not-found.xml`, `stream-error.xml` → error handling.
  - `download-ok.headers`, `stream-range-example.txt` → HTTP headers/range support.
  - `share-*.xml` → share endpoints.
- Internal:
  - Auth/login/refresh fixtures → auth endpoints.
  - Playlist/user/settings/admin/job fixtures → CRUD/admin flows.
  - Cover art headers/errors → image serving/upload validation.
  - Search pagination fixtures → pagination shape.

## CI Steps (recommended)
1) Start services with seeded data matching fixture IDs/names.
2) Run contract tests that assert responses match fixtures (structure, required fields).
3) Fail CI on drift; require fixture updates + spec updates for intentional changes.

## Contract Testing
- For any intentional deviations from the upstream Subsonic/OpenSubsonic spec, check `docs/API_DEFINITIONS.md` for documented differences.
- Run OpenSubsonic contract tests: `go test ./src/open_subsonic/... -v` to ensure compatibility.
- Run Melodee API contract tests: `go test ./src/internal/handlers/... -v` to validate internal API contracts.

## Unit vs Contract Tests
- **Unit tests** (`*_test.go` files): Focus on individual handler functions and service logic in isolation. Examples: `src/internal/handlers/user_test.go`, `src/internal/handlers/auth_test.go`.
- **Contract tests** (`*_contract_test.go` files): Validate API responses match documented contracts and fixtures. Examples: `src/open_subsonic/contract_test.go`, `src/internal/handlers/dlq_contract_test.go`.

## How to write new API tests
- For handler unit tests, follow the patterns in `src/internal/handlers/user_test.go` and `src/internal/handlers/playlist_test.go`
- For contract tests, follow the patterns in `src/internal/handlers/dlq_contract_test.go` and `src/open_subsonic/contract_test.go`
- Use the test database helpers from `src/internal/test/test_helpers.go` for database-dependent tests
- Always test both success and failure scenarios, including authentication/authorization checks

## Representative Test Examples by Feature

### Authentication Tests
- `src/internal/handlers/auth_test.go` - Comprehensive auth flow tests with success and failure cases

### User Management Tests
- `src/internal/handlers/user_test.go` - Complete CRUD operations with pagination and role checks

### Playlist Management Tests
- `src/internal/handlers/playlist_test.go` - Full playlist lifecycle with permission validation

### Library and Job Management Tests
- `src/internal/handlers/library_job_test.go` - Library stats, DLQ operations, settings, and shares
- `src/internal/handlers/library_contract_test.go` - Contract compliance for library endpoints

### Image Handling Tests
- `src/internal/handlers/image_test.go` - Avatar upload and retrieval with security validation

### Search Tests
- `src/internal/handlers/search_test.go` - Query parsing, type filtering, and pagination tests

### OpenSubsonic Tests
- `src/open_subsonic/handlers/browsing_test.go` - Music browsing operations
- `src/open_subsonic/handlers/media_test.go` - Streaming, cover art, and media operations
- `src/open_subsonic/handlers/playlist_test.go` - Playlist operations for compatibility API

## Pending Additions
- Add fixtures for avatar/cover upload success/invalid MIME and bind to upload tests once implemented.
