# Admin UI Client Spec — Next.js + shadcn/ui + Tailwind (Home‑Lab)

## 1) Architecture & stack
- Next.js (App Router), TypeScript, Tailwind, shadcn/ui, TanStack Query, Zod.
- Delivery modes:
	- Mode A (default, simplest): Static export served by melodee-api (same origin). The API manages cookie sessions and CSRF; the UI calls admin endpoints directly (no tokens in JS; HttpOnly cookie).
	- Mode B (advanced): Next.js server BFF with route handlers that call the API. Higher security isolation but adds a server component.
- Choose Mode A for home‑lab to reduce moving parts.

## 2) Auth
- Mode A: API-managed cookie session (HttpOnly; SameSite=strict). UI includes CSRF token in POST/PUT/DELETE. No tokens in JS.
- Mode B: OIDC handled in Next server (BFF); session cookies set by Next; server calls API with access token.
- PATs are available for legacy clients; not used by browser UI.

## 3) Information architecture
- Dashboard (KPIs, health, queue depth)
- Libraries (list/detail; policies; actions)
- Proposals (beets diffs; approve/reject; bulk)
- Jobs (scan/index/transcode; cancel/retry)
- Transcoding Profiles (CRUD)
- Users & Roles (RBAC management; PATs; device revoke)
- Devices (active sessions, revoke)
- Settings (OIDC, HLS, artwork)
- Audit Log (filters, export)

### Engagement visibility (read-only)
- Show engagement aggregates on entity detail pages where helpful:
	- Favorites count, average rating, likes count
- Display per-entity badges in browse/search results (favorite star, user’s own rating/reaction if authenticated via Admin session)
- No admin mutation of end-user engagement in Mode A; strictly view-only metrics for context.

## 4) Pages & flows (acceptance)
- Libraries
	- List with pagination and search
	- Create/Edit with policy_normalization and write_back_enabled
	- Actions: enqueue scan; relayout dry-run and execute
	- Acceptance: operations succeed; errors surfaced with retry; audit entries created
- Proposals
	- List with filters (confidence/state); bulk selection
	- Detail: side-by-side tag diff; approve/reject; comment optional
	- Acceptance: apply flows update DB and (if policy active) write tags; audit logged
- Jobs
	- List by kind/state; detail view with logs; cancel/retry
	- Acceptance: state transitions reflected near-real-time
- Users & Roles
	- Create user (if IdP permits provisioning) or link existing; assign roles; view devices; revoke
	- PAT management: create/revoke scoped tokens
	- Acceptance: role changes reflected in policy checks immediately
- Transcoding Profiles
	- CRUD presets (bitrates, codecs, replaygain)
- Settings & Audit
	- Global settings edit; audit search and CSV export

## 5) RBAC & guards
- Roles: admin, librarian, user.
- UI hides/disables forbidden actions; server enforces policy via scopes/roles.
- Per-library attribute rules respected in list/detail fetches.

## 6) API contracts used
- Admin: /admin/libraries, /admin/normalization/proposals, /admin/jobs, /admin/transcode-profiles, /admin/users, /admin/devices, /admin/settings, /admin/audit
- Auth: /auth/pat
- Health: /admin/health, /admin/version

## 7) Error handling & resilience
- Use error boundaries; display actionable messages; retry/backoff for transient failures.
- Optimistic UI only where safe; otherwise pessimistic with confirmation dialogs.

## 8) Testing
- Unit (Vitest/Jest), Component (RTL), E2E (Playwright), Contract tests (OpenAPI).
- Accessibility (axe); loading/error/empty states covered.

## 9) Deployment & security
- Default: build static and let melodee-api serve it from /admin (same origin). No CORS required.
- Optional: separate domain for Next server (Mode B). If used, keep tokens server-side; strict CSP with nonces.

### Static export instructions (Mode A)
- next.config.ts:
	- export default defineConfig({ output: 'export', basePath: '/admin', images: { unoptimized: true }, trailingSlash: true })
- Build: npm run build (or pnpm build) → Next.js writes static HTML/JS/CSS to ./out.
- Deliver to API container:
	- Option A (recommended simple): bind-mount the build output to /admin-static in the melodee-api container (read-only).
	- Option B: bake into the image at /admin-static in a multi-stage build.
- Runtime:
	- Set MELODEE_EMBED_ADMIN=true on the API. The server will serve /admin → /admin-static.
	- Ensure API and Admin are same-origin to use cookie session + CSRF.
