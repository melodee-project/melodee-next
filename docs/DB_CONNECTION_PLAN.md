# Database Connection and ORM Setup Plan

## Overview
This document outlines the plan for setting up PostgreSQL database connections and the GORM ORM in the Go-based Melodee system. The setup will focus on performance, scalability, and proper connection management for handling massive music libraries.

## Database Connection Configuration

### Connection Pool Settings
Based on the performance requirements for massive scale operations (10M+ songs), the following connection pool settings will be implemented:

```go
// Database connection configuration
type DatabaseConfig struct {
    Host     string `mapstructure:"host"`
    Port     int    `mapstructure:"port"`
    User     string `mapstructure:"user"`
    Password string `mapstructure:"password"`
    DBName   string `mapstructure:"dbname"`
    SSLMode  string `mapstructure:"sslmode"`
    
    // Connection Pool Settings (optimized for high concurrency)
    MaxOpenConns    int `mapstructure:"max_open_conns"`    // Maximum open connections to the database (recommended: 25-50)
    MaxIdleConns    int `mapstructure:"max_idle_conns"`    // Maximum idle connections to the database (recommended: 10-25)
    ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"` // Maximum time a connection can be reused (recommended: 30-60 minutes)
    ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"` // Maximum time a connection can be idle (recommended: 10-15 minutes)
}

// Default recommended values for large-scale operations
const (
    DefaultMaxOpenConns    = 50
    DefaultMaxIdleConns    = 25
    DefaultConnMaxLifetime = 30 * time.Minute
    DefaultConnMaxIdleTime = 15 * time.Minute
)
```

### DSN (Data Source Name) Construction
```go
// Function to build PostgreSQL DSN with proper encoding
func BuildDSN(config *DatabaseConfig) string {
    return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
        config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode)
}
```

## GORM Configuration

### Core GORM Settings
```go
// GORM configuration for performance optimization
type GormConfig struct {
    // Logging settings
    LogLevel               string        `mapstructure:"log_level"`                // silent, error, warn, info
    IgnoreNotFoundError    bool          `mapstructure:"ignore_not_found_error"`   // Return ErrRecordNotFound when record not found
    PrepareStmt            bool          `mapstructure:"prepare_stmt"`            // Execute the query with prepared statement
    DisableNestedTransaction bool       `mapstructure:"disable_nested_transaction"` // Disable nested transaction
    
    // Performance settings
    SkipDefaultTransaction bool          `mapstructure:"skip_default_transaction"`  // Create with skip default transaction
    DryRun                 bool          `mapstructure:"dry_run"`                  // Generate SQL without executing
}
```

### GORM Optimizations for Scale
```go
// Performance-oriented GORM configuration
var GORMConfig = &gorm.Config{
    Logger: logger.Default.LogMode(logger.Silent), // Performance: Set to Silent in production
    SkipDefaultTransaction: true, // Performance: Skip default transactions for single operations
    PrepareStmt: true, // Performance: Use prepared statements for frequently executed queries
    QueryFields: true, // Performance: Select by primary keys only when possible
    
    // Naming strategy for consistent database schema mapping
    NamingStrategy: schema.NamingStrategy{
        TablePrefix:   "melodee_", // Table prefix for organization
        SingularTable: false,      // Use plural table names
    },
    
    // Optimized for PostgreSQL partitioning support
    DisableForeignKeyConstraintWhenMigrating: true, // Allow migration of partitioned tables
}
```

## Database Initialization Process

### Database Connection Factory
```go
// Database connection factory with proper error handling and health checks
type DatabaseManager struct {
    config   *DatabaseConfig
    gormDB   *gorm.DB
    sqlDB    *sql.DB
    logger   *zerolog.Logger
}

// Initialize database connection with health check
func NewDatabaseManager(config *DatabaseConfig, logger *zerolog.Logger) (*DatabaseManager, error) {
    dsn := BuildDSN(config)
    
    db, err := gorm.Open(postgres.Open(dsn), GORMConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }
    
    // Configure connection pool
    sqlDB, err := db.DB()
    if err != nil {
        return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
    }
    
    sqlDB.SetMaxOpenConns(config.MaxOpenConns)
    sqlDB.SetMaxIdleConns(config.MaxIdleConns)
    sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)
    sqlDB.SetConnMaxIdleTime(config.ConnMaxIdleTime)
    
    // Verify connection
    if err := sqlDB.Ping(); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }
    
    // Run basic health check
    if err := d.runHealthCheck(); err != nil {
        return nil, fmt.Errorf("database health check failed: %w", err)
    }
    
    return &DatabaseManager{
        config: config,
        gormDB: db,
        sqlDB:  sqlDB,
        logger: logger,
    }, nil
}

