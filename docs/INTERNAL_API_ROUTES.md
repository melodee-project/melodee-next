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
- `GET /api/libraries` -> list library states
- `GET /api/libraries/:id` -> get specific library state
- `GET /api/libraries/stats` -> aggregate stats
- `POST /api/libraries/scan` -> enqueue scan job (fixtures)
- `POST /api/libraries/process` -> move inbound->staging
- `POST /api/libraries/move-ok` -> promote staging OK albums
- `GET /api/libraries/quarantine` -> list quarantine items
- `POST /api/libraries/quarantine/:id/resolve` -> resolve quarantine item
- `POST /api/libraries/quarantine/:id/requeue` -> requeue quarantine item

## Images (avatars/cover art)
- `GET /api/images/:id` -> binary with ETag/Last-Modified
- `POST /api/images/avatar` -> multipart `file`; max 2MB JPEG/PNG; returns `{id, etag}`
- Errors: 415 invalid MIME, 413 too large (fixtures pending success/invalid MIME)

## Shares (admin)
- `GET /api/shares` -> list shares with pagination
- `POST /api/shares` -> `{name, track_ids, expires_at, max_streaming_minutes, allow_download}`
- `PUT /api/shares/:id` -> update existing share
- `DELETE /api/shares/:id` -> delete share

## Settings (admin)
- `GET /api/settings` -> list all settings
- `PUT /api/settings/:key` -> update single key (fixtures)

## Jobs/Admin (admin)
- `GET /api/admin/jobs/dlq` -> list DLQ items `{data:[{id,queue,type,reason,payload}]}` (fixture needed)
- `POST /api/admin/jobs/requeue` -> requeue specified jobs
- `POST /api/admin/jobs/purge` -> purge specified jobs

## Capacity monitoring
- `GET /api/admin/capacity` -> capacity status for all libraries
- `GET /api/admin/capacity/:id` -> capacity status for specific library
- `POST /api/admin/capacity/probe-now` -> trigger immediate capacity probe

## Search
- `GET /api/search` -> `{data:[entities], pagination}`; supports `type=artist|album|song`, `q`, `offset`, `limit` (see pagination fixture)
