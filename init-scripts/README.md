# Database Management

## Local Development Setup

For localhost development:
- **Infrastructure** (PostgreSQL, Redis, Prometheus, Grafana) runs in Podman containers via `docker-compose.dev.yml`
- **Application** (API, Worker, Frontend) runs directly on host for hot-reloading

The database schema is managed via SQL files in `init-scripts/` instead of migrations.

### Start Infrastructure

```bash
# Use the dev compose file (infrastructure only)
podman-compose -f docker-compose.dev.yml up -d

# Or create an alias in your shell:
alias dc-dev='podman-compose -f docker-compose.dev.yml'
dc-dev up -d
```

### Start Application Services

```bash
# Terminal 1: API server
./run-api.sh

# Terminal 2: Worker (if needed)
cd src/worker && GO111MODULE=on go build -o melodee-worker main.go && CONFIG_PATH=../../config.yaml ./melodee-worker

# Terminal 3: Frontend
cd src/frontend && npm run dev
```

### Reset Database

To apply schema changes or reset your local database:

```bash
# Stop and remove volumes
podman-compose -f docker-compose.dev.yml down -v

# Start fresh infrastructure with new schema
podman-compose -f docker-compose.dev.yml up -d

# Restart your API server
./run-api.sh
```

### Schema Changes

1. Edit `init-scripts/001_schema.sql`
2. Reset database: `podman-compose -f docker-compose.dev.yml down -v && podman-compose -f docker-compose.dev.yml up -d`
3. Restart your API server: `./run-api.sh`
4. That's it - no migrations to manage!

### Manual Database Access

```bash
# Connect to the running database container
podman exec -it melodee-db psql -U melodee_user -d melodee

# Or from host (port 5432 is exposed to localhost)
psql -h localhost -U melodee_user -d melodee
```

## Production Deployment

Use the main `docker-compose.yml` for production - it includes API, Worker, and Web services.

```bash
# Production deployment
docker-compose up -d
```

## Production Deployment

For production, you'll want to:
1. Use proper database backups before schema changes
2. Consider a migration system (sql-migrate, golang-migrate)
3. Never use `down -v` (it deletes all data!)

For now, local development keeps it simple with container resets.
