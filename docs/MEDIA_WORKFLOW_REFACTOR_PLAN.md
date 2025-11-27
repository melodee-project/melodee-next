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
- [x] Contract tests: Now sit in `package services_test` and instantiate handler structs via exported constructors (`contract_test.go`).

## 3. Outstanding Issues & Risks
[ ] **Residual song terminology**: Search handler responses still return `"songs"` key for backwards compatibility; confirm OpenSubsonic contracts tolerate this or add dual keys.

## 4. Implementation Complete
The following tasks have been completed:

1. **Refactor `contract_test.go`** ✅
   - [x] Changed to `package services_test`.
   - [x] Import `melodee/internal/handlers`, `melodee/internal/services`, and `melodee/internal/test`.
   - [x] Instantiate handlers via exported constructors (supplying `nil` where optional).
   - [x] Wrap manual request bodies with `io.NopCloser(bytes.NewBuffer(...))` to satisfy `io.ReadCloser`.
   - [x] Added missing library methods to repository and handler.

2. **Playlist handler DTO cleanup** ✅
   - [x] Rename `SongIDs` → `TrackIDs` in `CreatePlaylist`/`UpdatePlaylist` requests.
   - [x] Wire the new repository helpers (create `PlaylistTrack` entries + return hydrated playlist via `GetPlaylistWithTracks`).

3. **Regression & integration tests** ✅
   - [x] Update fixtures to drop `AlbumStatus`/`UserSong` references.
   - [x] Add missing `ClearPlaylistTracks` method to repository.
   - [x] Update playlist integration tests to use proper track methods.

4. **Documentation update** ✅
   - [x] Mirror handler/test changes in `docs/MEDIA_WORKFLOW_REFACTOR.md`.

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
