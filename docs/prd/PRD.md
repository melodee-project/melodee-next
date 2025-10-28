# Product Requirements Document (PRD) — Melodee Platform

This PRD captures platform-wide product intent and success criteria across server and clients.

## 0) Purpose & scope
- Home‑lab media server with OpenSubsonic compatibility and first‑party clients.
- Simple by default: single container (API + workers), built‑in auth, local storage.
- Separate projects exist, but runtime can be a single container to reduce moving parts.

## 1) Personas
- Admin: config, users/roles, libraries, policies, audit.
- Editor: review normalization proposals, quality control.
- User: playback, search, favorites, offline (mobile/desktop).

## 2) High-level features (acceptance)
- Metadata normalization (non-destructive); proposals approve (optional write‑back).
- Filesystem layout normalization; safe moves; small, deterministic paths.
- Streaming: OpenSubsonic progressive; first‑party HLS with signed URLs; replaygain.
- Search/browse/playlists/starred/scrobbling/now‑playing.
- User engagement: rate (1–5), like/dislike, and favorite tracks, albums, and artists.
	- Mapping: favorites ↔ OpenSubsonic star/unstar; ratings ↔ OpenSubsonic setRating; like/dislike via first‑party extras.
	- Acceptance: actions persist immediately, sync across devices, and reflect in browse/search (e.g., favorite badges). Clearing is supported.
- Admin: libraries, jobs, transcoding profiles, users/roles, basic audit.
- Android Auto: browse, voice, background controls (optional but recommended).
- Desktop: media keys, tray mode, offline downloads.

## 3) Performance goals (practical)
- Start playback ~< 1.5s on LAN for HLS 128–192 kbps.
- Scan throughput target: 50–100 files/s on NVMe for simple tags (lower with Chromaprint).
- Browse/search snappy on LAN; add Postgres/Redis only if needed.

## 4) Constraints & non-goals
- Built‑in auth by default (local users + PATs); OIDC optional.
- Non‑goals: federation, enterprise compliance frameworks.

## 5) Release plan (milestones)
- M0: Single‑container API+workers (scan, browse, stream, built‑in auth/PATs).
- M1: Admin basic pages; proposals approve.
- M2: Android/Desktop MVPs (login via PATs, HLS, scrobble, downloads).

## 6) Risks & mitigations
- Large libraries → use sharded FS, Postgres (for Large profile), job back‑pressure.
- Cross‑origin auth → prefer same‑origin (embed Admin static) or simple cookie session; avoid tokens in browser.
