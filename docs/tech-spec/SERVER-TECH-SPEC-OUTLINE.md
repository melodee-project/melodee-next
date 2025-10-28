# Server Technical Specification — Go + Gin API

Outline for the server-side TECH_SPEC; populate with details from OpenSubsonic-Server-Spec.md.

## 1) Architecture
- Go + Gin API service; workers (scanner/tagger, transcoder, indexer).
- Reverse proxy; HLS; websockets; observability.

## 2) Data model
- Tables: users, libraries, artists, albums, tracks, files, artworks, playlists, scrobbles, jobs,
  user_favorites, user_ratings, user_reactions (per-user engagement for tracks/albums/artists).
- Indexes, partitioning, retention.

## 3) Metadata normalization
- In-server canonicalization rules; beets sidecar; write policies; audit log.

## 4) Filesystem layout normalization
- Sharded scheme; safety limits; derived assets.

## 5) Streaming & playback
- OpenSubsonic progressive; first-party HLS; bitrate ladders; replaygain.

## 6) AuthN/Z
- OIDC validation; roles/scopes; device binding; CSRF for admin-facing routes;
	user engagement scopes (read/write) for first-party/OIDC flows.

## 7) API surface
- OpenSubsonic compatibility matrix; admin endpoints; extras namespace.
- User engagement:
	- Favorites ↔ star/unstar (songs/albums/artists)
	- Ratings (0–5) ↔ setRating (0 clears)
	- Like/Dislike via first-party extras (no OpenSubsonic equivalent)

## 8) Job system & workflows
- Queues; idempotency; scanner/transcode flows; back-pressure.

## 9) Security & compliance
- Container hardening; secrets; logging/PII.

## 10) Observability & SLOs
- OTel traces/metrics/logs; dashboards; SLOs.

## 11) Migration & backups
- Import without moves; relayout; DB and object backups; restore drills.

## 12) ADRs & versioning
- API versioning; deprecation; ADR index.
