# Shared API Contracts (OpenAPI) — Plan

Source of truth for API contracts will live in a dedicated repo (recommended: melodee-specs). While that repo is set up, this document outlines structure and codegen.

## Structure (in melodee-specs)
- openapi/
  - melodee.yaml (root; $ref to sub-files)
  - paths/
  - components/
  - security.yaml
- jsonschema/
  - proposal-diff.schema.json
- codegen/
  - openapitools.json (configs per language)

## Code generation
- Go (server): models + server stubs → melodee-sdk-go
- TypeScript (client): fetch/axios client → melodee-sdk-ts
- Dart (client): dio/chopper client → melodee-sdk-dart

## Versioning
- SemVer; breaking changes bump MAJOR; deprecations with Sunset headers; API uses /v1 path.

## Security
- oauth2 Authorization Code + PKCE; scopes for admin, playback, and user engagement (read/write);
  PATs as apiKey (header) for legacy clients and first-party /me/* endpoints.

## Contract testing
- API and clients validate against melodee.yaml in CI using Dredd/Schemathesis or similar.

## Endpoints summary (incremental)
- User engagement (first-party + OIDC/PAT):
  - GET/PUT /me/favorites
  - GET/PUT /me/ratings (0 clears)
  - GET/PUT /me/reactions (like|dislike|none)
- OpenSubsonic interop for engagement:
  - star/unstar ↔ favorites
  - setRating ↔ ratings

## Browse sorting conventions
- Add a `sort` query parameter for entity lists (artists, albums, tracks). Supported values include:
  - `likes_score`: order by Wilson score (likes vs dislikes), then likes_count, then id
  - `rating`: order by rating_avg, then rating_count, then id
- Example: `GET /browse/artists?page=1&pageSize=50&sort=likes_score`
- Caching guidance: cache first pages per entity type and `sort` for 60–300 seconds.
