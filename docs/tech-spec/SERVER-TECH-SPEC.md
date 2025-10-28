# Melodee Server Technical Specification (Go + Gin)

Version: 1.0

This document defines the backend server scope: OpenSubsonic-compatible API, first-party extras (HLS signing, replaygain, proposals), job system, security, and operations. It refactors content from OpenSubsonic-Server-Spec into a server-focused spec for the Go API and workers.

---

## 1) Architecture

Pattern: Modular monolith API (Go 1.22+ with Gin) with embedded workers for home‑lab simplicity.

### Components
- API Service (Go + Gin)
  - OpenSubsonic‑compatible endpoints
  - Admin endpoints for management (same‑origin when serving static Admin UI)
  - HLS playlist/signing endpoint
  - WebSockets for now‑playing
- Workers (in‑process by default)
  - Scanner/Tagger (TagLib + Chromaprint; optional beets sidecar)
  - Transcoder (FFmpeg; HLS/progressive; replaygain)
  - Indexer (CDC → FTS): PostgreSQL FTS by default; OpenSearch optional later for advanced search needs
- Data stores (home‑lab defaults)
  - SQLite primary DB; PostgreSQL optional for large libraries
  - In‑process job queue over SQLite tables; Redis optional (cache/rate limits)
  - Filesystem storage for artwork, thumbnails, HLS; S3/MinIO optional
- Edge
  - Optional reverse proxy (Caddy/NGINX) for TLS if exposed beyond LAN; HTTP/3/QUIC support at the proxy if desired
- Observability (optional)
  - Structured logs to stdout; health endpoints; Prometheus metrics if desired

### Repository layout (melodee-api combined)
```
/cmd/
  api/              # HTTP API (Gin) — starts embedded workers by default
  worker-scanner/   # optional separate binary
  worker-transcoder/# optional separate binary
  worker-indexer/   # optional separate binary
/internal/          # domain, repos, services, middleware
/pkg/               # reusable packages (logging, auth, rbac, signing)
/migrations/        # SQL migrations
/deploy/            # docker-compose for home-lab; k8s overlays optional
```

### Build artifacts
- Single container image (api + workers) by default; multi‑image optional

### Local dev & CI
- Makefile tasks (build/test/lint); docker‑compose with only what you need (SQLite in‑container by default)
- CI builds binaries and image; runs unit tests; optional contract tests against OpenAPI

### Admin static export (Mode A)
- The API can serve the Admin UI as static files at the path prefix /admin when MELODEE_EMBED_ADMIN=true.
- Static assets location (inside the container): /admin-static. The server will map /admin → files in /admin-static.
- How to supply assets:
  - Build the Admin UI with Next.js static export (see Admin UI spec) producing an out/ directory.
  - Either bake the files into the image under /admin-static at build time, or bind-mount a host directory to /admin-static:ro.
  - Ensure the Next.js build uses basePath=/admin and output: 'export'.
- Security: same-origin cookie session (HttpOnly; SameSite=strict) and CSRF tokens on state-changing requests; no tokens in JS.

---

## 2) Data model

Tables
- users(id, sub, email, roles[], created_at, ...)
- libraries(id, name, path, policy_normalization, write_back_enabled, ...)
- artists(id, library_id, name, sort_name, mbid, shard, hash, ...)
- albums(id, artist_id, title, sort_title, year, mbid, compilation, ...)
- tracks(id, album_id, artist_id, title, track_no, disc_no, duration, codec, sample_rate, bit_depth, isrc, mbid, fingerprint, file_id, replaygain_track_i, replaygain_album_i, ...)
- files(id, library_id, rel_path, file_hash, size, mtime, tag_raw_json, format, ...)
- artworks(id, kind, object_id, mime, width, height, s3_key, ...)
- playlists(id, user_id, title, is_smart, rules_json, ...)
- scrobbles(id, user_id, track_id, device_id, started_at, finished_at, ...)
- jobs(id, kind, payload_json, state, attempt, last_error, ...)
 - user_favorites(user_id, entity_type, entity_id, created_at)
 - user_ratings(user_id, entity_type, entity_id, rating smallint check 0<=rating and rating<=5, updated_at)
 - user_reactions(user_id, entity_type, entity_id, reaction enum('like','dislike'), updated_at)