// Health check function to verify database connectivity
func (d *DatabaseManager) runHealthCheck() error {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    var result int
    return d.gormDB.WithContext(ctx).Raw("SELECT 1").Scan(&result).Error
}
```

## Migrations Strategy

### Migration Management
```go
// Migration management for partitioned tables and schema updates
type MigrationManager struct {
    db     *gorm.DB
    logger *zerolog.Logger
}

// Run database migrations with partition-aware approach
func (m *MigrationManager) Migrate() error {
    // Migrate static tables first
    if err := m.migrateStaticTables(); err != nil {
        return fmt.Errorf("failed to migrate static tables: %w", err)
    }
    
    // Migrate partitioned tables (special handling needed)
    if err := m.migratePartitionedTables(); err != nil {
        return fmt.Errorf("failed to migrate partitioned tables: %w", err)
    }
    
    // Create indexes and constraints
    if err := m.createIndexes(); err != nil {
        return fmt.Errorf("failed to create indexes: %w", err)
    }
    
    // Create materialized views
    if err := m.createMaterializedViews(); err != nil {
        return fmt.Errorf("failed to create materialized views: %w", err)
    }
    
    m.logger.Info().Msg("Database migrations completed successfully")
    return nil
}

// Special handling for partitioned tables
func (m *MigrationManager) migratePartitionedTables() error {
    // First, create the main partitioned table
    if err := m.db.Exec(`
        CREATE TABLE IF NOT EXISTS songs (
            id BIGSERIAL PRIMARY KEY,
            api_key UUID UNIQUE DEFAULT gen_random_uuid(),
            name VARCHAR(255) NOT NULL,
            name_normalized VARCHAR(255) NOT NULL,
            album_id BIGINT,
            artist_id BIGINT,
            duration BIGINT,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
        ) PARTITION BY RANGE (created_at);
    `).Error; err != nil {
        return fmt.Errorf("failed to create partitioned songs table: %w", err)
    }
    
    // Create initial partitions
    if err := m.createInitialPartitions(); err != nil {
        return fmt.Errorf("failed to create initial partitions: %w", err)
    }
    
    // Similar process for albums table
    if err := m.db.Exec(`
        CREATE TABLE IF NOT EXISTS albums (
            id BIGSERIAL PRIMARY KEY,
            api_key UUID UNIQUE DEFAULT gen_random_uuid(),
            name VARCHAR(255) NOT NULL,
            artist_id BIGINT,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
        ) PARTITION BY RANGE (created_at);
    `).Error; err != nil {
        return fmt.Errorf("failed to create partitioned albums table: %w", err)
    }
    
    if err := m.createAlbumPartitions(); err != nil {
        return fmt.Errorf("failed to create album partitions: %w", err)
    }
    
    return nil
}

