# OpenSubsonic-Compatible Media Server — Technical Specification
**Version:** 1.0  
**Date:** 2025-09-06 (UTC)  
**Scope:** Server, mobile, and desktop system capable of serving “dozens of users / millions of tracks,” fully compatible with OpenSubsonic clients and providing a first-class Flutter Android app with Android Auto support. Includes metadata normalization (with beets), filesystem layout normalization, and unified authentication/authorization across Web Admin, Flutter Android, and Electron Desktop clients.

---

## 0) Goals & Non‑Goals
**Goals**
- Predictable performance and stability at multi‑million track scale.
- Accurate, consistent metadata via **non-destructive** normalization and optional **beets**-assisted retagging.
- Clean, scalable directory/filename layout preventing FS hot-spots.
- Full Android Auto experience: media browsing, playback, voice (“Play X by Y”), and background controls.
- Unified **AuthN/Z** for web (admin), Flutter Android, and Electron desktop players, using **OIDC**.
- OpenSubsonic compliance (baseline + optional extensions), plus HLS streaming for the first-party apps.

**Non‑Goals**
- Federated social/discovery features (out of scope).
- P2P distribution (out of scope).

---

## 1) Architecture Overview
**Pattern:** Modular monolith (Go) + dedicated workers (scanner/tagger, transcoder, indexer).  
**Core components**
- **API/Web (Go 1.22+)** — OpenSubsonic endpoints, admin UI (BFF), HLS playlist/signing, websockets for now‑playing.
- **Scanner/Tagger Worker (Go + TagLib + Chromaprint)** — fast ingest, fingerprinting, beets integration (optional write-back).
- **Transcoder Worker (Go + FFmpeg)** — on-demand HLS and progressive streams; replaygain support.
- **Indexer Worker** — change-data-capture from DB → search (PostgreSQL FTS initially; OpenSearch optional).
- **PostgreSQL 16+** — primary metadata store; partitioned append tables (e.g., scrobbles).
- **Redis** — cache (hot metadata), rate limits, short-lived tokens, job coordination.
- **Object Storage (S3/MinIO)** — artwork, thumbnails, HLS segments; originals may remain on ZFS.
- **Reverse proxy (Caddy/NGINX)** — TLS termination, HTTP/3, static/HLS serving.

**Observability & Ops**
- OpenTelemetry → Prometheus/Grafana; Loki for logs; CI: SBOM/signing; backups: pgBackRest PITR, object versioning.

---

## 2) Data Model (high level)
- `users(id, sub, email, roles[], created_at, ...)`
- `libraries(id, name, path, policy_normalization, write_back_enabled, ...)`
- `artists(id, library_id, name, sort_name, mbid, shard, hash, ...)`
- `albums(id, artist_id, title, sort_title, year, mbid, compilation, ...)`
- `tracks(id, album_id, artist_id, title, track_no, disc_no, duration, codec, sample_rate, bit_depth, isrc, mbid, fingerprint, file_id, replaygain_track_i, replaygain_album_i, ...)`
- `files(id, library_id, rel_path, file_hash, size, mtime, tag_raw_json, format, ...)`
- `artworks(id, kind, object_id, mime, width, height, s3_key, ...)`
- `playlists(id, user_id, title, is_smart, rules_json, ...)`
- `scrobbles(id, user_id, track_id, device_id, started_at, finished_at, ...)`
- `jobs(id, kind, payload_json, state, attempt, last_error, ... )`

**Indexes**
- `(library_id, file_hash)` unique; `artists(sort_name)`; trigram/GIN on normalized names and titles.
- Partition `scrobbles` monthly; TTL archive policy.

---

## 3) Metadata Normalization (with beets integration)
### 3.1 Principles
- **Non‑destructive**: always store original raw tags alongside canonicalized fields.
- **Deterministic canonicalization**: NFC normalization, whitespace trim, consistent separators, casefolded sort keys.
- **Assisted authority**: use **beets** + MusicBrainz/Discogs for authoritative matches when confidence is high.

