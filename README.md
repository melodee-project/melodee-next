# Melodee

[![Build](https://github.com/melodee-project/melodee-next/actions/workflows/ci.yml/badge.svg)](https://github.com/melodee-project/melodee-next/actions)
![Go version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
![Home lab friendly](https://img.shields.io/badge/homelab-ready-4caf50)

Melodee is a self-hosted music server for people with **big libraries** and **home labs**. If you’ve outgrown the usual “throw it all on a NAS and hope” setup, this is for you.

Under the hood you’ll find a Go backend, background workers, and a React/TypeScript frontend designed to handle large collections without feeling sluggish.

> **Status:** Early development / experimental. Expect rough edges and changing APIs.

---

## Features

- **Dual API architecture** – Melodee API for native clients and admin functions, plus OpenSubsonic compatibility API for third-party clients.
- **Modern Go backend** – Fiber + PostgreSQL + GORM + Redis + Asynq.
- **Built for big libraries** – directory codes, partitioning, and an optimized schema aimed at millions of tracks and beyond.
- **Background media pipeline** – scans, ingests, normalizes, and writes back metadata without blocking playback.
- **Web UI** – React + TypeScript + Vite + Tailwind admin portal.
- **Health checks & metrics** – ready for containers, homelabs, and light-touch monitoring.

If you like digging into internals, see `docs/PRD.md` and `docs/TECHNICAL_SPEC.md` for the full product and architecture spec.

---

## Repository Layout

The workspace is a multi-module Go project:

- `go.work` – Go workspaces tying the submodules together.
- `docs/` – Specifications, design docs, and fixtures.
  - `TECHNICAL_SPEC.md` – High-level architecture and service responsibilities.
  - `DATABASE_SCHEMA.md` – PostgreSQL schema and partitioning strategy.
  - `METADATA_MAPPING.md` – Tag ↔ DB field mapping.
  - `INTERNAL_API_ROUTES.md` – Internal and admin API contracts.
  - `fixtures/` – Golden request/response fixtures for internal and OpenSubsonic APIs.
- `src/` – Root Go module `melodee` and service entrypoints.
  - `main.go` – Top-level Fiber HTTP server for health / orchestration.
  - `api/` – `melodee/api` – Public/internal JSON API service.
  - `web/` – `melodee/web` – Web UI backend serving the built frontend and API proxy.
  - `worker/` – `melodee/worker` – Asynq-based background job workers.
  - `open_subsonic/` – `melodee/open_subsonic` – OpenSubsonic compatibility surface & contract tests.
  - `internal/` – `melodee/internal` – Shared domain logic, configuration, database, media pipeline, etc.
    - `config/` – Configuration loading and validation.
    - `database/` – DB connection, migrations, and partition management.
    - `directory/` – Directory codes and path templates.
    - `media/` – Media ingestion, FFmpeg integration, metadata handling.
    - `handlers/`, `services/`, `middleware/` – HTTP handlers, services, and middleware for the API and web servers.
    - `health/`, `metrics/`, `logging/`, `tracing/` – Observability.
    - `tests/`, `test/` – Internal test harnesses.

For more detail on how these pieces fit together, see `docs/DIRECTORY_ORGANIZATION_PLAN.md` and `docs/IMPLEMENTATION_GUIDE.md`.

---

## Quick Start (Development)

Melodee is designed to run primarily via **containers**. The recommended way to run it locally and in production is with **Podman** (or Docker-compatible Podman).

### Typical dev loop (TL;DR)

1. Clone the repo:

  ```bash
  git clone https://github.com/melodee-project/melodee-next.git
  cd melodee-next
  ```

2. Start the full stack in the background:

  ```bash
  podman compose -f docker-compose.yml up -d
  ```

3. Edit code under `src/` on your host.

4. When you’re ready to test your changes, rebuild and restart the services:

  ```bash
  podman compose -f docker-compose.yml build
  podman compose -f docker-compose.yml up -d
  ```

5. Hit the web UI/API on the configured localhost ports (see `docker-compose.yml`) and iterate.

### Podman / Compose (recommended)

Prerequisites:

- Podman installed on your host.
- Either the Docker-compatible shim (`podman-docker`) **or** `podman compose` / `podman-compose` available.

#### One command to start the full stack

From the repo root:

```bash
# Using Docker-compatible CLI (podman-docker) or Podman Compose
podman compose -f docker-compose.yml up -d

# Or, if you prefer the standalone podman-compose wrapper
podman-compose -f docker-compose.yml up -d
```

This will start:

- API, Web, and Worker services.
- PostgreSQL and Redis.
- Supporting infrastructure such as migrations and health checks.

All required tools (Go, Node.js, PostgreSQL, Redis, FFmpeg, etc.) are provided **inside the containers**, so you do **not** need to install them directly on your host just to run Melodee.

Once the stack is up, you can hit the web UI and API on the mapped host ports (see `docker-compose.yml` for exact values, typically something like `http://localhost:8080` / `8081`) and `/healthz` should report healthy when everything is ready.

#### Seeing code changes during local development

For the simplest, low-fuss flow:

1. Start the full stack with Podman as above.
2. Make code changes on your host in `src/`.
3. Rebuild and restart the affected service images when you want to test those changes:

  ```bash
  # From the repo root, rebuild and restart everything
  podman compose -f docker-compose.yml build
  podman compose -f docker-compose.yml up -d

  # Or, rebuild just one service (e.g., api) and restart it
  podman compose -f docker-compose.yml build api
  podman compose -f docker-compose.yml up -d api
  ```

Behind the scenes this will rebuild images with your latest source and restart the containers so they pick up the new code.

If you want a tighter dev loop (live reload without rebuilding images), you can run the Go API/web and the frontend directly on your host using `go run` / `npm run dev` while still relying on Podman for Postgres/Redis. That setup is more manual and is not the primary path described here.

### Clone and workspace setup

```bash
git clone https://github.com/melodee-project/melodee-next.git
cd melodee-next

# Ensure Go picks up all modules
cd src
go work sync || true
```

### Backend configuration

Configuration is loaded by `melodee/internal/config` (see `docs/CONFIG_ENTRY_POINT_PLAN.md`). The default loader expects environment variables and/or a config file.

Common environment variables (exact names may evolve; see `internal/config` for the source of truth):

- `MELODEE_DB_HOST`, `MELODEE_DB_PORT`, `MELODEE_DB_USER`, `MELODEE_DB_PASSWORD`, `MELODEE_DB_NAME`, `MELODEE_DB_SSLMODE`
- `MELODEE_REDIS_ADDR`, `MELODEE_REDIS_PORT`
- `MELODEE_SERVER_HOST`, `MELODEE_SERVER_PORT`
- `MELODEE_JWT_SECRET`

A typical local `.env` might look like:

```bash
MELODEE_DB_HOST=127.0.0.1
MELODEE_DB_PORT=5432
MELODEE_DB_USER=melodee
MELODEE_DB_PASSWORD=melodee
MELODEE_DB_NAME=melodee_dev
MELODEE_DB_SSLMODE=disable

MELODEE_REDIS_ADDR=127.0.0.1
MELODEE_REDIS_PORT=6379

MELODEE_SERVER_HOST=127.0.0.1
MELODEE_SERVER_PORT=8080

MELODEE_JWT_SECRET=dev-secret-change-me
```

### Database initialization & migrations

The API and Web services both run migrations on startup via `internal/database`:

1. Create the database in PostgreSQL:

   ```bash
   sudo -u postgres createuser --superuser melodee || true
   sudo -u postgres createdb melodee_dev -O melodee || true
   ```

2. Start the API once to run migrations:

   ```bash
   cd src
   go run ./api
   ```

Check `docs/DATABASE_SCHEMA.md` for schema details and partitioning strategy.

### Bootstrapping the first admin user

Melodee does not expose a public "self-registration" endpoint. All user and admin management happens through authenticated admin APIs, which means you need at least one admin account in the database before you can log in to the UI.

For homelab and fresh installs, there is a helper script in `scripts/add-admin-user.sh` that inserts an admin user directly into the Postgres database.

**Requirements**

- Postgres is reachable and already initialized with the Melodee schema (migrations run).
- The `melodee_users` table is empty or at least does not contain the username you are about to create.
- Go is installed (the script uses a tiny Go helper to generate a bcrypt password hash).

**Environment variables**

The script uses the same DB-related environment variables as the backend, with sensible defaults:

- `MELODEE_DB_HOST` (default `localhost`)
- `MELODEE_DB_PORT` (default `5432`)
- `MELODEE_DB_USER` (default `melodee_user`)
- `MELODEE_DB_NAME` (default `melodee`)
- `MELODEE_DB_PASSWORD` (**required**)

**Usage**

From the repo root:

```bash
chmod +x scripts/add-admin-user.sh   # first time only

export MELODEE_DB_PASSWORD='your-db-password'
# optionally override host/user/db/port if they differ from defaults
# export MELODEE_DB_HOST=127.0.0.1
# export MELODEE_DB_USER=melodee
# export MELODEE_DB_NAME=melodee_dev

./scripts/add-admin-user.sh <username> <email> <password>
```

Example:

```bash
./scripts/add-admin-user.sh admin admin@example.com 'YourS3cureP@ssw0rd'
```

Behind the scenes the script:

- Generates a bcrypt hash for the provided password using `scripts/cmd/bcrypt-hash/`.
- Connects to Postgres via `psql`.
- Ensures the username does not already exist.
- Inserts a row into `melodee_users` with `is_admin = TRUE`.

After running it, you can log in to the UI/API using that username/password, and the JWTs issued by the backend will carry `is_admin: true`, granting access to admin-only routes.

### Running services

In development you can run each service independently from `src/`:

```bash
cd src

# API (JSON) service
go run ./api

# Web service (serves built frontend + API routes)
go run ./web

# Worker service (background jobs)
go run ./worker

# Root service (health, config validation, basic checks)
go run .
```

The default ports/hosts are controlled by your config; expect something like:

- API: `http://127.0.0.1:8080`
- Web: `http://127.0.0.1:8081` (example; see config)
- Worker: uses Redis and DB; no HTTP port.

Health check endpoints are generally exposed at `GET /healthz` (see `internal/health`).

---

## Frontend (Web UI)

The frontend lives in `src/api/frontend/` and is a Vite + React + TypeScript project.

### Install dependencies

```bash
cd src/api/frontend
npm install
# or: pnpm install / yarn install
```

### Run the dev server

```bash
npm run dev
```

By default Vite runs on something like `http://localhost:5173` and proxies or talks to the Go API depending on the configured base URLs.

### Build for production

```bash
npm run build
```

The built assets are served by the `melodee/web` service from its `./dist` directory.

---

## Testing

### Go tests

From `src/` you can run tests across modules:

```bash
cd src
go test ./...
```

The OpenSubsonic contract tests live in `src/open_subsonic/` (see `contract_test.go`) and exercise response structure compatibility. Internal API fixtures live in `docs/fixtures/` and should be kept in sync with `docs/INTERNAL_API_ROUTES.md` and `docs/TECHNICAL_SPEC.md`.

### Frontend tests

If/when configured, run from `src/api/frontend`:

```bash
npm test
```

---

## API Usage

Melodee provides two primary APIs for different use cases:

### Melodee API (Native/Management)
- **Base path:** `/api`
- **Authentication:** JWT tokens via `/api/auth/login`
- **Usage:** Admin functions, user management, library operations, system monitoring
- **Example:** Managing users, configuring settings, monitoring jobs, library operations

```bash
# Get authentication token
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}'

# Use token for admin operations
curl -X GET http://localhost:8080/api/users \
  -H "Authorization: Bearer <jwt_token>"
```

### OpenSubsonic API (Compatibility)
- **Base path:** `/rest`
- **Authentication:** Subsonic-style credentials
- **Usage:** Third-party Subsonic/OpenSubsonic clients
- **Example:** Mobile apps, desktop players that support Subsonic

```bash
# Browse artists with Subsonic API
curl "http://localhost:8080/rest/getArtists.view?u=username&p=enc:password&v=1.16.1&c=melodee"
```

For complete API documentation, see:
- `docs/API_DEFINITIONS.md` - Overview and examples
- `docs/INTERNAL_API_ROUTES.md` - Admin/internal API routes
- `docs/SERVICE_ARCHITECTURE.md` - Service configuration and ports
- OpenAPI specs in `docs/` directory

## Developer Onboarding

This section is for new contributors (or curious users) who want to poke around the codebase.

### 1. Read the specs

Start with the design documents in `docs/`:

- `PRD.md` – What Melodee is trying to achieve.
- `TECHNICAL_SPEC.md` – Services, modules, and flows.
- `DIRECTORY_ORGANIZATION_PLAN.md` – Source tree and responsibilities.
- `MEDIA_FILE_PROCESSING.md` – The ingest → staging → production media pipeline.
- `INTERNAL_API_ROUTES.md` and `OPEN_SUBSONIC` docs – External and internal API contracts.

These give you context before you touch code.

### 2. Pick a workspace area

Common contribution areas:

- **Core API** – `src/api`, `src/internal/{handlers,services,middleware}`.
- **Media pipeline** – `src/internal/{media,directory}` and worker tasks.
- **Database & capacity** – `src/internal/database`, partitioning, `docs/CAPACITY_PROBES.md`.
- **OpenSubsonic compatibility** – `src/open_subsonic` and `docs/fixtures/opensubsonic`.
- **Frontend UI** – `src/api/frontend`.
- **Operations/observability** – `src/internal/{health,metrics,logging,tracing}`.

### 3. Running a full dev stack

For most contributors, the easiest way to run a full stack is via Podman/Compose as described in the Quick Start. If you prefer to run services directly, you can still:

1. Start Postgres, Redis, and ensure FFmpeg is installed.
2. Export or configure the env vars shown in the Quick Start.
3. Run the API, Web, and Worker in separate terminals from `src/`.

### 4. Coding style & conventions

- **Language:** Go for backend, TypeScript/React for frontend.
- **Go style:** Follow `gofmt` and idiomatic Go patterns. Prefer clear names over short ones.
- **Packages:** Respect the existing `internal` boundaries; do not introduce new cross-cutting dependencies without design discussion.
- **Errors & logging:** Use existing logging utilities in `internal/logging` where possible.
- **Config:** Thread configuration through `config.AppConfig` rather than reading env vars deep in the call stack.

### 5. Tests & fixtures

- Whenever you add or change an API route, update:
  - Corresponding handlers in `internal/handlers`.
  - Contract documentation in `docs/INTERNAL_API_ROUTES.md` or `docs/open_subsonic` notes.
  - Fixtures in `docs/fixtures/` (requests and responses).
- For media and directory code changes, keep `MEDIA_FILE_PROCESSING.md` and `DIRECTORY_ORGANIZATION_PLAN.md` aligned with reality.

Run tests before submitting changes:

```bash
cd src
go test ./...
cd api/frontend
npm test
```

### 6. Submitting changes

This repo follows a pretty standard FOSS contribution flow:

1. Fork the repository on GitHub.
2. Create a feature branch from `main`.
3. Add or update tests and documentation relevant to your change.
4. Ensure `go test ./...` (and frontend tests, if affected) pass.
5. Open a Pull Request with:
   - A clear description of the change.
   - Any migration or configuration notes.
   - Links to related docs you updated in `docs/`.

---

## Operations & Deployment (High Level)

A production deployment typically consists of:

- **API service** (`melodee/api`) – Handles JSON APIs for admin UI and possibly mobile/desktop clients.
- **Web service** (`melodee/web`) – Serves the built React UI and fronts some API endpoints.
- **Worker service** (`melodee/worker`) – Processes background tasks via Redis and Asynq.
- **PostgreSQL** – Main relational database with partitioned tables.
- **Redis** – Job queue and ephemeral application state.

Health checks and metrics endpoints (`/healthz`, `/metrics`) are designed to integrate with orchestrators like Kubernetes. See `docs/HEALTH_CHECK.md` and `docs/CAPACITY_PROBES.md` for details.

More detailed deployment notes live in `docs/IMPLEMENTATION_GUIDE.md` and `docs/TECHNICAL_SPEC.md`.

### Docker / Podman / Docker Compose

There is a `docker-compose.yml` in the repo that describes a multi-service stack (API, Web, Worker, Postgres, Redis, etc.). The name is historical – you can use it as a starting point for both local and production-like deployments.

From the repo root, a typical flow with Docker looks like:

```bash
docker compose -f docker-compose.yml up -d
```

If you prefer **Podman**, you can either use the Docker-compatible CLI shim (`podman-docker`) or run Compose via `podman compose`:

```bash
podman compose -f docker-compose.yml up -d
```

Both approaches should spin up the database, Redis, and Melodee services with sensible defaults. Check the compose file for the exact ports and environment variables that are exposed; you can override them with your own `.env` or `docker-compose.override.yml` if you want to customize for your homelab.

Once the stack is up, you can hit the web UI and API on the mapped host ports (for example `http://localhost:8080` / `8081`, depending on the compose config) and `/healthz` should report healthy when everything is ready.

---

## License

This project is licensed under the **MIT License**. See the `LICENSE` file at the repo root for the full text.

---

## Getting Help

- Review the documents in `docs/` – they are the source of truth for behavior and contracts.
- Open a GitHub issue with logs, reproduction steps, and your environment details.
- For questions about architecture or significant refactors, start a discussion attached to relevant design docs in `docs/`.