// Create initial partitions for the current and next few months
func (m *MigrationManager) createInitialPartitions() error {
    now := time.Now()
    for i := 0; i < 6; i++ { // Create partitions for 6 months
        partitionDate := now.AddDate(0, i, 0)
        partitionName := fmt.Sprintf("songs_%d_%02d", partitionDate.Year(), partitionDate.Month())
        startDate := time.Date(partitionDate.Year(), partitionDate.Month(), 1, 0, 0, 0, 0, time.UTC)
        endDate := time.Date(partitionDate.Year(), partitionDate.Month()+1, 1, 0, 0, 0, 0, time.UTC)
        
        query := fmt.Sprintf(`
            CREATE TABLE IF NOT EXISTS %s
            PARTITION OF songs
            FOR VALUES FROM ('%s') TO ('%s');
        `, partitionName, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
        
        if err := m.db.Exec(query).Error; err != nil {
            return fmt.Errorf("failed to create partition %s: %w", partitionName, err)
        }
    }
    
    return nil
}
```

## Performance Optimizations
### Migration Ordering and Safety
- Order: (1) extensions (`uuid-ossp`, `pg_trgm`), (2) enum types, (3) static tables, (4) partitioned parents, (5) partitions, (6) per-partition indexes, (7) materialized/views.
- Tooling: run `melodee migrate --dry-run` in CI to capture SQL for review; promote only after signed-off.
- Downgrades: mark destructive migrations irreversible in production; prefer patch migrations over drops.
- Concurrency: set `lock_timeout=5s`, `statement_timeout=10m`; retry migrations up to 3 times with jitter.

### Query Optimization Strategies
```go
// Optimized query patterns for large-scale operations
type Repository struct {
    db *gorm.DB
}

// Optimized method for fetching paginated results from large tables
func (r *Repository) GetSongsPaginated(offset, limit int, conditions map[string]interface{}) ([]Song, int64, error) {
    var songs []Song
    var total int64
    
    // Build query with conditions
    query := r.db.Model(&Song{})
    
    // Apply conditions if provided
    if len(conditions) > 0 {
        query = query.Where(conditions)
    }
    
    // Count total records (this can be expensive for large tables)
    // Consider using materialized views for frequent counts
    err := query.Count(&total).Error
    if err != nil {
        return nil, 0, fmt.Errorf("failed to count records: %w", err)
    }
    
    // Apply pagination and fetch records
    err = query.Offset(offset).Limit(limit).Find(&songs).Error
    if err != nil {
        return nil, 0, fmt.Errorf("failed to fetch records: %w", err)
    }
    
    return songs, total, nil
}

// Optimized method using raw SQL for complex queries that benefit from it
func (r *Repository) GetSongsWithAlbumAndArtist(songIds []int64) ([]SongWithDetails, error) {
    var results []SongWithDetails
    
    // Using raw SQL for complex JOINs to leverage PostgreSQL optimizations
    query := `
        SELECT 
            s.id, s.name, s.duration,
            a.name as artist_name,
            al.name as album_name
        FROM songs s
        JOIN artists a ON s.artist_id = a.id
        JOIN albums al ON s.album_id = al.id
        WHERE s.id = ANY(?)
        ORDER BY s.id
    `
    
    err := r.db.Raw(query, pq.Int64Array(songIds)).Scan(&results).Error
    if err != nil {
        return nil, fmt.Errorf("failed to execute optimized query: %w", err)
    }
    
    return results, nil
}
```

### Caching Integration
```go
// Integration with Redis for caching frequently accessed data
type CachedRepository struct {
    repo       *Repository
    redis      *redis.Client
    logger     *zerolog.Logger
}

// Get song by ID with Redis caching
func (c *CachedRepository) GetSongByID(id int64) (*Song, error) {
    // Try to get from Redis cache first
    cacheKey := fmt.Sprintf("song:%d", id)
    
    cached, err := c.redis.Get(context.Background(), cacheKey).Result()
    if err == nil {
        // Cache hit
        var song Song
        if err := json.Unmarshal([]byte(cached), &song); err == nil {
            return &song, nil
        }
        // If unmarshaling fails, log and continue to DB
        c.logger.Warn().Err(err).Int64("song_id", id).Msg("Failed to unmarshal cached song")
    }
    
    // Cache miss - fetch from database
    song, err := c.repo.GetSongByID(id)
    if err != nil {
        return nil, err
    }
    
    // Cache the result for 1 hour
    data, _ := json.Marshal(song)
    c.redis.SetEX(context.Background(), cacheKey, data, time.Hour)
    
    return song, nil
}
```

## Monitoring and Health Checks

### Database Health Monitoring
```go
// Health check implementation for monitoring
type DatabaseHealthChecker struct {
    db     *gorm.DB
    logger *zerolog.Logger
}

// Perform comprehensive database health check
func (d *DatabaseHealthChecker) CheckHealth() HealthStatus {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    // Check basic connectivity
    if err := d.db.WithContext(ctx).Exec("SELECT 1").Error; err != nil {
        return HealthStatus{
            Status:  "unhealthy",
            Message: fmt.Sprintf("Basic query failed: %v", err),
        }
    }
    
    // Check connection pool metrics
    sqlDB, err := d.db.DB()
    if err != nil {
        return HealthStatus{
            Status:  "degraded",
            Message: fmt.Sprintf("Could not get connection metrics: %v", err),
        }
    }
    
    stats := sqlDB.Stats()
    if stats.OpenConnections > stats.MaxOpenConnections*90/100 {
        d.logger.Warn().
            Int("open_connections", stats.OpenConnections).
            Int("max_connections", stats.MaxOpenConnections).
            Msg("High database connection usage detected")
    }
    
    // More comprehensive checks can be added as needed
    
    return HealthStatus{
        Status: "healthy",
        Message: fmt.Sprintf("Connected with %d open connections", stats.OpenConnections),
    }
}
```

## Error Handling and Recovery

### Robust Error Handling
```go
// Database error wrapper with context
type DBError struct {
    Op          string // operation that failed
    Table       string // table involved
    QueryParams map[string]interface{} // query parameters
    Err         error  // underlying error
}

func (e *DBError) Error() string {
    return fmt.Sprintf("database operation %q on table %q failed: %v", e.Op, e.Table, e.Err)
}

func (e *DBError) Unwrap() error {
    return e.Err
}

// Function wrapper with error handling
func (r *Repository) QueryWithRecovery(ctx context.Context, operation string, table string, queryFn func() error) error {
    err := queryFn()
    if err != nil {
        dbErr := &DBError{
            Op:          operation,
            Table:       table,
            QueryParams: extractParams(ctx), // helper to extract query params
            Err:         err,
        }
        return dbErr
    }
    return nil
}
```

This database connection and ORM setup plan ensures optimal performance for the Go-based Melodee system, with specific attention to the needs of handling massive music libraries while maintaining scalability and reliability.
