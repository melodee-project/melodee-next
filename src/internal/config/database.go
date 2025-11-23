package config

import (
	"time"

	"github.com/spf13/viper"
)

// Default recommended values for large-scale operations
const (
	DefaultMaxOpenConns    = 50
	DefaultMaxIdleConns    = 25
	DefaultConnMaxLifetime = 30 * time.Minute
	DefaultConnMaxIdleTime = 15 * time.Minute
)

// LoadDatabaseConfig loads database configuration from viper
func LoadDatabaseConfig() (*DatabaseConfig, error) {
	var config DatabaseConfig

	if err := viper.UnmarshalKey("database", &config); err != nil {
		return nil, err
	}

	// Set defaults if not configured
	if config.MaxOpenConns == 0 {
		config.MaxOpenConns = DefaultMaxOpenConns
	}
	if config.MaxIdleConns == 0 {
		config.MaxIdleConns = DefaultMaxIdleConns
	}
	if config.ConnMaxLifetime == 0 {
		config.ConnMaxLifetime = DefaultConnMaxLifetime
	}
	if config.ConnMaxIdleTime == 0 {
		config.ConnMaxIdleTime = DefaultConnMaxIdleTime
	}

	return &config, nil
}