# Melodee Platform Blueprint (Home‑Lab Edition)

This document defines the platform split across multiple repos with a shared auth mechanism and shared API models. It serves as the high-level source of truth for cross-repo conventions and release coordination.

---

## 1) Service and repo catalog

Recommended GitHub org: melodee-project

- melodee-specs (optional for small installs)
  - Ownership of API contracts: OpenAPI (YAML), JSON Schema, codegen configs
  - Generated SDKs (packages):
    - melodee-sdk-go (models + server stubs)
    - melodee-sdk-ts (browser/server clients)
    - melodee-sdk-dart (Dart client)
  - Versioned releases (SemVer), published to registries (GH Packages or public)
- melodee-api (Go + Gin + embedded workers)
  - Single repo and single image by default: API + scanner + transcoder + indexer
  - Built-in auth (local users, bcrypt passwords, optional TOTP, PATs). OIDC is optional (advanced users).
  - Minimal dependencies: SQLite (default) or PostgreSQL (optional); filesystem storage for artwork and HLS by default; S3/MinIO optional.
  - Admin endpoints exposed for Admin UI.
  - Suggested layout:
    - /cmd/api (starts background workers in-process by default)
    - /internal, /pkg
    - /deploy (docker-compose for single-container home lab)
- melodee-admin (Next.js + shadcn/ui + Tailwind)
  - Admin BFF and UI; server-side session; no tokens in the browser
  - Consumes melodee-sdk-ts on the server only
- melodee-android (Flutter)
  - Android + Android Auto client; HLS playback; offline downloads; voice search
  - Consumes melodee-sdk-dart
- melodee-desktop (Electron)
  - Desktop client (Win/macOS/Linux); HLS playback; media keys; offline
  - Consumes melodee-sdk-ts

Notes:
- Home‑lab default is simple: one container (API + workers) and local storage. Contracts repo (melodee-specs) is helpful for growth but not required to self-host.

---

## 2) Auth and identity (simple by default)

- Default: built-in auth with local users (bcrypt), roles (admin/editor/user), and Personal Access Tokens (PATs).
- Optional: enable OIDC (Keycloak/Zitadel/Auth0) if you want SSO across multiple home services.
- Admin UI: uses server-side session cookie (HttpOnly; SameSite=strict) via Next.js BFF.
- Mobile/Desktop: Authorization Code + PKCE when OIDC enabled; otherwise username/password login to obtain PATs.
  - If OIDC is enabled, include scopes for user engagement read/write; PATs imply user context for /me/* endpoints.

---

## 3) Shared API models and contracts

Source of truth: melodee-specs

- OpenAPI for:
  - OpenSubsonic-compatible endpoints
  - Admin endpoints (namespaced under /admin)
  - Extras namespace (replaygain, device capabilities, proposals, user engagement)
- JSON Schemas for complex payloads (e.g., proposal diffs)
- Codegen rules:
  - Go models and server stubs for melodee-api
  - TypeScript client for melodee-admin (server-side only)
  - Dart client for melodee-android
  - Include SDK surfaces for engagement: /me/favorites, /me/ratings, /me/reactions
- Versioning:
  - SemVer per melodee-specs release; clients depend on ^MAJOR.MINOR (pin MINOR for stability during a release train)
  - API versioning strategy: path or header version for breaking changes (e.g., /v1), deprecations with sunset headers

---

## 4) Domains, networking, and CORS

- Home‑lab defaults:
  - Single host or LAN: http(s)://melodee.local for API and serve Admin UI statically from API if desired.
  - Optional split domains: api.local and admin.local (CORS simplified since Admin BFF calls API server-side).
  - No mTLS required. Use TLS via reverse proxy if exposing to the internet.

---

## 5) Build, release, and environments

- Simple path: prebuilt Docker image for melodee-api (includes workers). Run with docker-compose on a single host.
- Admin UI can be served as static export by melodee-api for a single-container install, or run as a separate container if preferred.
- Releases published as versioned images; no complex release train required for home‑lab.
- Contracts (OpenAPI) optional; useful if you build custom clients.

---

## 6) Observability

- Defaults: structured logs to stdout; basic health endpoints; optional Prometheus metrics endpoint.
- Optional: add Grafana/Prometheus stack if you already run it in your lab.

---

## 7) Security

- Use strong admin password; optional TOTP 2FA for admin user(s).
- If exposing to the internet: enable TLS via reverse proxy (Caddy/NGINX) and set HSTS.
- Short‑lived signed HLS URLs by default; PATs for legacy clients.

---

## 8) Documentation split (this repo vs others)

- PRD: docs/prd/PRD.md (platform-wide product intent)
- Server TECH_SPEC: docs/tech-spec/ (Go API + workers)
- Client specs: docs/clients/ (admin-ui.md, android.md, desktop.md)
- Compliance: docs/compliance/open-subsonic.md
- ADRs: docs/adr/

---

## 9) Initial milestones

- M0: Single-container melodee-api with scan, browse, stream (HLS), simple auth and PATs.
- M1: Admin UI basic pages (Libraries, Jobs, Users); proposals approve.
- M2: Android/desktop MVPs (login/PATs, HLS, scrobble, downloads).

---

## 10) Scale profiles (home‑lab)

Use these defaults to match your library size without adding unnecessary services.

- Small (≤ 200k tracks)
  - DB: SQLite default
  - Cache: none
  - Storage: filesystem for art/HLS
  - Container: single melodee-api image

- Medium (200k – 2M tracks)
  - DB: SQLite can work but PostgreSQL recommended for faster scans/search
  - Cache: optional Redis for hot metadata
  - Storage: filesystem for art/HLS; consider separate disk for HLS
  - Container: melodee-api + optional Postgres container

- Large (2M – 8M tracks) — your case (~4M)
  - DB: PostgreSQL 16+ (tuned); place DB on SSD/NVMe
  - Cache: Redis optional; enable if browse/search latency creeps up
  - Jobs: increase worker concurrency for scanner and transcoder; stagger scans
  - Storage: filesystem for art/HLS on fast disk; S3/MinIO optional for remote storage
  - Network: keep everything on LAN; reverse proxy only if exposing externally
  - Compose: see `docs/deploy/docker-compose.large.yml`

Notes
- Regardless of size, the Admin UI can be embedded and served statically by melodee-api to keep things simple.
- You can start Small and move to Medium/Large by switching env vars and adding containers; no migrations required when staying on SQLite → Postgres may require a one-time export/import.

---

## 10) Risks and mitigations

- Contract drift → centralized melodee-specs + contract tests in CI
- Cross-origin auth complexity → BFF pattern for admin; tokens server-side only
- HLS URL expiry → client-side auto-refresh flow with early refresh threshold
- Large libraries → FS sharding + DB indexes + background jobs with back-pressure
