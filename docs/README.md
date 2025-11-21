# Melodee

This directory contains planning documents for Melodee. Key documents to start with:

## Essential Documents
- **`TECHNICAL_SPEC.md`** - Architecture and service specifications
- **`DATABASE_SCHEMA.md`** - Optimized database schema with partitioning
- **`METADATA_MAPPING.md`** - Tag ↔ DB mapping and conflict rules

## Technical Stack
- Backend: Go + Fiber
- Database: PostgreSQL + GORM
- Queue: Redis + Asynq
- Frontend: React + TypeScript

## Key Features
- OpenSubsonic API compatibility
- Directory codes for 300k+ artist performance
- Horizontal partitioning for massive scale
- Media processing pipeline (inbound → staging → production)

## Test & Fixture Expectations
- Contract fixtures for OpenSubsonic/internal APIs belong in `docs/fixtures/`; include request, response, and notes on auth context.
- Golden responses should mirror the conventions in `TECHNICAL_SPEC.md` (pagination, errors, date formats).
- See `docs/fixtures/README.md` for structure and naming.
