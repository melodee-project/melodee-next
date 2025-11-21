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

## Pending Additions
- Add fixtures for avatar/cover upload success/invalid MIME and bind to upload tests once implemented.
