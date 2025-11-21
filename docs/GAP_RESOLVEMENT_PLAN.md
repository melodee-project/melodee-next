# Gap Resolvement Plan

Checkboxes track whether each gap is resolved (`[x]`) or still needs work (`[ ]`).

- [x] Avatar/Cover Upload Contracts  
  - Define content-type requirements, max size, response schema (success/error), storage location, dedup/ETag policy. Add success/invalid-MIME fixtures.

- [x] OpenSubsonic Coverage Completeness  
  - Add avatar upload (if supported), playlist delete success body, missing/unauthorized error fixtures, and parameter defaults/order rules for search/search2/search3 documented in spec.

- [x] Auth Reset/Lockout Contracts  
  - Provide endpoint contracts and fixtures for `POST /api/auth/request-reset` and `POST /api/auth/reset` including payloads and error cases.

- [x] Internal API Schemas  
  - Publish authoritative schema/OpenAPI or param/type table for all internal endpoints to avoid inference from prose/fixtures.

- [x] Job Admin Endpoints  
  - Document routes, auth requirements, and responses for DLQ list/inspect and job detail endpoints. Add fixtures for listing DLQ items.

- [x] Monitoring/Health Examples  
  - Add health check JSON fixture (path, status codes/headers) and specify units/labels for metrics including storage capacity checks.

- [x] Library Capacity Probes  
  - Provide cross-platform probe guidance and error-handling expectations when probes fail; include sample configs.

- [x] Environment Config Matrix  
  - Create a matrix of required env vars per environment (dev/stage/prod) and secret management guidance (env vs file vs vault).

- [x] Testing/Contract Enforcement  
  - Document which fixtures map to automated contract tests and outline the test harness config to exercise them in CI.
