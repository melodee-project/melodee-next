# Go Module Structure and Dependencies Plan

## Overview
This document outlines the planned Go module structure and dependency management for the Melodee music management and streaming system. The structure is designed to support the microservices architecture while maintaining code organization and dependency management best practices.

## Module Structure

### Root Module: `melodee`
The root module serves as the main project container and coordinates between all services.

### Submodules
1. **`melodee/api`** - OpenSubsonic API service
2. **`melodee/web`** - Internal REST API for web interface  
3. **`melodee/worker`** - Background job processing service
4. **`melodee/internal`** - Shared internal packages
   - `melodee/internal/models` - Database models
   - `melodee/internal/database` - Database connection and migration utilities
   - `melodee/internal/services` - Business logic services
   - `melodee/internal/utils` - Utility functions
   - `melodee/internal/config` - Configuration management

## Dependencies Plan

### Core Dependencies

#### API Server (Fiber Framework)
```go
// For api service
github.com/gofiber/fiber/v2 v2.52.0
github.com/gofiber/contrib/fiberzap v0.0.0
github.com/gofiber/contrib/swagger v0.0.0
```

#### Database & ORM
```go
// For database access
gorm.io/gorm v1.25.6
gorm.io/driver/postgres v1.5.8
github.com/jackc/pgx/v5 v5.5.3
github.com/lib/pq v1.10.9
```

#### Configuration Management
```go
// For configuration handling
github.com/spf13/viper v1.18.0
github.com/spf13/cobra v1.8.1
```

#### Job Queue & Background Processing
```go
// For job processing
github.com/hibiken/asynq v0.25.0
github.com/go-redis/redis/v9 v9.0.5
```

#### Audio Processing & File Handling
```go
// For media processing
github.com/go-audio/audio v1.0.0
github.com/go-audio/wav v1.0.0
github.com/h2non/bimg v1.1.1  // For image processing
```

#### Security & Authentication
```go
// For authentication
github.com/golang-jwt/jwt/v5 v5.0.0
golang.org/x/crypto v0.22.0  // For bcrypt
github.com/gofiber/contrib/fiberjwks v0.0.0
```

#### Validation & Utilities
```go
// For validation and utilities
github.com/go-playground/validator/v10 v10.15.5
github.com/google/uuid v1.5.0
github.com/rs/zerolog v1.30.0
github.com/stretchr/testify v1.9.0  // For testing
```

#### Testing Dependencies
```go
// For testing
github.com/DATA-DOG/go-sqlmock v1.6.0
github.com/stretchr/testify v1.9.0
gotest.tools/v3 v3.5.0
```

## Module Organization Strategy

### 1. Service Modules
Each service (api, web, worker) will have its own go.mod file but will import shared internal packages for:

- Database models
- Configuration
- Common utilities
- Business logic services

### 2. Internal Module Structure
The `internal` module will be organized as follows:

```
internal/
├── models/          # Database model definitions
├── database/        # Database initialization and migrations
├── services/        # Business logic services
├── utils/           # Utility functions and helpers
├── config/          # Configuration loading and validation
├── handlers/        # API request handlers
└── middleware/      # HTTP middleware
```

### 3. Dependency Management Strategy

#### Shared Dependencies
Common dependencies will be defined in the internal module, with the service modules importing the internal packages.

#### Version Management
- Use semantic versioning for all dependencies
- Regular updates to security patches
- Lock files to ensure reproducible builds
- Periodic dependency audits

#### Go Version
- Target Go 1.22+ for optimal performance and features
- Use go work for local development across multiple modules

## Go Workspace Configuration

For development, a go.work file will coordinate all modules:

```go
go 1.22

use (
    ./api
    ./web
    ./worker
    ./internal
)
```

## Build and Deployment Considerations

### Multi-stage Docker Builds
Each service will have its own Dockerfile optimized for size and security:

1. Build stage: Compile Go binaries
2. Runtime stage: Copy binary to minimal alpine image

### Cross-compilation Support
- Plan for ARM64 support (for Raspberry Pi, Apple Silicon)
- AMD64 for standard deployments

## Security Considerations

### Dependency Security
- Regular scanning for vulnerabilities
- Use of trusted sources only
- Minimal dependency approach where possible
- Dependency pinning for production builds

### Module Security
- Internal packages will be kept private using Go's internal visibility rules
- No external access to internal business logic
- Proper module boundaries between services

## Performance Considerations

### Optimized Dependencies
- Fiber framework for high-performance HTTP handling
- Optimized database drivers for PostgreSQL
- Efficient image processing libraries
- Memory-efficient audio processing

### Resource Management
- Proper connection pooling for database and Redis
- Memory management for large file operations
- Efficient goroutine usage for concurrent operations

This dependency and module structure plan ensures a maintainable, scalable, and performant Go-based music management system while following Go best practices and security guidelines.