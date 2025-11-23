# Melodee System Documentation

**Audience:** Contributors, engineers, operators

**Purpose:** Entry point and index for all Melodee docs.

**Source of truth for:** High-level orientation, where to find detailed specs.

---

## 1. Orientation

Melodee is a high-performance music streaming server that implements the OpenSubsonic API specification. This folder contains product, architecture, implementation, and operations documentation.

If you're new, start with:
- `PRD.md`  Product requirements and core capabilities
- `TECHNICAL_SPEC.md`  System behavior, APIs, data model, and services
- `TECHNICAL_STACK.md`  Technology choices and rationale

For implementation sequencing, see `IMPLEMENTATION_GUIDE.md`.

---

## 2. Document Index

### 2.1 Product & UX
- `PRD.md`  Product requirements and success criteria
- `MISSING_FEATURES.md`  Known gaps vs target
- `uat_summary.md`  UAT outcomes and notes
- `clients/ADMIN-UI-SPEC.md`  Admin web UI specification
- `clients/ANDROID-SPEC.md`  Android/mobile client notes
- `clients/DESKTOP-SPEC.md`  Desktop client notes

### 2.2 Architecture & Behavior
- `TECHNICAL_SPEC.md`  Canonical technical spec (architecture, APIs, jobs, normalization)
- `TECHNICAL_STACK.md`  Canonical stack & tooling choices
- `DATABASE_SCHEMA.md`  Canonical DB schema and partitioning playbook
- `DIRECTORY_ORGANIZATION_PLAN.md`  Directory codes, path templates, and FS integration
- `MEDIA_FILE_PROCESSING.md`  Media pipeline, FFmpeg profiles, idempotency rules
- `METADATA_MAPPING.md`  Metadata rules and mappings
- `GO_MODULE_PLAN.md`  Go module layout
- `DB_CONNECTION_PLAN.md`  DB connection strategy and pooling
- `SERVICE_ARCHITECTURE.md` ‒ Service binaries and runtime configuration
- `CONFIG_ENTRY_POINT_PLAN.md`  Config entrypoint and env strategy

### 2.3 APIs & Contracts
- `API_DEFINITIONS.md`  High-level API groupings
- `INTERNAL_API_ROUTES.md`  Canonical internal REST endpoints and contracts
- `melodee-v1.0.0-openapi.yaml`  Melodee OpenAPI definition
- `opensubsonic-v1.16.1-openapi.yaml`  Upstream OpenSubsonic spec
- `subsonic-v1.16.1-openapi.yaml`  Upstream Subsonic spec
- `TESTING_CONTRACTS.md`  Contract testing strategy and fixtures mapping
- `fixtures/`  Request/response fixtures (internal + OpenSubsonic)

### 2.4 Operations & SRE
- `HEALTH_CHECK.md`  `/healthz` contract and thresholds
- `CAPACITY_PROBES.md`  Capacity probe design and thresholds
- `BACKUP_RECOVERY_PROCEDURES.md`  Backup and restore playbooks
- `runbooks.md`  Operational runbooks
- `secrets_management.md`  Secrets and key management

### 2.5 Testing & Quality
- `IMPLEMENTATION_GUIDE.md`  Phase plan and coding agent guidance
- `TESTING_CONTRACTS.md`  (also listed above) contract tests
- `API_IMPLEMENTATION_PHASES.md`  API implementation milestones

---

## 3. Installation & Operations

For installation, configuration, deployment, monitoring, and troubleshooting details, use:
- Top-level `README.md` at repo root  primary install/deploy guide
- `config/config.template.yaml`  sample config
- `scripts/` and `init-scripts/`  helper scripts for setup and ops

This `docs` folder focuses on specifications, contracts, and operational playbooks rather than step-by-step install instructions.

---

## 4. Single Sources of Truth

When updating behavior, prefer these canonical documents:
- Product behavior: `PRD.md`
- Architecture & contracts: `TECHNICAL_SPEC.md`
- Database design: `DATABASE_SCHEMA.md`
- Directory/path behavior: `DIRECTORY_ORGANIZATION_PLAN.md`
- Media pipeline: `MEDIA_FILE_PROCESSING.md`
- Internal REST routes: `INTERNAL_API_ROUTES.md`
- OpenSubsonic behavior: `TECHNICAL_SPEC.md` + fixtures under `fixtures/opensubsonic/`

Avoid re-stating these details in new docs; link to them instead.

---

## 5. Conventions for New Docs

New documents in this folder should:
- Start with **Audience**, **Purpose**, and **Source of truth for**.
- Link to existing canonical docs instead of duplicating content.
- Be added to the index above so `docs/README.md` remains the up-to-date map.
