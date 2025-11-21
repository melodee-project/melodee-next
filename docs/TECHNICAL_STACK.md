# Melodee - Technical Stack Recommendations

## Executive Summary

This document outlines the recommended technology stack for the Melodee music management and streaming system, with a primary focus on high-performance file serving for music streaming while maintaining security and best practices.

## Core Stack: Go-Focused Architecture

### Backend: Go + Fiber
- **Primary Framework**: [Fiber](https://github.com/gofiber/fiber) - Express.js inspired web framework for Go
- **Rationale**: Superior performance for file serving and concurrent streaming connections
- **Key Benefits**:
  - 3-10x faster than Node.js for serving static files
  - Excellent concurrent streaming performance with Goroutines
  - Minimal memory overhead
  - Superb range request handling for streaming
  - Built-in compression and security middleware

### Database: PostgreSQL (Enhanced for Performance)
- **ORM/Query Builder**: [GORM](https://gorm.io/)
- **Rationale**: Enhanced schema for massive scale operations (see DATABASE_SCHEMA.md) with performance-first design
- **Features**:
  - JSON/JSONB support for flexible metadata storage
  - Full-text search for music library search functionality
  - Horizontal partitioning support with GORM
  - Connection pooling with pgx driver
  - Materialized views for aggregate statistics

### File Processing and Streaming
- **Audio Processing**: Native Go with external FFmpeg binary calls
- **Image Processing**: [bimg](https://github.com/h2non/bimg) for efficient image manipulation
- **Streaming Optimization**: Direct file serving through Go's optimized HTTP server
- **Transcoding**: FFmpeg integration for real-time format conversion
- **Directory Organization**: Service for generating artist directory codes and managing configurable directory layouts
- **Metadata Reading**: Native Go libraries for reading existing metadata from audio files
- **Metadata Writing**: Update existing metadata in audio files when user makes edits

## Frontend Stack

### Core Framework
- **Frontend Framework**: React with TypeScript and Vite
- **Component Library**: Radix UI for unstyled, accessible components
- **Styling**: Tailwind CSS for utility-first approach
- **State Management**: Zustand for simplicity and performance

## Security Stack

### Authentication & Authorization
- **JWT Implementation**: [go-jwt](https://github.com/golang-jwt/jwt) for secure tokens
- **Password Hashing**: bcrypt with proper salt management
- **Rate Limiting**: Built-in Fiber middleware for API throttling
- **Session Management**: Secure HTTP-only cookies

### Security Headers & Practices
- **Middleware**: Fiber security middleware for protection against common attacks
- **Input Validation**: [validator](https://github.com/go-playground/validator) for Go
- **SQL Injection Prevention**: Prepared statements with GORM
- **XSS Protection**: Automatic HTML escaping in templates

## Caching & Performance

### Primary Caching
- **In-Memory Caching**: Go's sync.Map for frequently accessed metadata
- **Redis Integration**: [go-redis](https://github.com/go-redis/redis) for distributed caching
- **CDN Strategy**: Static asset optimization with ETag support

### Performance Optimizations
- **HTTP/2 Support**: Built into Go's HTTP server
- **Compression**: Built-in gzip and brotli compression
- **Connection Pooling**: Database connection pooling with pgx
- **Range Request Support**: Native Go HTTP range request handling for streaming

## Job Scheduling & Background Processing

### Job Queue System
- **Primary Queue**: [Asynq](https://github.com/hibiken/asynq) - Redis-based job queue
- **Cron Scheduling**: Built-in time package with Redis for distributed scheduling
- **Worker Management**: Goroutine-based workers for parallel processing

## Testing Strategy

### Backend Testing
- **Unit Tests**: Go's built-in `testing` package
- **API Testing**: [testify](https://github.com/stretchr/testify) for assertions and mocking
- **Database Testing**: In-memory PostgreSQL or test containers
- **Integration Tests**: Test entire API flows and database operations

### Frontend Testing
- **Unit Tests**: Vitest for fast unit testing
- **Component Tests**: React Testing Library with Jest
- **E2E Tests**: Playwright for end-to-end testing
- **Visual Regression**: Storybook with Chromatic

## Infrastructure & Deployment

### Containerization
- **Primary Runtime**: Docker + Docker Compose
- **Build Tools**: Multi-stage builds for optimized container size
- **Orchestration**: Kubernetes-ready but deployable with Docker Compose

### Monitoring & Observability
- **Logging**: [zerolog](https://github.com/rs/zerolog) for structured logging
- **Metrics**: Prometheus client for Go
- **Tracing**: OpenTelemetry integration
- **Health Checks**: Built-in health check endpoints

## API Design

### OpenSubsonic API Implementation
- **Framework Benefits**: Fiber's performance advantages for API endpoints
- **Request Handling**: Native Go HTTP performance for range requests
- **Response Formatting**: Fast JSON marshaling with proper caching
- **Authentication**: Built-in middleware for API security

### Internal API Design
- **RESTful Design**: Following REST conventions with Go Fiber
- **Documentation**: Swagger/OpenAPI with [swaggo](https://github.com/swaggo/swag)
- **Validation**: Request/response validation with structured types
- **Rate Limiting**: Per-user and global rate limiting

## Performance-Specific Considerations

### File Serving Optimizations
- **Zero-Copy Serving**: Go's optimized file serving with io.Copy
- **Memory Mapping**: Efficient large file handling with mmap
- **Range Requests**: Native support for HTTP range requests (crucial for streaming)
- **Buffer Management**: Optimized buffer sizes for large file serving

### Concurrency Handling
- **Goroutines**: Efficient concurrent connection handling
- **Channel Communication**: Safe inter-worker communication
- **Connection Pooling**: Optimized database and Redis connections
- **Memory Management**: Efficient garbage collection with Go 1.19+

## Security Best Practices

### Data Protection
- **Encryption at Rest**: PostgreSQL encryption options
- **Encryption in Transit**: HTTPS/TLS for all communications
- **Input Sanitization**: Comprehensive input validation and sanitization
- **Error Handling**: Secure error logging without exposing internals

### API Security
- **Authentication**: JWT with refresh token rotation
- **Authorization**: Role-based access control with permissions
- **Rate Limiting**: Per-user and global rate limiting
- **CORS**: Proper CORS configuration for web interface

## Development Tooling

### Backend Development
- **IDE Support**: VS Code with Go extension
- **Code Generation**: Go generate for boilerplate reduction
- **Formatting**: gofmt with pre-commit hooks
- **Dependency Management**: Go modules

### Frontend Development
- **Build Tool**: Vite for fast development
- **Type Safety**: TypeScript with strict mode
- **Linting**: ESLint with TypeScript rules
- **Formatting**: Prettier for consistent code style

## Migration Considerations

### Data Migration
- **Schema Evolution**: GORM migrations with PostgreSQL
- **Data Transfer**: Efficient batch processing for large datasets
- **Validation**: Comprehensive data validation during migration
- **Rollback Strategy**: Safe rollback procedures

### API Compatibility
- **OpenSubsonic Compliance**: 100% API compatibility testing
- **Client Testing**: Integration testing with popular Subsonic clients
- **Performance Baseline**: Performance benchmarks vs original system
- **Gradual Migration**: Support for both systems during transition