Denormalized engagement counters (per entity row)
- artists: likes_count int, reactions_count int, rating_sum int, rating_count int, rating_avg numeric, wilson_score numeric
- albums: likes_count int, reactions_count int, rating_sum int, rating_count int, rating_avg numeric, wilson_score numeric
- tracks: likes_count int, reactions_count int, rating_sum int, rating_count int, rating_avg numeric, wilson_score numeric
Notes
- rating_avg and wilson_score may be generated stored columns in PostgreSQL computed from the base counters; in SQLite, store numeric values and update them in triggers/app.

Indexes
- (library_id, file_hash) unique; artists(sort_name)
- Trigram/GIN on normalized names/titles
- Partition scrobbles monthly; TTL archive policy
 - Unique per-user per-entity on engagement:
   - user_favorites(user_id, entity_type, entity_id) unique
   - user_ratings(user_id, entity_type, entity_id) unique
   - user_reactions(user_id, entity_type, entity_id) unique
 - Optional aggregates (denormalized for speed):
   - artists.favorites_count, likes_count, rating_avg, rating_count
   - albums.favorites_count, likes_count, rating_avg, rating_count
   - tracks.favorites_count, likes_count, rating_avg, rating_count
   - Maintain via triggers or background jobs; tolerate slight staleness.
- Sorting indexes for browse:
  - artists(wilson_score DESC, likes_count DESC, id)
  - albums(wilson_score DESC, likes_count DESC, id)
  - tracks(wilson_score DESC, likes_count DESC, id)

---

## 3) Metadata normalization

Principles
- Non-destructive: store original raw tags alongside canonicalized fields
- Deterministic canonicalization: NFC normalization, whitespace trim, separators, casefolded sort keys
- Assisted authority: beets + MusicBrainz/Discogs when confidence is high

In-server canonicalization (read-only)
- Parse tags (ID3v2.3/2.4, Vorbis, MP4) via TagLib with a pure-Go fallback
- Normalize fields
  - Artists: preserve raw; compute artists_primary[] and artists_featured[] (detect “feat.”/“ft.”)
  - Sort keys: use TSOP/TSOT/TSOA if present; otherwise synthesize (ignore The/An/A with locale rules)
  - Dates: prefer ID3v2.4 TDRC → ISO; fallback TYER+TDAT; store release_date and year
  - Track/Disc: parse n/m; validate ranges
  - ReplayGain/Loudness: RVA2 or REPLAYGAIN_*/R128 → canonical *_i
  - Lyrics: presence flag and sanitized text (lang if available)
- Compute Chromaprint fingerprints and BLAKE3 content hashes

Beets sidecar (optional write-back)
- Flow: detect change → enqueue tag_enrich → compute fp → call beets → propose patch → apply per policy
- Policies
  - Passive: DB only; no file writes
  - Assisted: stage; admin/librarian approves; then write tags and update DB
  - Active: auto-write; rollback on error (backup tags/sidecar .bak)
- Audit log: original → proposed → applied (who/when/tool/version)
- Write policy
  - ID3 v2.4 UTF-8 default (per-library override to v2.3)
  - Strip stale APEv2 if conflicting with ID3 on MP3
  - Embed single primary art ≤ 1.5 MB; store derivatives externally

---

## 4) Filesystem layout normalization

Goals
- Avoid hot-spots; cross-platform safe names; deterministic paths with tolerances

Scheme (artist-level sharding)
- /<AlphaBucket>/<HashShard>/<ArtistSlug>/<AlbumDir>/<TrackFile>
- AlphaBucket: artist_sort first letter; 0-9 → #; non-alnum → _; letters A..Z
- HashShard: first two hex of BLAKE3(artist_sort) → 00..ff
- ArtistSlug: NFC, diacritics removed, spaces → _, reserved chars → _, ≤60 chars
- AlbumDir: (year)_album_slug or album_slug; append _disc{disc_no} for box sets
- TrackFile: {track_no:02d} - {title_slug}.{ext}; fallback hash prefix

Safety & limits
- Max path length ≤ 240 bytes; truncate with _
- Case sensitivity: store canonical path in DB; detect case-insensitive collisions
- Atomic moves; rollback on error
- Disallow external symlinks in library roots

