# Melodee

This directory contains planning documents for Melodee. Key documents to start with:

## Essential Documents
- **`TECHNICAL_SPEC.md`** - Architecture and service specifications
- **`DATABASE_SCHEMA.md`** - Optimized database schema with partitioning

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