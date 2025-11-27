package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"melodee/internal/config"
)

// DatabaseManager manages database connections
type DatabaseManager struct {
	config *config.DatabaseConfig
	gormDB *gorm.DB
	sqlDB  *sql.DB
	logger *zerolog.Logger
}

// BuildDSN creates a PostgreSQL DSN from configuration
func BuildDSN(config *config.DatabaseConfig) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode)
}

// GORMConfig represents GORM configuration for performance optimization
var GORMConfig = &gorm.Config{
	Logger: logger.Default.LogMode(logger.Silent), // Performance: Set to Silent in production
	SkipDefaultTransaction: true, // Performance: Skip default transactions for single operations
	PrepareStmt: true, // Performance: Use prepared statements for frequently executed queries
	QueryFields: true, // Performance: Select by primary keys only when possible

	// Naming strategy for consistent database schema mapping
	NamingStrategy: schema.NamingStrategy{
		TablePrefix:   "", // No table prefix - using clean table names
		SingularTable: false,      // Use plural table names
	},

	// Optimized for PostgreSQL partitioning support
	DisableForeignKeyConstraintWhenMigrating: true, // Allow migration of partitioned tables
}

// NewDatabaseManager creates a new database manager
func NewDatabaseManager(config *config.DatabaseConfig, logger *zerolog.Logger) (*DatabaseManager, error) {
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

	// Configure connection pool

	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	// Verify connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Run basic health check
	if err := runHealthCheck(db); err != nil {
		return nil, fmt.Errorf("database health check failed: %w", err)
	}

	return &DatabaseManager{
		config: config,
		gormDB: db,
		sqlDB:  sqlDB,
		logger: logger,
	}, nil
}

// runHealthCheck performs a basic query to verify database connectivity
func runHealthCheck(db *gorm.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var result int
	return db.WithContext(ctx).Raw("SELECT 1").Scan(&result).Error
}

// GetGormDB returns the GORM database instance
func (d *DatabaseManager) GetGormDB() *gorm.DB {
	return d.gormDB
}

// GetSQLDB returns the underlying SQL database instance
func (d *DatabaseManager) GetSQLDB() *sql.DB {
	return d.sqlDB
}

// Close closes the database connection
func (d *DatabaseManager) Close() error {
	return d.sqlDB.Close()
}

// NewDatabaseManagerFromExisting creates a DatabaseManager from existing GORM and SQL instances
func NewDatabaseManagerFromExisting(gormDB *gorm.DB, sqlDB *sql.DB) *DatabaseManager {
	return &DatabaseManager{
		gormDB: gormDB,
		sqlDB:  sqlDB,
	}
}