Derived assets
- Artwork S3 keys; HLS: hls/{track_id}/{variant}/segment_<n>.ts with signed short-TTL URLs

---

## 5) Streaming & playback

- Progressive stream for OpenSubsonic clients (on-the-fly transcode)
- First-party HLS preferred with bitrate ladders (64/128/192/256/320 kbps AAC/Opus)
- Passthrough when supported; replaygain applied at transcode time; optional client-side gain

---

## 6) Authentication & authorization

Identity Provider
- Built‑in auth by default (local users with bcrypt). Optional OIDC (Keycloak/Zitadel/Auth0) when enabled.

Clients & flows
- Admin UI (static, same origin): cookie session (HttpOnly; SameSite=strict); CSRF token on POST/PUT/DELETE; no tokens in JS
- Android/Desktop:
  - If OIDC enabled: Authorization Code + PKCE; tokens in secure storage/keychain
  - Otherwise: login to obtain PATs; store PATs securely; revoke as needed
- Legacy clients: PATs scoped/time‑bounded; revocable

Tokens & sessions
- Built‑in sessions for Admin; PATs for clients when OIDC disabled
 - If OIDC enabled: Access JWT 5–10 min; rotating refresh tokens stored server‑side (e.g., Redis) with device binding and optional IP metadata
- TLS via reverse proxy if exposed publicly

Authorization model
- Roles: admin, librarian, user; scopes: library:read/write, playback:stream, playlist:write, admin:*
- Policy checks at API boundary; attribute‑based (per‑library)

Session security
- CSRF tokens for admin; same‑site cookies; CSP nonces; basic rate limits; simple audit log (sign‑ins and sensitive actions)

---

## 7) API surface

- OpenSubsonic: authentication, browse, search, playlists, starred/favorites, now-playing, scrobbling, transcoding params, paging
- First-party extras: HLS signed URLs, replaygain info, fingerprint match, proposals queue, device capabilities
- Admin endpoints: libraries, jobs, proposals approve/reject, transcoder profiles, users/roles, devices, settings, audit, PATs
- Versioning: /v1 base; deprecations with Sunset headers

### User engagement (first-party + OpenSubsonic mapping)
- Favorites
  - First-party: PUT /me/favorites (body: entityType, entityId, favorite=true|false)
  - OpenSubsonic mapping: star/unstar for songId/albumId/artistId
- Ratings (1–5; 0 clears)
  - First-party: PUT /me/ratings (body: entityType, entityId, rating 0..5)
  - OpenSubsonic mapping: setRating (songId/albumId/artistId, rating)
- Like/Dislike
  - First-party: PUT /me/reactions (body: entityType, entityId, reaction: like|dislike|none)
  - OpenSubsonic: no direct equivalent; exposed only to first-party clients
- Lists
  - GET /me/favorites?entityType=track|album|artist (paged)
  - GET /me/reactions?reaction=like|dislike&entityType=...
  - GET /me/ratings?entityType=... (optionally include aggregates)

Security
- Auth via cookie session (Admin in browser) or PAT/OIDC for first-party clients.
- Idempotent and last-write-wins; all endpoints are safe to retry.

### Ranking by likes (denormalized counters + triggers)
Goal
- Return first page of artists/albums/tracks ordered by a robust likes-based score without scanning the entire reactions table.

Score function
- Use Wilson score lower bound for a Bernoulli proportion (likes vs dislikes) to balance ratio and volume.
- Store wilson_score on entity rows; compute from likes_count and reactions_count.

Maintenance strategy
- On reaction upsert/delete in user_reactions, update the affected entity row counters and recompute wilson_score (and rating_avg if relevant) in a single transaction.
- Implement with DB triggers (PostgreSQL) or application-level updates (SQLite) using UPSERT.
- Provide a background backfill job to recalculate counters for integrity checks.

API & queries
- Add sort=likes_score to browse endpoints (/browse/artists, /browse/albums, /browse/tracks).
- Map likes_score to ORDER BY wilson_score DESC, likes_count DESC, id ASC.
- Cache the first page per entityType and sort for 60–300s; invalidate on large shifts is optional.

