# Media Workflow Refactor – Action Plan

_Last updated: November 27, 2025_

## 1. Snapshot Overview
- **Scope**: Complete the song→track terminology migration, finish playlist workflow wiring, and restore the `melodee/internal/services` test suite.
- **Current build status**: Core binaries build, but `GO111MODULE=on go test ./internal/services/...` still fails.
- **Focus areas**: Playlist helpers, auth reset flow, contract tests that instantiate HTTP handlers, and removal of legacy song-specific types (`UserSong`, `AlbumStatus`).

## 2. Key Findings
- [x] Models: `Playlist` now exposes `Tracks []PlaylistTrack`, but downstream helpers remain to be wired (`src/internal/models/models.go`).
- [x] Repository: Added `AddTrackToPlaylist`, `GetPlaylistWithTracks`, and now preload tracks in `GetAlbumByID` (`src/internal/services/repository.go`).
- [x] Auth tests: Removed duplicate `auth_test.go` that conflicted with `auth_service_test.go`.
- [x] Password reset fixtures: Converted bcrypt byte slices to strings before assigning to pointer fields (`auth_service_test.go`).
- [x] Benchmark fixtures: Replaced `models.UserSong` with `models.UserTrack`.
- [ ] Contract tests: Still sit in `package services` and instantiate handler structs directly, causing undefined identifiers once handlers moved packages (`contract_test.go`).

## 3. Outstanding Issues & Risks
[ ] **Services test suite fails**: `contract_test.go` references handler types (`AuthHandler`, `UserHandler`, etc.) that now live under `handlers`. Fix requires moving the test to `package services_test`, importing the handler package, and constructing them via proper constructors.
[ ] **Playlist handler API**: Request payload still names arrays `SongIDs`, but repository/helpers expect tracks. Update handler DTOs + validation to prevent regressions once playlist tests re-enable.
[ ] **Residual song terminology**: Search handler responses still return `"songs"` key for backwards compatibility; confirm OpenSubsonic contracts tolerate this or add dual keys.
[ ] **Auth fixtures**: Contract tests manually mutate `req.Body = bytes.NewBuffer(...)`, which now fails because `*bytes.Buffer` is not an `io.ReadCloser`. Needs `io.NopCloser` wrapper.
[ ] **Health contract**: Local edits revealed unused DB handles and missing handler wiring in the health contract scenario—test currently does nothing.

## 4. Immediate Next Steps
1. **Refactor `contract_test.go`**
   - [ ] Change to `package services_test`.
   - [ ] Import `melodee/internal/handlers`, `melodee/internal/services`, and `melodee/internal/test`.
   - [ ] Instantiate handlers via exported constructors (supplying `nil` where optional).
   - [ ] Wrap manual request bodies with `io.NopCloser(bytes.NewBuffer(...))` to satisfy `io.ReadCloser`.
2. **Playlist handler DTO cleanup**
   - [ ] Rename `SongIDs` → `TrackIDs` in `CreatePlaylist`/`UpdatePlaylist` requests.
   - [ ] Wire the new repository helpers (create `PlaylistTrack` entries + return hydrated playlist via `GetPlaylistWithTracks`).
3. **Regression & integration tests**
   - [ ] Update fixtures to drop `AlbumStatus`/`UserSong` references.
   - [ ] Re-run `GO111MODULE=on go test ./internal/services/...` and capture any residual failures for the blocker log.
4. **Documentation touch-up**
   - [ ] Mirror handler/test changes in `docs/MEDIA_WORKFLOW_REFACTOR.md` once code stabilizes.

## 5. Test Matrix (Current vs Target)
| Test Suite | Current Result | Target |
| --- | --- | --- |
| `GO111MODULE=on go test ./internal/services/...` | 🔴 Fails – contract tests don’t compile | ✅ Pass
| `go test ./internal/handlers/...` | ⚠️ Not yet rerun after handler DTO rename | ✅ Pass post-playlist updates
| `npm test` (frontend) | ⏸ Not part of this refactor | ⏸

## 6. Blocker Log Template
When rate limits allow, capture remaining failures here:
```
### <Date/Time>
- **Command**: <cmd>
- **Result**: <pass/fail>
- **Failure summary**: <stack trace excerpt>
- **Next action**: <owner + fix idea>
```

---
Use this plan as the living checklist while we clear the remaining test failures and finish the media workflow refactor. Update timestamps and tables as progress is made.
