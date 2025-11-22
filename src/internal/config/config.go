package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// ProcessingConfig represents media processing configuration
type ProcessingConfig struct {
	FFmpegPath    string            `mapstructure:"ffmpeg_path"`
	Profiles      map[string]string `mapstructure:"profiles"` // name -> command template
	MaxBitrate    int               `mapstructure:"max_bitrate"`
	DefaultFormat string            `mapstructure:"default_format"`
}

// AppConfig represents the main application configuration
type AppConfig struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	JWT        JWTConfig        `mapstructure:"jwt"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Processing ProcessingConfig `mapstructure:"processing"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	Host         string        `mapstructure:"host"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

// JWTConfig represents JWT configuration
type JWTConfig struct {
	Secret       string        `mapstructure:"secret"`
	AccessExpiry time.Duration `mapstructure:"access_expiry"`
	RefreshExpiry time.Duration `mapstructure:"refresh_expiry"`
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
	Address  string        `mapstructure:"address"`
	Password string        `mapstructure:"password"`
	DB       int           `mapstructure:"db"`
	PoolSize int           `mapstructure:"pool_size"`
	Timeout  time.Duration `mapstructure:"timeout"`
}

// LoadConfig loads application configuration from various sources
func LoadConfig() (*AppConfig, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("./src/config")

	// Set default values
	viper.SetDefault("server.port", 3000)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.read_timeout", 15*time.Second)
	viper.SetDefault("server.write_timeout", 15*time.Second)
	viper.SetDefault("server.idle_timeout", 60*time.Second)
	
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "")
	viper.SetDefault("database.dbname", "melodee")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.max_open_conns", 50)
	viper.SetDefault("database.max_idle_conns", 25)
	viper.SetDefault("database.conn_max_lifetime", 30*time.Minute)
	viper.SetDefault("database.conn_max_idle_time", 15*time.Minute)
	
	viper.SetDefault("jwt.secret", "default-secret-key-change-in-production")
	viper.SetDefault("jwt.access_expiry", 15*time.Minute)
	viper.SetDefault("jwt.refresh_expiry", 14*24*time.Hour) // 14 days
	
	viper.SetDefault("redis.address", "localhost:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.pool_size", 10)
	viper.SetDefault("redis.timeout", 5*time.Second)

	// Processing configuration defaults
	viper.SetDefault("processing.ffmpeg_path", "ffmpeg")
	viper.SetDefault("processing.max_bitrate", 320) // in kbps
	viper.SetDefault("processing.default_format", "mp3")
	// Default profiles for transcoding
	viper.SetDefault("processing.profiles", map[string]string{
		"mp3":    "-c:a libmp3lame -b:a %vk",
		"ogg":    "-c:a libvorbis -b:a %vk",
		"flac":   "-c:a flac",
		"opus":   "-c:a libopus -b:a %vk",
		"wav":    "-c:a pcm_s16le",
	})

	// Read configuration file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found, using defaults
	}

	// Read from environment variables
	viper.AutomaticEnv()

	var config AppConfig
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation error: %w", err)
	}

	return &config, nil
}

// validateConfig validates the configuration values
func validateConfig(config *AppConfig) error {
	if config.JWT.Secret == "default-secret-key-change-in-production" {
		return fmt.Errorf("JWT secret is using default value - please change in production")
	}

	if config.JWT.Secret == "" {
		return fmt.Errorf("JWT secret cannot be empty")
	}

	if config.JWT.AccessExpiry <= 0 {
		return fmt.Errorf("JWT access expiry must be positive")
	}

	if config.JWT.RefreshExpiry <= 0 {
		return fmt.Errorf("JWT refresh expiry must be positive")
	}

	return nil
}

// validateConfig validates the configuration values
func validateConfig(config *AppConfig) error {
	if config.JWT.Secret == "default-secret-key-change-in-production" {
		return fmt.Errorf("JWT secret is using default value - please change in production")
	}

	if config.JWT.Secret == "" {
		return fmt.Errorf("JWT secret cannot be empty")
	}

	if config.JWT.AccessExpiry <= 0 {
		return fmt.Errorf("JWT access expiry must be positive")
	}

	if config.JWT.RefreshExpiry <= 0 {
		return fmt.Errorf("JWT refresh expiry must be positive")
	}

	// Validate processing configuration
	if err := validateProcessingConfig(&config.Processing); err != nil {
		return fmt.Errorf("processing config validation error: %w", err)
	}

	return nil
}

// validateProcessingConfig validates processing-specific configuration
func validateProcessingConfig(config *ProcessingConfig) error {
	if config.FFmpegPath == "" {
		return fmt.Errorf("FFmpeg path cannot be empty")
	}

	// Check if FFmpeg binary exists and is executable
	if err := utils.CheckFFmpeg(config.FFmpegPath); err != nil {
		return fmt.Errorf("FFmpeg validation failed: %w", err)
	}

	// Validate profiles
	if config.Profiles == nil || len(config.Profiles) == 0 {
		return fmt.Errorf("at least one transcoding profile must be defined")
	}

	// Validate max bitrate
	if config.MaxBitrate <= 0 {
		return fmt.Errorf("max bitrate must be positive")
	}

	return nil
}