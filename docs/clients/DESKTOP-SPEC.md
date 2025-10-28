# Desktop Client Spec — Electron (Win/macOS/Linux)

## 1) Architecture & stack
- Electron (main/renderer), TypeScript, React (optional), hls.js for HLS playback.
- Auto-update (electron-builder), code signing/notarization.
- Proxy support for corporate networks; configurable CA store if SSL inspection.

## 2) Auth
- OIDC Authorization Code + PKCE via loopback (preferred) or custom URI scheme.
- Store refresh token in OS keychain (Keychain/DPAPI/libsecret); access token only in-memory.
- Support device binding and revocation.

## 3) Playback & streaming
- HLS via hls.js; progressive fallback if needed.
- Media keys integration across OSes; tray mode background playback.
- Optional client-side replaygain adjustment when using progressive passthrough.

## 3a) User engagement (rate/like/favorite)
- Controls available in track rows and entity detail panes:
	- Favorite toggle (star)
	- Like/Dislike (thumbs up/down; mutually exclusive; second tap clears)
	- Rating (1–5) via hover stars or context menu; 0/clear supported
- State management:
	- Optimistic updates; queue offline mutations and flush on reconnect
	- Resolve conflicts by last-write-wins; show toast on reconciliation
- Discovery:
	- Favorites section per entity type; sort options using rating/like counts

## 4) Offline
- Download manager with concurrency control, rate limiting, checksum verification, and resume.
- Storage quotas and eviction policies; deduplicate by track_id/hash; safe migrations.

## 5) UX & OS integration
- Media keys; tray mode; notifications; configurable global shortcuts.
- Drag-and-drop into playlists; keyboard-first navigation.

## 6) Diagnostics & privacy
- Diagnostics pane with log levels and export; crash reporting opt-in.
- Redact PII; rotate logs; avoid token persistence in plain files.

## 7) Testing
- Unit/Component; E2E (Playwright); OS matrix (Win 10/11, macOS, Ubuntu LTS).
- Validate media keys, auth redirect flows, and auto-update pipeline.

## 8) API contracts used
- OpenSubsonic core; Extras (HLS, replaygain); device registration and revocation endpoints.
- Engagement: /me/favorites, /me/ratings, /me/reactions.