### 3.2 In‑server Canonicalization (Read‑Only)
- Parse tags (ID3v2.3/2.4, Vorbis, MP4) via TagLib with a pure-Go fallback for resilience.
- Normalize:
  - **Artists**: preserve raw; compute `artists_primary[]` and `artists_featured[]` (detect “feat.”/“ft.” patterns).
  - **Sort keys**: use `TSOP/TSOT/TSOA` if present; otherwise synthesize (ignore articles “The/An/A” with locale rules).
  - **Dates**: prefer ID3v2.4 `TDRC` → ISO date; fall back to `TYER`+`TDAT`. Store `release_date` and `year`.
  - **Track/Disc**: parse `n/m` into ints; validate ranges.
  - **ReplayGain/Loudness**: read RVA2 or REPLAYGAIN_* / R128; store canonical `*_i`.
  - **Lyrics**: store presence flag and sanitized text (language code if available).
- Compute fingerprints (Chromaprint) and BLAKE3 content hashes.

### 3.3 beets Sidecar (Optional Write‑Back)
**Flow**
1. **Detect change** → enqueue `tag_enrich` (file_id list).  
2. Worker computes Chromaprint (if missing) and calls beets (container) with MusicBrainz/Discogs.  
3. On **high-confidence match**, prepare a **proposed patch** (diff of tag fields).  
4. Apply according to library policy:
   - **Passive**: accept into DB only; do not modify files.
   - **Assisted**: stage patch → admin/librarian must **approve**; then write tags via beets and update DB.
   - **Active**: auto-write tags with rollback on error (backup original tags/sidecar `.bak`).

**Audit log**: original → proposed → applied (who/when/tool/version).

**beets config (starter)**:
```yaml
directory: /music    # destination (if using beet import/move)
library: /data/beets.db
import:
  move: no           # server owns file layout; beets does not move by default
  write: yes         # only for Assisted/Active
plugins: chroma discogs mbsubmit ftintitle edit scrub
ui:
  color: yes
match:
  preferred:
    countries: ['US', 'GB']
    media: ['CD', 'Digital Media']
  strong_rec_threshold: 0.15
paths:
  default: $albumartist_sort/$album%aunique{}/$track - $title
clutter: ['Thumbs.db', '.DS_Store']
```

**Write policy**
- **Write ID3 v2.4 UTF‑8** by default (per‑library override to v2.3 if legacy devices demand).
- Strip stale APEv2 if conflicting with ID3 on MP3.
- Limit embedded art to a single primary image ≤ 1.5 MB; store derivatives externally.

---

## 4) Filesystem Layout Normalization
### 4.1 Goals
- Avoid tens of thousands of entries in any directory.
- Cross‑platform safe filenames (Windows/macOS/Linux).
- Stable, deterministic paths derived from canonical metadata (but tolerate missing data).

### 4.2 Sharded Directory Scheme
**Artist-level sharding**: `/<AlphaBucket>/<HashShard>/<ArtistSlug>/<AlbumDir>/<TrackFile>`

- **AlphaBucket**: first letter of **artist_sort**; map `0-9` to `#`, non‑alnum to `_`, letters to `A..Z`.
- **HashShard**: first two hex chars of BLAKE3(artist_sort) → `00..ff` (256 subdirs per letter bucket).
- **ArtistSlug**: human-friendly, NFC-normalized, diacritics removed, spaces as `_`; reserved chars replaced (`\/:\*?"<>|` → `_`); trim to 60 chars.
- **AlbumDir**: `({year})_{album_slug}` or `{album_slug}` if year unknown; append disc suffix for box sets: `_disc{disc_no}` when needed.
- **TrackFile**: `{track_no:02d} - {title_slug}.{ext}`; if unknown, fallback to file hash prefix.

