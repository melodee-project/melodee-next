# Android Client Spec — Flutter (Android Auto)

## 1) Architecture & stack
- Flutter; just_audio + audio_service; optional native shim with Jetpack Media3 (MediaLibraryService) for richer browse/voice.
- State: minimal app state; query-first with caching; background service for playback.

## 2) Auth
- OIDC Authorization Code + PKCE via Custom Tabs.
- Access token 5–10 min; rotating refresh token bound to device.
- Secure storage: Android Keystore (flutter_secure_storage); no tokens in plaintext logs.

## 3) Playback & streaming
- HLS playback with signed short-TTL URLs; preemptive refresh before expiry threshold.
- Ladder selection by network profile and device capability (AAC vs Opus); switch ladders on bandwidth changes.
- Replaygain applied server-side when transcoding; expose client-side toggle only for passthrough.

## 4) Browse, search, and voice
- OpenSubsonic browse (artists → albums → tracks), playlists, recent.
- Search maps to API; highlight exact/partial matches.
- Android Auto
	- MediaLibrary tree nodes for Library, Artists, Albums, Tracks, Playlists, Recent
	- Implement onPlayFromSearch/onSearch to support "Play Song by Artist" and "Play Album"
	- App Actions (BII for Media) in shortcuts.xml to route Assistant queries

## 4a) User engagement (rate/like/favorite)
- Per item (track/album/artist):
	- Favorite toggle (star)
	- Like/Dislike toggle (thumbs up/down; mutually exclusive; second tap clears)
	- Rating control (1–5; long-press or overflow menu); 0/clear supported
- Sync behavior:
	- Actions send immediately to /me/* endpoints; queue when offline and replay with ordering on reconnect
	- Local optimistic UI with conflict resolution: last-write-wins by updatedAt
- Surfaces:
	- Badges in lists and detail pages (favorite icon; like/dislike state; average rating where shown)
	- Filters: Favorites-only views per entity type

## 5) Offline
- Download manager with queueing, network constraints (Wi‑Fi only option), concurrency, and storage quotas.
- Resume partial downloads; verify integrity (BLAKE3/hash if provided); handle eviction policies.

## 6) Now-playing & scrobbling
- Now-playing updates by WebSocket with polling fallback; handle reconnections.
- Scrobble rules: after 50% or 4 minutes; on stop; queue when offline and flush on reconnect.

## 7) Error handling & resilience
- Token refresh with jittered backoff; reauth prompt only after refresh exhaustion.
- Network loss: stall → retry; downgrade ladder; cache small lookahead segments when possible.

## 8) Testing matrix
- Devices: mid/low/high-tier phones; Android Auto emulator + representative OEM head units.
- Scenarios: voice playback, deep browsing, poor connectivity, loss/reconnect, phone calls, nav prompts, BT headset controls.

## 9) API contracts used
- OpenSubsonic: auth, browse, search, playlists, starred, scrobble, now-playing, stream.
- Extras: HLS playlist/signing; replaygain info; device capabilities; user engagement (/me/favorites, /me/ratings, /me/reactions).
