# Home-lab deployment (single container)

This example uses a single melodee-api container (API + workers). Admin UI can be served as a static export from the same container when `MELODEE_EMBED_ADMIN=true`.

## Quick start
1. Copy `docker-compose.example.yml` to your host and adjust ports and paths.
2. Create a bind mount if you prefer: replace the named volume with a host path.
3. Start the service and open http://localhost:8080.

For large libraries (~2M–8M tracks), use `docker-compose.large.yml` which includes PostgreSQL and Redis and sets sensible Postgres flags.

## Configuration
- MELODEE_DB: `sqlite:///data/melodee.db` (default) or `postgres://...` if you run Postgres.
- MELODEE_STORAGE_DIR: base directory for library scan destination paths (not required if you point to existing music folders).
- MELODEE_HLS_DIR and MELODEE_ART_DIR: local directories for derived assets.
- MELODEE_AUTH_MODE: `local` by default; set to `oidc` to enable external IdP.
 - MELODEE_CACHE_REDIS: `redis://redis:6379` to enable caching/token store (optional).

## Optional extras
- Reverse proxy (Caddy/NGINX) with TLS if you expose it outside your LAN.
- Prometheus scrape if you enable metrics.

## Backup
Back up data regularly; test restores occasionally.

### SQLite (default)
- DB path: `/data/melodee.db` inside the container.
- Quick online backup to a file within the volume:
	- Example (host shell):
		- Create backup directory on the data volume (once):
			- docker exec melodee mkdir -p /data/backup
		- Create a timestamped backup using sqlite3 online backup:
			- docker exec melodee sh -c "sqlite3 /data/melodee.db \".backup '/data/backup/melodee-$(date +%F).db'\""
- Off-host copy: copy `/data/melodee.db` and `/data/backup/` via `docker cp` or by backing up the mounted host path.
- Restore: stop the container, replace `/data/melodee.db` with a known-good copy, then start.

### PostgreSQL (large profile)
- Service name in compose: `melodee-postgres` (container) with DB `melodee`, user `melodee`.
- Dump (host shell):
	- docker exec -t melodee-postgres pg_dump -U melodee melodee | gzip > pg-backup-$(date +%F).sql.gz
- Restore (host shell):
	- gunzip -c pg-backup-YYYY-MM-DD.sql.gz | docker exec -i melodee-postgres psql -U melodee -d melodee

## Serving Admin UI (static, Mode A)
- Build the Admin UI with Next.js static export (see Admin UI spec); output directory is `out/`.
- Mount into the API container at `/admin-static` (read-only), e.g. add to compose:
	- volumes:
		- ./admin-out:/admin-static:ro
- Set `MELODEE_EMBED_ADMIN=true`; the API will serve the UI at `/admin` (same origin).