**Example**
```
/A/9f/AC_DC/(1980)_Back_In_Black/01 - Hells_Bells.flac
/T/2a/The_Beatles/(1966)_Revolver/07 - Eleanor_Rigby.flac
/#/0a/808_State/(1989)_Ninety/02 - Pacific_202.flac
/_/e7/¡Forward_Russia!/(2006)_Give_Me_A_Wall/01 - Thirteen.mp3
```

### 4.3 Safety & Limits
- **Max path length**: target ≤ 240 bytes (reserve for OS limits). Truncate slug segments with ellipsis `_`.
- **Case sensitivity**: store canonical path in DB; detect collisions on case-insensitive FS.
- **Atomic moves**: temp path → rename; rollback on error.
- **Symlink policy**: disallow external symlinks in library roots.

### 4.4 Derived Assets
- **Artwork**: S3 key `art/a/{alpha}/{hash2}/{artist_id}/{size}.webp` … similar for albums.
- **HLS**: `hls/{track_id}/{variant}/segment_<n>.ts` with short‑TTL signed URLs.

---

## 5) Streaming & Playback
- Progressive `stream` endpoints for OpenSubsonic clients (with on‑the‑fly transcode).
- First‑party apps prefer **HLS** with bitrate ladders (64/128/192/256/320 kbps AAC or Opus); passthrough for supported formats.
- ReplayGain applied at transcode time; optional client‑side gain.

---

## 6) Android Auto — Full Capability (Flutter App)
### 6.1 Overview
- **Flutter UI** powered by `just_audio` for ExoPlayer-backed playback and `audio_service` for background service & MediaSession.
- Optional thin **native shim** using **Jetpack Media3** (`MediaLibraryService`) for richer browse nodes; exposed to Dart via MethodChannels/Ffi.

### 6.2 Required Capabilities
- **Background playback** with **MediaSession** (lockscreen, notifications, headset buttons).
- **Android Auto browse** via MediaLibrary tree: Library → Artists → Albums → Tracks, Playlists, Recent.
- **Voice**: implement `onPlayFromSearch` & `onSearch`:
  - Map utterances (“Play *Song* by *Artist*”, “Play *Album*”) to OpenSubsonic search.
  - Provide **App Actions** (BII for Media) in `shortcuts.xml` so Assistant routes queries.
- **Audio focus & ducking**; **resume from last position**; **download manager** for offline playback.

### 6.3 Network Integration
- Use **short‑lived signed URLs** for HLS segments; automatic refresh via API when near expiry.
- Device profile chooses default ladder (e.g., 128–192 kbps on cellular; higher on Wi‑Fi).

### 6.4 Testing Matrix
- Head units: Android Auto emulator + representative OEM units.
- Scenarios: voice play, browse deep trees, poor connectivity, loss/reconnect, phone calls, nav prompts, BT headset controls.

---

## 7) Unified Authentication & Authorization
### 7.1 Identity Provider
- OIDC provider (e.g., **Keycloak**, Auth0, or cloud IdP).
- **Users** managed in IdP; roles mirrored into app DB via claims mapping.

### 7.2 Clients & Flows
- **Web Admin (SPA + BFF)**: Authorization Code with PKCE between BFF and IdP. The browser receives **HttpOnly, SameSite=strict** session cookies only; **no access tokens stored in JS**.
- **Flutter Android App**: Authorization Code + PKCE using system browser/Custom Tabs; tokens stored in OS‑secure storage (Keystore). Refresh tokens are **rotating** and bound to device.
- **Electron Desktop** (Win/macOS/Linux): System browser flow with **custom URI scheme** `com.acme.player://oauth2redirect` (or loopback 127.0.0.1). Store secrets in platform keychain (Keychain/DPAPI/libsecret).
- **Legacy OpenSubsonic Clients**: Personal Access Tokens (PAT) scoped and time‑bounded; can be revoked. PATs map to the same roles/scopes.

