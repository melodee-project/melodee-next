package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// AppConfig holds the main application configuration
type AppConfig struct {
	Server          ServerConfig    `mapstructure:"server"`
	Database        DatabaseConfig  `mapstructure:"database"`
	Redis           RedisConfig     `mapstructure:"redis"`
	JWT             JWTConfig       `mapstructure:"jwt"`
	Processing      ProcessingConfig `mapstructure:"processing"`
	Capacity        CapacityConfig  `mapstructure:"capacity"`
	Logging         LoggingConfig   `mapstructure:"logging"`
	Security        SecurityConfig  `mapstructure:"security"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	Host         string        `mapstructure:"host"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
	CORS         CORSConfig    `mapstructure:"cors"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	DBName          string        `mapstructure:"dbname"`
	SSLMode         string        `mapstructure:"sslmode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Address  string        `mapstructure:"address"`
	Password string        `mapstructure:"password"`
	DB       int           `mapstructure:"db"`
	PoolSize int           `mapstructure:"pool_size"`
	Timeout  time.Duration `mapstructure:"timeout"`
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret       string        `mapstructure:"secret"`
	AccessExpiry time.Duration `mapstructure:"access_expiry"`
	RefreshExpiry time.Duration `mapstructure:"refresh_expiry"`
}

// ProcessingConfig holds media processing configuration
type ProcessingConfig struct {
	FFmpegPath      string            `mapstructure:"ffmpeg_path"`
	Profiles        map[string]string `mapstructure:"profiles"`
	MaxConcurrent   int               `mapstructure:"max_concurrent"`
	MaxBitrate      int               `mapstructure:"max_bitrate"`
	DefaultFormat   string            `mapstructure:"default_format"`
	TranscodeCache  TranscodeCacheConfig `mapstructure:"transcode_cache"`
}

// TranscodeCacheConfig holds transcoding cache configuration
type TranscodeCacheConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	MaxSize     int64  `mapstructure:"max_size"` // in MB
	CacheDir    string `mapstructure:"cache_dir"`
	MaxAge      int64  `mapstructure:"max_age"`  // in hours
	MaxFiles    int    `mapstructure:"max_files"`
}

// CapacityConfig holds capacity monitoring configuration
type CapacityConfig struct {
	Enabled       bool          `mapstructure:"enabled"`
	Interval      time.Duration `mapstructure:"interval"`       // How often to check
	WarningThreshold float64    `mapstructure:"warning_threshold"` // Percentage for warning
	AlertThreshold float64     `mapstructure:"alert_threshold"`  // Percentage for alert
	Libraries     []string      `mapstructure:"libraries"`      // Paths to monitor
	ProbeCommand  string        `mapstructure:"probe_command"`  // Command to check usage
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level     string `mapstructure:"level"`
	Format    string `mapstructure:"format"` // "json", "text"
	File      string `mapstructure:"file"`   // Log file path (optional)
	MaxSize   int    `mapstructure:"max_size"` // Max file size in MB
	MaxAge    int    `mapstructure:"max_age"`  // Max age in days
	Compress  bool   `mapstructure:"compress"`
}

