package config

import (
	"time"

	"github.com/spf13/viper"
)

// DatabaseConfig represents the database configuration
type DatabaseConfig struct {
	Host              string        `mapstructure:"host"`
	Port              int           `mapstructure:"port"`
	User              string        `mapstructure:"user"`
	Password          string        `mapstructure:"password"`
	DBName            string        `mapstructure:"dbname"`
	SSLMode           string        `mapstructure:"sslmode"`
	MaxOpenConns      int           `mapstructure:"max_open_conns"`      // Default: 50
	MaxIdleConns      int           `mapstructure:"max_idle_conns"`      // Default: 25
	ConnMaxLifetime   time.Duration `mapstructure:"conn_max_lifetime"`   // Default: 30 minutes
	ConnMaxIdleTime   time.Duration `mapstructure:"conn_max_idle_time"`  // Default: 15 minutes
}

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