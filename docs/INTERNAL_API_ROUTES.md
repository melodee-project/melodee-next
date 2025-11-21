# Internal API Routes (Contracts)

Authoritative list of internal REST endpoints, params, and roles. JSON by default unless noted.

## Auth
- `POST /api/auth/login` -> `{access_token, refresh_token, expires_in, user}`
- `POST /api/auth/refresh` -> same shape; requires refresh token (header/body)
- `POST /api/auth/request-reset` -> body `{email}`; 202 on success (no enumeration of user)
- `POST /api/auth/reset` -> body `{token, password}`; returns `{status:"ok"}`; errors: `invalid_token`, `password_policy`

## Users (admin)
- `GET /api/users` -> `{data, pagination}`
- `POST /api/users` -> create (see fixtures)
- `PUT /api/users/:id` -> update (see fixtures)
- `DELETE /api/users/:id` -> `{status:"deleted"}`

## Playlists
- `GET /api/playlists` -> `{data, pagination}`
- `POST /api/playlists` -> create (fixtures)
- `GET /api/playlists/:id` -> playlist detail with song ids
- `PUT /api/playlists/:id` -> update (fixtures)
- `DELETE /api/playlists/:id` -> `{status:"deleted"}`

## Libraries
- `POST /api/libraries/scan` -> enqueue scan job (fixtures)
- `POST /api/libraries/process` -> move inbound->staging
- `POST /api/libraries/move-ok` -> promote staging OK albums
- `GET /api/libraries/stats` -> aggregate stats

## Images (avatars/cover art)
- `GET /api/images/:id` -> binary with ETag/Last-Modified
- `POST /api/images/avatar` -> multipart `file`; max 2MB JPEG/PNG; returns `{id, etag}`
- Errors: 415 invalid MIME, 413 too large (fixtures pending success/invalid MIME)

## Shares (admin)
- `GET /api/shares`
- `POST /api/shares` -> `{name, ids, expires_at, max_streaming_minutes, allow_download}`
- `DELETE /api/shares/:id`

## Settings (admin)
- `GET /api/settings`
- `PUT /api/settings` -> update single key (fixtures)

## Jobs/Admin (admin)
- `GET /api/admin/jobs/dlq` -> list DLQ items `{data:[{id,queue,type,reason,payload}]}` (fixture needed)
- `POST /api/admin/jobs/requeue` -> see fixtures
- `POST /api/admin/jobs/purge` -> see fixtures
- `GET /api/admin/jobs/:id` -> job detail/status

## Search
- `GET /api/search` -> `{data:[entities], pagination}`; supports `type=artist|album|song`, `q`, `offset`, `limit` (see pagination fixture)