// SecurityConfig holds security configuration
type SecurityConfig struct {
	MaxRequestBodySize int64    `mapstructure:"max_request_body_size"` // in bytes
	AllowedHosts       []string `mapstructure:"allowed_hosts"`
	CORS               CORSConfig `mapstructure:"cors"`
	BasicAuth          BasicAuthConfig `mapstructure:"basic_auth"`
	RateLimiting       RateLimitConfig `mapstructure:"rate_limiting"`
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowOrigins     []string `mapstructure:"allow_origins"`
	AllowMethods     []string `mapstructure:"allow_methods"`
	AllowHeaders     []string `mapstructure:"allow_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
	ExposeHeaders    []string `mapstructure:"expose_headers"`
	MaxAge           int      `mapstructure:"max_age"`
}

// BasicAuthConfig holds basic authentication configuration
type BasicAuthConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled    bool          `mapstructure:"enabled"`
	Requests   int           `mapstructure:"requests"`   // Requests per window
	Window     time.Duration `mapstructure:"window"`     // Time window
	Message    string        `mapstructure:"message"`    // Rate limit message
	StatusCode int           `mapstructure:"status_code"` // HTTP status code for rate-limited requests
}

// DefaultAppConfig returns default configuration values
func DefaultAppConfig() *AppConfig {
	return &AppConfig{
		Server: ServerConfig{
			Port:        8080,
			Host:        "0.0.0.0",
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
			CORS: CORSConfig{
				AllowOrigins:     []string{"*"},
				AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
				AllowCredentials: false,
				MaxAge:           300,
			},
		},
		Database: DatabaseConfig{
			Host:            "localhost",
			Port:            5432,
			User:            "melodee_user",
			Password:        "default_password_change_in_prod",
			DBName:          "melodee",
			SSLMode:         "disable",
			MaxOpenConns:    25,
			MaxIdleConns:    10,
			ConnMaxLifetime: 30 * time.Minute,
			ConnMaxIdleTime: 15 * time.Minute,
		},
		Redis: RedisConfig{
			Address:  "localhost:6379",
			Password: "",
			DB:       0,
			PoolSize: 10,
			Timeout:  5 * time.Second,
		},
		JWT: JWTConfig{
			Secret:       "default-secret-key-change-in-production",
			AccessExpiry: 15 * time.Minute,
			RefreshExpiry: 24 * time.Hour,
		},
		Processing: ProcessingConfig{
			FFmpegPath:    "/usr/bin/ffmpeg",
			MaxConcurrent: 4,
			MaxBitrate:    320, // kbps
			DefaultFormat: "mp3",
			Profiles: map[string]string{
				"transcode_high":      "-c:a libmp3lame -b:a 320k -ar 44100 -ac 2",
				"transcode_mid":       "-c:a libmp3lame -b:a 192k -ar 44100 -ac 2",
				"transcode_opus_mobile": "-c:a libopus -b:a 96k -application audio",
			},
			TranscodeCache: TranscodeCacheConfig{
				Enabled:  true,
				MaxSize:  1024, // 1GB
				CacheDir: "/tmp/melodee-transcode-cache",
				MaxAge:   168,   // 7 days
				MaxFiles: 10000, // 10k files max
			},
		},
		Capacity: CapacityConfig{
			Enabled:        true,
			Interval:       10 * time.Minute,
			WarningThreshold: 80.0, // Percent
			AlertThreshold:  90.0, // Percent
			Libraries:      []string{"/storage"},
			ProbeCommand:   "df --output=pcent /storage",
		},
		Logging: LoggingConfig{
			Level:    "info",
			Format:   "json",
			MaxSize:  100, // MB
			MaxAge:   30,  // days
			Compress: true,
		},
		Security: SecurityConfig{
			MaxRequestBodySize: 10 * 1024 * 1024, // 10MB
			AllowedHosts:       []string{"*"},
			CORS: CORSConfig{
				AllowOrigins:     []string{"*"},
				AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
				AllowCredentials: false,
				MaxAge:           300,
			},
			BasicAuth: BasicAuthConfig{
				Enabled: false,
			},
			RateLimiting: RateLimitConfig{
				Enabled:    true,
				Requests:   100,
				Window:     1 * time.Minute,
				Message:    "Rate limit exceeded",
				StatusCode: 429,
			},
		},
	}
}

// LoadConfig loads application configuration from various sources
func LoadConfig() (*AppConfig, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("/etc/melodee/")
	
	// Set default values
	setDefaults()
	
	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Printf("Warning: could not read config file: %v", err)
		}
		// Continue with defaults and environment variables
	}

	// Automatically read from environment variables
	viper.AutomaticEnv()
	viper.SetEnvPrefix("MELODEE") // All env vars will be prefixed with MELODEE_
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Unmarshal the configuration
	var config AppConfig
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Apply any environment variable overrides
	applyEnvironmentOverrides(&config)

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.read_timeout", "15s")
	viper.SetDefault("server.write_timeout", "15s")
	viper.SetDefault("server.idle_timeout", "60s")

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "melodee_user")
	viper.SetDefault("database.password", "default_password_change_in_prod")
	viper.SetDefault("database.dbname", "melodee")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 10)
	viper.SetDefault("database.conn_max_lifetime", "30m")
	viper.SetDefault("database.conn_max_idle_time", "15m")

	// Redis defaults
	viper.SetDefault("redis.address", "localhost:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.pool_size", 10)
	viper.SetDefault("redis.timeout", "5s")

	// JWT defaults
	viper.SetDefault("jwt.secret", "default-secret-key-change-in-production")
	viper.SetDefault("jwt.access_expiry", "15m")
	viper.SetDefault("jwt.refresh_expiry", "24h")

	// Processing defaults
	viper.SetDefault("processing.ffmpeg_path", "/usr/bin/ffmpeg")
	viper.SetDefault("processing.max_concurrent", 4)
	viper.SetDefault("processing.max_bitrate", 320)
	viper.SetDefault("processing.default_format", "mp3")
	viper.SetDefault("processing.profiles.transcode_high", "-c:a libmp3lame -b:a 320k -ar 44100 -ac 2")
	viper.SetDefault("processing.profiles.transcode_mid", "-c:a libmp3lame -b:a 192k -ar 44100 -ac 2")
	viper.SetDefault("processing.profiles.transcode_opus_mobile", "-c:a libopus -b:a 96k -application audio")
	
	// Transcode cache defaults
	viper.SetDefault("processing.transcode_cache.enabled", true)
	viper.SetDefault("processing.transcode_cache.max_size", 1024) // 1GB
	viper.SetDefault("processing.transcode_cache.cache_dir", "/tmp/melodee-transcode-cache")
	viper.SetDefault("processing.transcode_cache.max_age", 168) // 7 days in hours
	viper.SetDefault("processing.transcode_cache.max_files", 10000)

	// Capacity defaults
	viper.SetDefault("capacity.enabled", true)
	viper.SetDefault("capacity.interval", "10m")
	viper.SetDefault("capacity.warning_threshold", 80.0)
	viper.SetDefault("capacity.alert_threshold", 90.0)
	viper.SetDefault("capacity.libraries", []string{"/storage"})
	viper.SetDefault("capacity.probe_command", "df --output=pcent /storage")

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.max_size", 100) // MB
	viper.SetDefault("logging.max_age", 30) // days
	viper.SetDefault("logging.compress", true)

	// Security defaults
	viper.SetDefault("security.max_request_body_size", 10485760) // 10MB
	viper.SetDefault("security.allowed_hosts", []string{"*"})
	viper.SetDefault("security.cors.allow_origins", []string{"*"})
	viper.SetDefault("security.cors.allow_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	viper.SetDefault("security.cors.allow_headers", []string{"Origin", "Content-Type", "Accept", "Authorization"})
	viper.SetDefault("security.cors.allow_credentials", false)
	viper.SetDefault("security.cors.max_age", 300)
	viper.SetDefault("security.basic_auth.enabled", false)
	viper.SetDefault("security.rate_limiting.enabled", true)
	viper.SetDefault("security.rate_limiting.requests", 100)
	viper.SetDefault("security.rate_limiting.window", "1m")
	viper.SetDefault("security.rate_limiting.message", "Rate limit exceeded")
	viper.SetDefault("security.rate_limiting.status_code", 429)
}

// applyEnvironmentOverrides applies configuration overrides from environment variables
func applyEnvironmentOverrides(config *AppConfig) {
	// Database overrides
	if dbHost := getEnv("MELODEE_DATABASE_HOST", ""); dbHost != "" {
		config.Database.Host = dbHost
	}
	if dbPort := getEnvInt("MELODEE_DATABASE_PORT", config.Database.Port); dbPort > 0 {
		config.Database.Port = dbPort
	}
	if dbUser := getEnv("MELODEE_DATABASE_USER", ""); dbUser != "" {
		config.Database.User = dbUser
	}
	if dbPass := getEnv("MELODEE_DATABASE_PASSWORD", ""); dbPass != "" {
		config.Database.Password = dbPass
	}
	if dbName := getEnv("MELODEE_DATABASE_DBNAME", ""); dbName != "" {
		config.Database.DBName = dbName
	}
	if dbSSLMode := getEnv("MELODEE_DATABASE_SSLMODE", ""); dbSSLMode != "" {
		config.Database.SSLMode = dbSSLMode
	}

	// Redis overrides
	if redisAddr := getEnv("MELODEE_REDIS_ADDRESS", ""); redisAddr != "" {
		config.Redis.Address = redisAddr
	}
	if redisPass := getEnv("MELODEE_REDIS_PASSWORD", ""); redisPass != "" {
		config.Redis.Password = redisPass
	}
	if redisDB := getEnvInt("MELODEE_REDIS_DB", config.Redis.DB); redisDB >= 0 {
		config.Redis.DB = redisDB
	}

	// JWT overrides
	if jwtSecret := getEnv("MELODEE_JWT_SECRET", ""); jwtSecret != "" {
		config.JWT.Secret = jwtSecret
	}

	// Processing overrides
	if ffmpegPath := getEnv("FFMPEG_PATH", ""); ffmpegPath != "" {
		config.Processing.FFmpegPath = ffmpegPath
	}

	// Capacity overrides
	if capEnabled := getEnvBool("MELODEE_CAPACITY_ENABLED", config.Capacity.Enabled); capEnabled {
		config.Capacity.Enabled = capEnabled
	}
	if capInterval := getEnvDuration("MELODEE_CAPACITY_INTERVAL", config.Capacity.Interval); capInterval > 0 {
		config.Capacity.Interval = capInterval
	}

	// Logging overrides
	if logLevel := getEnv("MELODEE_LOGGING_LEVEL", ""); logLevel != "" {
		config.Logging.Level = logLevel
	}
}

// getEnv gets an environment variable with a default fallback
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets an environment variable as an integer with a default fallback
func getEnvInt(key string, defaultValue int) int {
	if valueStr := os.Getenv(key); valueStr != "" {
		if value, err := strconv.Atoi(valueStr); err == nil {
			return value
		}
	}
	return defaultValue
}

// getEnvBool gets an environment variable as a boolean with a default fallback
func getEnvBool(key string, defaultValue bool) bool {
	if valueStr := os.Getenv(key); valueStr != "" {
		if value, err := strconv.ParseBool(valueStr); err == nil {
			return value
		}
	}
	return defaultValue
}

// getEnvDuration gets an environment variable as a time.Duration with a default fallback
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if valueStr := os.Getenv(key); valueStr != "" {
		if value, err := time.ParseDuration(valueStr); err == nil {
			return value
		}
	}
	return defaultValue
}

// Validate validates the configuration values
func (c *AppConfig) Validate() error {
	if c.JWT.Secret == "default-secret-key-change-in-production" {
		return fmt.Errorf("JWT secret is using default value - please change in production")
	}

	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT secret cannot be empty")
	}

	if c.Database.Host == "" {
		return fmt.Errorf("database host cannot be empty")
	}

	if c.Redis.Address == "" {
		return fmt.Errorf("redis address cannot be empty")
	}

	if c.Processing.FFmpegPath == "" {
		return fmt.Errorf("FFmpeg path cannot be empty")
	}

	// Validate processing profiles
	for name, profile := range c.Processing.Profiles {
		if profile == "" {
			return fmt.Errorf("processing profile '%s' cannot be empty", name)
		}
	}

	// Validate capacity thresholds
	if c.Capacity.WarningThreshold < 0 || c.Capacity.WarningThreshold > 100 {
		return fmt.Errorf("capacity warning threshold must be between 0 and 100, got: %f", c.Capacity.WarningThreshold)
	}
	if c.Capacity.AlertThreshold < 0 || c.Capacity.AlertThreshold > 100 {
		return fmt.Errorf("capacity alert threshold must be between 0 and 100, got: %f", c.Capacity.AlertThreshold)
	}

	return nil
}