### 7.3 Tokens & Sessions
- **Access tokens**: JWT, 5–10 min; include `sub`, `sid`, `device_id`, `roles`, `scopes`, `tenant` (if multi‑tenant).
- **Refresh tokens**: opaque, rotation, one‑time use, stored server‑side (Redis) with device binding and IP metadata.
- **mTLS (internal)**: between server components; public endpoints TLS 1.2+ with HSTS.

### 7.4 Authorization Model
- Roles: `admin`, `librarian`, `user`.  
- Scopes (examples): `library:read`, `library:write`, `playback:stream`, `playlist:write`, `admin:*`.
- Policy checks at API boundary; attribute-based rules (e.g., per-library access).

### 7.5 Session Security
- BFF sets CSRF tokens for admin UI; same‑site cookies; CSP nonces; rate limits on auth endpoints.
- Device limits per user; revoke by device.
- Audit trails for sign-ins and sensitive actions.

---

## 8) OpenSubsonic & First‑Party API
- Strict compatibility for: authentication, browsing, search, playlists, starred/favorites, now-playing, scrobbling, transcoding params, paging.
- **HLS** endpoint for first‑party apps with signed URLs; classic `stream` for third‑party clients.
- **Extras (namespaced)**: replaygain info, fingerprint match status, proposals queue (normalization), device capabilities.

---

## 9) Job System & Workflows
- **Queues** (Redis Streams/NATS): `scan`, `tag_enrich`, `index`, `transcode`.
- **Retry & backoff** with circuit breakers; idempotent jobs keyed by `{kind}:{file_id}`.
- **Back-pressure** from transcoder to API (per-user concurrency limits).

**Scanner Flow**
1. Walk filesystem (watch inotify/fanotify for deltas).  
2. Detect new/changed file by `(size, mtime, hash)`.  
3. Parse tags → DB row; schedule `tag_enrich` + `index`.  
4. If library policy moves files, schedule **relayout** to normalized path.

**Transcode Flow**
- On playback request: evaluate client capability → choose profile (passthrough or HLS ladder).  
- Generate signed playlist/URLs; enqueue warm segments optionally.

---

## 10) Security & Compliance
- Distroless containers; read‑only root FS; drop Linux caps; seccomp profiles.
- Secrets via Vault/KMS or sealed secrets; no secrets baked into images.
- Logs exclude PII beyond necessity; IP anonymization for analytics; DSAR‑friendly export of user data.

---

## 11) Testing & SLOs
- **SLOs**: P50 API < 50 ms cached, P99 < 300 ms; start playback < 1.5 s; scan throughput ≥ 50–100 files/s on NVMe.
- **Load**: k6 scenarios for browse/search/stream; soak transcoding with mixed bitrates.
- **Metadata**: golden‑set fixtures covering tricky tags (ID3 2.3 vs 2.4, feat., multi‑disc, huge embedded art).
- **Android Auto**: scripted utterances + UI automation; tunnel poor‑network conditions.

---

## 12) Migration & Backups
- Initial import from existing libraries without moving files; later **relayout** with dry‑run and rollback plan.
- DB: pgBackRest (weekly full, 5‑min WAL); object storage versioning + lifecycle policies.
- **Restore drills** quarterly.

---

## 13) Appendix
### 13.1 Filename Sanitization Rules
- Normalize to NFC; strip control chars; map `\/:\*?"<>|` → `_`; collapse repeated `_`.
- Trim leading/trailing dots/spaces; enforce 60‑char max segment; path ≤ 240 bytes.

### 13.2 Sample API (Extras)
- `POST /admin/normalization/proposals/{{id}}:approve`
- `GET /player/replaygain?trackId=...`
- `POST /auth/pat` (create scoped PAT for legacy clients)
- `GET /browse/tree?node=artists&alpha=A`

### 13.3 Electron OAuth Notes
- Use system browser; register deep link; on success, app exchanges code with PKCE verifier (no embedded secrets).

---

**End of Spec**