PostgreSQL implementation notes
- Consider generated stored columns:
  - rating_avg generated as rating_sum::double precision / NULLIF(rating_count,0)
  - wilson_score generated from likes_count and reactions_count via an immutable SQL expression
- Create btree indexes on (wilson_score DESC, id) to accelerate ORDER BY + LIMIT.

SQLite implementation notes
- Store rating_avg and wilson_score explicitly and update them in triggers or the app on each mutation.
- Avoid heavy aggregations at read time for Large profile; rely on denormalized counters.

---

## 8) Job system & workflows

Queues (Redis Streams or NATS)
- scan, tag_enrich, index, transcode

Retry/backoff & idempotency
- Idempotent keys {kind}:{file_id}; circuit breakers

Flows
- Scanner: walk FS → detect by (size, mtime, hash) → parse tags → DB → schedule tag_enrich + index → maybe relayout
- Transcoder: on playback request → choose profile → generate signed playlist/URLs → optional warm segments
- Back-pressure: per-user concurrency limits from transcoder to API

---

## 9) Security & compliance

- Distroless containers; read-only root FS; drop Linux caps; seccomp profiles
- Secrets via Vault/KMS or sealed secrets; no baked secrets
- Logs exclude PII beyond necessity; IP anonymization; DSAR-friendly user data export
 - Optional mTLS between internal components if you split services (not required in the default single‑container home‑lab setup)

---

## 10) Observability & SLOs

- OTel traces/metrics/logs; dashboards in Grafana; Loki logs; alerting
- SLO targets: P50 API < 50 ms cached; P99 < 300 ms; start playback < 1.5 s; scan throughput ≥ 50–100 files/s on NVMe
- Load testing: k6 for browse/search/stream; transcoding soak
- Metadata tests: golden-set fixtures for tricky tags

---

## 11) Migration & backups

- Initial import without moving files; later relayout with dry-run and rollback plan
- DB backups: pgBackRest (weekly full, 5-min WAL); object storage versioning + lifecycle
- Restore drills quarterly

---

## 12) Appendices

Filename sanitization rules
- NFC normalize; strip control chars; map \/:*?"<>| → _; collapse underscores; trim leading/trailing dots/spaces; 60-char segment; path ≤ 240 bytes

Sample admin APIs
- POST /admin/normalization/proposals/{id}:approve
- GET /player/replaygain?trackId=...
- POST /auth/pat
- GET /browse/tree?node=artists&alpha=A

Electron OAuth notes
- Use system browser; register deep link; or loopback 127.0.0.1; exchange code with PKCE verifier

---

## 13) Large profile tuning (~4M tracks)

Database (PostgreSQL 16+)
- Start settings (adjust for your hardware):
  - shared_buffers: 25% RAM (cap around 8 GB)
  - effective_cache_size: 50–75% RAM
  - work_mem: 16–64 MB (increase cautiously; affects per-sort/per-hash)
  - maintenance_work_mem: 1–2 GB during index creation
  - wal_compression = on; checkpoint_timeout = 15min; max_wal_size tuned for your disk
- Indexes:
  - artists(sort_name), albums(sort_title, year)
  - tracks(album_id, disc_no, track_no), tracks(artist_id)
  - GIN/trigram indexes on normalized names and titles for fast search
- Scrobbles: monthly partitions; prune/archival policy to keep DB lean

Cache (optional)
- Add Redis if browse/search latency increases; cache hot artist/album/track lookups and short-lived tokens

Scanner/Tagger
- Concurrency: 4–8 workers depending on disk throughput (NVMe helps)
- Fingerprinting is CPU-bound; throttle to avoid starving the system
- Use inotify/fanotify for deltas; schedule full rescans during off-hours

Transcoder
- Limit concurrent transcodes per user (1–2) to preserve CPU for interactive playback
- Prefer passthrough when client supports source codec; HLS only when necessary

Filesystem & storage
- Keep HLS segments and thumbnails on fast SSD; consider separate disk from library
- Ensure sufficient inotify watches (fs.inotify.max_user_watches)

Throughput guidelines
- Scanner: target 50–100 files/s on NVMe for simple tags; expect lower with Chromaprint enabled
- Indexer: batch CDC updates; avoid per-row index churn during large imports
