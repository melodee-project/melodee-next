# Configuration and Entry Point Structure Plan

## Overview
This document outlines the configuration management system and entry point structure for the Go-based Melodee system. The design focuses on flexibility, security, and maintainability while supporting the microservices architecture planned for the system.

## Configuration Management

### Configuration Structure
The configuration will be organized using a hierarchical structure that supports multiple environments (development, staging, production) and allows for both file-based and environment variable configuration.

```go
// Application configuration structure
type AppConfig struct {
    Server     ServerConfig     `mapstructure:"server"`
    Database   DatabaseConfig   `mapstructure:"database"`
    Redis      RedisConfig      `mapstructure:"redis"`
    Logging    LoggingConfig    `mapstructure:"logging"`
    Security   SecurityConfig   `mapstructure:"security"`
    Features   FeatureConfig    `mapstructure:"features"`
    Paths      PathConfig       `mapstructure:"paths"`
    Processing ProcessingConfig `mapstructure:"processing"`
    Directory  DirectoryConfig  `mapstructure:"directory"`
}

// Server configuration
type ServerConfig struct {
    Host         string        `mapstructure:"host"`         // Server host address
    Port         int           `mapstructure:"port"`         // Server port
    ReadTimeout  time.Duration `mapstructure:"read_timeout"` // Read timeout for HTTP requests
    WriteTimeout time.Duration `mapstructure:"write_timeout"` // Write timeout for HTTP requests
    IdleTimeout  time.Duration `mapstructure:"idle_timeout"`  // Idle timeout for connections
    CORS         CORSConfig    `mapstructure:"cors"`         // Cross-Origin Resource Sharing settings
    TLS          TLSConfig     `mapstructure:"tls"`          // TLS/SSL settings
}

// CORS configuration
type CORSConfig struct {
    AllowOrigins []string `mapstructure:"allow_origins"`
    AllowMethods []string `mapstructure:"allow_methods"`
    AllowHeaders []string `mapstructure:"allow_headers"`
    ExposeHeaders []string `mapstructure:"expose_headers"`
    AllowCredentials bool   `mapstructure:"allow_credentials"`
}

// TLS configuration
type TLSConfig struct {
    Enabled  bool   `mapstructure:"enabled"`
    CertFile string `mapstructure:"cert_file"`
    KeyFile  string `mapstructure:"key_file"`
}

// Database configuration
type DatabaseConfig struct {
    Host            string        `mapstructure:"host"`
    Port            int           `mapstructure:"port"`
    User            string        `mapstructure:"user"`
    Password        string        `mapstructure:"password"`  // Should be loaded from environment
    DBName          string        `mapstructure:"dbname"`
    SSLMode         string        `mapstructure:"sslmode"`
    MaxOpenConns    int           `mapstructure:"max_open_conns"`
    MaxIdleConns    int           `mapstructure:"max_idle_conns"`
    ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
    ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
}

// Redis configuration
type RedisConfig struct {
    Addr     string        `mapstructure:"addr"`
    Password string        `mapstructure:"password"` // Should be loaded from environment
    DB       int           `mapstructure:"db"`
    PoolSize int           `mapstructure:"pool_size"`
    Timeout  time.Duration `mapstructure:"timeout"`
}

// Logging configuration
type LoggingConfig struct {
    Level       string `mapstructure:"level"`        // log level: debug, info, warn, error
    Format      string `mapstructure:"format"`       // format: json, console
    Output      string `mapstructure:"output"`       // output: stdout, stderr, file path
    DisableTime bool   `mapstructure:"disable_time"` // disable time in log output
    TimeFormat  string `mapstructure:"time_format"`  // time format
}

// Security configuration
type SecurityConfig struct {
    JWT JWTConfig `mapstructure:"jwt"` // JWT configuration
    RateLimit RateLimitConfig `mapstructure:"rate_limit"` // Rate limiting configuration
    APIKeys APIKeyConfig `mapstructure:"api_keys"` // API key management
}

// JWT configuration
type JWTConfig struct {
    Secret    string        `mapstructure:"secret"`     // Should be loaded from environment
    Issuer    string        `mapstructure:"issuer"`     // JWT issuer
    Audience  string        `mapstructure:"audience"`   // JWT audience
    ExpiresIn time.Duration `mapstructure:"expires_in"` // Token expiration time
    RefreshExpiry time.Duration `mapstructure:"refresh_expiry"` // Refresh token expiration
}

// Rate limiting configuration
type RateLimitConfig struct {
    Enabled     bool          `mapstructure:"enabled"`
    Requests    int           `mapstructure:"requests"`    // Number of requests allowed
    Window      time.Duration `mapstructure:"window"`     // Time window for rate limiting
    Message     string        `mapstructure:"message"`    // Message for rate limited responses
    StatusCode  int           `mapstructure:"status_code"` // Status code for rate limited responses
}

// API key configuration
type APIKeyConfig struct {
    Length    int  `mapstructure:"length"`     // Default API key length
    Expiry    time.Duration `mapstructure:"expiry"`    // API key expiry time
    AutoGenerate bool `mapstructure:"auto_generate"` // Whether to auto-generate API keys for new users
}

// Feature configuration
type FeatureConfig struct {
    OpenSubsonicAPI bool `mapstructure:"open_subsonic_api"` // Enable OpenSubsonic API compatibility
    UserRegistration bool `mapstructure:"user_registration"` // Enable user registration
    SharingEnabled bool `mapstructure:"sharing_enabled"`    // Enable content sharing
    ScrobblingEnabled bool `mapstructure:"scrobbling_enabled"` // Enable scrobbling
    TranscodingEnabled bool `mapstructure:"transcoding_enabled"` // Enable transcoding
    MetadataEditing bool `mapstructure:"metadata_editing"`  // Enable metadata editing
}

// Path configuration
type PathConfig struct {
    DataDir          string            `mapstructure:"data_dir"`           // Base directory for application data
    StorageDir       string            `mapstructure:"storage_dir"`        // Base directory for music files
    InboundDir       string            `mapstructure:"inbound_dir"`        // Directory for inbound music files
    StagingDir       string            `mapstructure:"staging_dir"`        // Directory for staging music files
    UserImagesDir    string            `mapstructure:"user_images_dir"`    // Directory for user images
    PlaylistDir      string            `mapstructure:"playlist_dir"`       // Directory for playlist files
    TempDir          string            `mapstructure:"temp_dir"`           // Temporary directory
    AllowedExtensions map[string]bool   `mapstructure:"allowed_extensions"` // Allowed file extensions
}

// Processing configuration
type ProcessingConfig struct {
    Concurrency    int           `mapstructure:"concurrency"`     // Number of concurrent processing operations
    BatchSize      int           `mapstructure:"batch_size"`      // Batch size for processing operations
    MaxFileSize    int64         `mapstructure:"max_file_size"`   // Maximum file size in bytes
    TempDir        string        `mapstructure:"temp_dir"`       // Directory for temporary processing files
    CleanupAfter   time.Duration `mapstructure:"cleanup_after"`   // Clean up temporary files after this duration
    Conversion     ConversionConfig `mapstructure:"conversion"`   // Conversion settings
}

// Conversion configuration
type ConversionConfig struct {
    Enabled    bool    `mapstructure:"enabled"`
    Bitrate    int     `mapstructure:"bitrate"`     // Target bitrate in kbps
    SampleRate int     `mapstructure:"sample_rate"` // Target sample rate in Hz
    Format     string  `mapstructure:"format"`      // Target format (mp3, flac, etc.)
    FFmpegPath string  `mapstructure:"ffmpeg_path"` // Path to FFmpeg binary
}

// Directory organization configuration
type DirectoryConfig struct {
    Template string            `mapstructure:"template"` // Default directory template
    CodeConfig DirectoryCodeConfig `mapstructure:"code_config"` // Directory code configuration
    MaxDepth int               `mapstructure:"max_depth"` // Maximum directory depth
    ReservedNames map[string]bool `mapstructure:"reserved_names"` // Reserved directory names
}
```

### Configuration Loader

```go
// Configuration loader with multiple source support
type ConfigLoader struct {
    logger *zerolog.Logger
    viper  *viper.Viper
}

// NewConfigLoader creates a new configuration loader
func NewConfigLoader(logger *zerolog.Logger) *ConfigLoader {
    v := viper.New()
    
    // Set configuration file paths
    v.SetConfigName("config") // name of config file (without extension)
    v.SetConfigType("yaml")   // REQUIRED if the config file does not have the extension in the name
    v.AddConfigPath("/etc/melodee/") // path to look for the config file in
    v.AddConfigPath("$HOME/.melodee") // call multiple times to add many search paths
    v.AddConfigPath(".") // optionally look for config in the working directory
    
    // Set default values
    setDefaults(v)
    
    // Read environment variables with prefix MELODEE_
    v.SetEnvPrefix("MELODEE")
    v.AutomaticEnv()
    
    // Use a more flexible delimiter for nested environment variables
    v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
    
    return &ConfigLoader{
        logger: logger,
        viper:  v,
    }
}

// Load configuration from all sources
func (c *ConfigLoader) Load() (*AppConfig, error) {
    // Try to read the config file
    if err := c.viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); ok {
            c.logger.Info().Msg("Config file not found, using environment variables and defaults")
        } else {
            c.logger.Warn().Err(err).Msg("Error reading config file")
        }
    } else {
        c.logger.Info().Str("file", c.viper.ConfigFileUsed()).Msg("Using config file")
    }
    
    // Unmarshal configuration
    var config AppConfig
    if err := c.viper.Unmarshal(&config); err != nil {
        return nil, fmt.Errorf("failed to unmarshal config: %w", err)
    }
    
    // Validate the configuration
    if err := validateConfig(&config); err != nil {
        return nil, fmt.Errorf("configuration validation failed: %w", err)
    }
    
    return &config, nil
}

// Set default configuration values
func setDefaults(v *viper.Viper) {
    // Server defaults
    v.SetDefault("server.host", "0.0.0.0")
    v.SetDefault("server.port", 8080)
    v.SetDefault("server.read_timeout", 30*time.Second)
    v.SetDefault("server.write_timeout", 60*time.Second)
    v.SetDefault("server.idle_timeout", 120*time.Second)
    
    // Database defaults
    v.SetDefault("database.host", "localhost")
    v.SetDefault("database.port", 5432)
    v.SetDefault("database.sslmode", "disable")
    v.SetDefault("database.max_open_conns", 50)
    v.SetDefault("database.max_idle_conns", 25)
    v.SetDefault("database.conn_max_lifetime", 30*time.Minute)
    v.SetDefault("database.conn_max_idle_time", 15*time.Minute)
    
    // Redis defaults
    v.SetDefault("redis.addr", "localhost:6379")
    v.SetDefault("redis.db", 0)
    v.SetDefault("redis.pool_size", 20)
    v.SetDefault("redis.timeout", 5*time.Second)
    
    // Logging defaults
    v.SetDefault("logging.level", "info")
    v.SetDefault("logging.format", "json")
    v.SetDefault("logging.output", "stdout")
    
    // Security defaults
    v.SetDefault("security.rate_limit.enabled", true)
    v.SetDefault("security.rate_limit.requests", 100)
    v.SetDefault("security.rate_limit.window", 15*time.Minute)
    v.SetDefault("security.rate_limit.message", "Rate limit exceeded")
    v.SetDefault("security.rate_limit.status_code", 429)
    
    // JWT defaults
    v.SetDefault("security.jwt.expires_in", 24*time.Hour)
    v.SetDefault("security.jwt.refresh_expiry", 7*24*time.Hour)
    
    // Feature defaults
    v.SetDefault("features.open_subsonic_api", true)
    v.SetDefault("features.user_registration", true)
    v.SetDefault("features.sharing_enabled", true)
    v.SetDefault("features.scrobbling_enabled", true)
    v.SetDefault("features.transcoding_enabled", true)
    v.SetDefault("features.metadata_editing", true)
    
    // Path defaults
    v.SetDefault("paths.data_dir", "/var/lib/melodee")
    v.SetDefault("paths.storage_dir", "/storage/music")
    v.SetDefault("paths.inbound_dir", "/storage/inbound")
    v.SetDefault("paths.staging_dir", "/storage/staging")
    v.SetDefault("paths.user_images_dir", "/storage/user_images")
    v.SetDefault("paths.playlist_dir", "/storage/playlists")
    v.SetDefault("paths.temp_dir", "/tmp/melodee")
    
    // Processing defaults
    v.SetDefault("processing.concurrency", 10)
    v.SetDefault("processing.batch_size", 100)
    v.SetDefault("processing.max_file_size", 1024*1024*500) // 500MB
    v.SetDefault("processing.cleanup_after", 24*time.Hour)
    
    // Conversion defaults
    v.SetDefault("processing.conversion.enabled", true)
    v.SetDefault("processing.conversion.bitrate", 320)
    v.SetDefault("processing.conversion.sample_rate", 44100)
    v.SetDefault("processing.conversion.format", "mp3")
    v.SetDefault("processing.conversion.ffmpeg_path", "ffmpeg")
    
    // Directory defaults
    v.SetDefault("directory.template", "{artist_dir_code}/{artist}/{year} - {album}")
    v.SetDefault("directory.max_depth", 5)
    v.SetDefault("directory.code_config.max_length", 10)
    v.SetDefault("directory.code_config.min_length", 2)
    v.SetDefault("directory.code_config.use_suffixes", true)
    v.SetDefault("directory.code_config.suffix_pattern", "-%d")
}
```

### Configuration Validation

```go
// Validate the loaded configuration
func validateConfig(config *AppConfig) error {
    var errs []string
    
    // Validate server configuration
    if config.Server.Port < 1 || config.Server.Port > 65535 {
        errs = append(errs, "server.port must be between 1 and 65535")
    }
    
    // Validate database configuration
    if config.Database.Host == "" {
        errs = append(errs, "database.host cannot be empty")
    }
    if config.Database.User == "" {
        errs = append(errs, "database.user cannot be empty")
    }
    if config.Database.DBName == "" {
        errs = append(errs, "database.dbname cannot be empty")
    }
    
    // Validate Redis configuration
    if config.Redis.Addr == "" {
        errs = append(errs, "redis.addr cannot be empty")
    }
    
    // Validate path configuration
    if config.Paths.DataDir == "" {
        errs = append(errs, "paths.data_dir cannot be empty")
    }
    if config.Paths.StorageDir == "" {
        errs = append(errs, "paths.storage_dir cannot be empty")
    }
    
    // Validate directory configuration
    if config.Directory.Template == "" {
        errs = append(errs, "directory.template cannot be empty")
    }
    if config.Directory.CodeConfig.MaxLength < 2 {
        errs = append(errs, "directory.code_config.max_length must be at least 2")
    }
    
    if len(errs) > 0 {
        return fmt.Errorf("configuration validation errors: %s", strings.Join(errs, "; "))
    }
    
    return nil
}
```

## Entry Point Structure

### Main Application Structure

```go
// Application is the main application structure
type Application struct {
    config      *AppConfig
    logger      *zerolog.Logger
    dbManager   *DatabaseManager
    redisClient *redis.Client
    server      *fiber.App
    services    *Services
    scheduler   *asynq.Server
    shutdown    chan os.Signal
}

// Services contains all the services used by the application
type Services struct {
    UserService          *UserService
    ArtistService        *ArtistService
    AlbumService         *AlbumService
    SongService          *SongService
    PlaylistService      *PlaylistService
    DirectoryCodeService *DirectoryCodeService
    LibraryService       *LibraryService
    FileService          *FileService
    SearchService        *SearchService
    TranscodingService   *TranscodingService
}
```

### Main Entry Point

```go
// main.go - The main entry point for the application
func main() {
    // Create a new logger instance
    logger := initLogger()
    
    // Create configuration loader and load configuration
    configLoader := NewConfigLoader(logger)
    config, err := configLoader.Load()
    if err != nil {
        logger.Fatal().Err(err).Msg("Failed to load configuration")
    }
    
    // Create the main application instance
    app, err := NewApplication(config, logger)
    if err != nil {
        logger.Fatal().Err(err).Msg("Failed to create application")
    }
    
    // Initialize the application
    if err := app.Initialize(); err != nil {
        logger.Fatal().Err(err).Msg("Failed to initialize application")
    }
    
    // Start the application
    if err := app.Start(); err != nil {
        logger.Error().Err(err).Msg("Application stopped with error")
    }
    
    // Wait for shutdown signal
    app.WaitForShutdown()
}

// Initialize the logger with proper configuration
func initLogger() *zerolog.Logger {
    // Determine log level from environment or config
    logLevel := getLogLevel()
    
    // Create logger with console writer for development or JSON for production
    var writer io.Writer
    if isDevelopment() {
        writer = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
    } else {
        writer = os.Stdout
    }
    
    logger := zerolog.New(writer).
        Level(logLevel).
        With().
        Timestamp().
        Caller().
        Logger()
    
    return &logger
}

// Helper to get log level from environment
func getLogLevel() zerolog.Level {
    levelStr := os.Getenv("LOG_LEVEL")
    if levelStr == "" {
        levelStr = "info"
    }
    
    level, err := zerolog.ParseLevel(levelStr)
    if err != nil {
        level = zerolog.InfoLevel
    }
    
    return level
}

// Helper to determine if running in development
func isDevelopment() bool {
    env := os.Getenv("ENVIRONMENT")
    return env == "development" || env == "dev"
}
```

### Application Constructor

```go
// NewApplication creates a new application instance
func NewApplication(config *AppConfig, logger *zerolog.Logger) (*Application, error) {
    app := &Application{
        config:   config,
        logger:   logger,
        shutdown: make(chan os.Signal, 1),
    }
    
    return app, nil
}

// Initialize the application by setting up all dependencies
func (a *Application) Initialize() error {
    a.logger.Info().Msg("Initializing application...")
    
    // Initialize database connection
    if err := a.initDatabase(); err != nil {
        return fmt.Errorf("failed to initialize database: %w", err)
    }
    
    // Initialize Redis connection
    if err := a.initRedis(); err != nil {
        return fmt.Errorf("failed to initialize Redis: %w", err)
    }
    
    // Initialize services
    if err := a.initServices(); err != nil {
        return fmt.Errorf("failed to initialize services: %w", err)
    }
    
    // Initialize web server
    if err := a.initServer(); err != nil {
        return fmt.Errorf("failed to initialize server: %w", err)
    }
    
    // Initialize job scheduler
    if err := a.initScheduler(); err != nil {
        return fmt.Errorf("failed to initialize scheduler: %w", err)
    }
    
    // Run migrations if enabled
    if a.config.Database.Migrations.AutoRun {
        if err := a.runMigrations(); err != nil {
            return fmt.Errorf("failed to run migrations: %w", err)
        }
    }
    
    a.logger.Info().Msg("Application initialized successfully")
    return nil
}

// Initialize database connection
func (a *Application) initDatabase() error {
    dbManager, err := NewDatabaseManager(&a.config.Database, a.logger)
    if err != nil {
        return fmt.Errorf("failed to create database manager: %w", err)
    }
    
    a.dbManager = dbManager
    return nil
}

// Initialize Redis connection
func (a *Application) initRedis() error {
    redisClient := redis.NewClient(&redis.Options{
        Addr:     a.config.Redis.Addr,
        Password: a.config.Redis.Password,
        DB:       a.config.Redis.DB,
    })
    
    // Test Redis connection
    if err := redisClient.Ping(context.Background()).Err(); err != nil {
        return fmt.Errorf("failed to connect to Redis: %w", err)
    }
    
    a.redisClient = redisClient
    return nil
}

// Initialize all services
func (a *Application) initServices() error {
    // Create instances of all services with proper dependencies
    a.services = &Services{
        UserService: &UserService{
            db:     a.dbManager.gormDB,
            logger: a.logger,
        },
        ArtistService: &ArtistService{
            db:     a.dbManager.gormDB,
            logger: a.logger,
        },
        AlbumService: &AlbumService{
            db:     a.dbManager.gormDB,
            logger: a.logger,
        },
        SongService: &SongService{
            db:     a.dbManager.gormDB,
            logger: a.logger,
        },
        PlaylistService: &PlaylistService{
            db:     a.dbManager.gormDB,
            logger: a.logger,
        },
        DirectoryCodeService: &DirectoryCodeService{
            db:     a.dbManager.gormDB,
            config: &a.config.Directory.CodeConfig,
            logger: a.logger,
        },
        LibraryService: &LibraryService{
            db:     a.dbManager.gormDB,
            config: &a.config.Paths,
            logger: a.logger,
        },
        FileService: &FileService{
            config: &a.config.Paths,
            logger: a.logger,
        },
        SearchService: &SearchService{
            db:     a.dbManager.gormDB,
            redis:  a.redisClient,
            logger: a.logger,
        },
        TranscodingService: &TranscodingService{
            config: &a.config.Processing.Conversion,
            logger: a.logger,
        },
    }
    
    return nil
}

// Initialize the web server
func (a *Application) initServer() error {
    // Configure Fiber app
    fiberConfig := fiber.Config{
        ServerHeader:          "Melodee",
        AppName:               fmt.Sprintf("Melodee v%s", Version),
        ReadTimeout:           a.config.Server.ReadTimeout,
        WriteTimeout:          a.config.Server.WriteTimeout,
        IdleTimeout:           a.config.Server.IdleTimeout,
        BodyLimit:             int(a.config.Processing.MaxFileSize),
        ErrorHandler:          customErrorHandler,
    }
    
    a.server = fiber.New(fiberConfig)
    
    // Set up middleware
    a.setupMiddleware()
    
    // Register routes
    a.registerRoutes()
    
    return nil
}
```

### Middleware Setup

```go
// Setup middleware for the application
func (a *Application) setupMiddleware() {
    // Logging middleware
    a.server.Use(logger.New())
    
    // Recover middleware to handle panics
    a.server.Use(recover.New())
    
    // CORS middleware
    a.server.Use(cors.New(cors.Config{
        AllowOrigins: a.config.Server.CORS.AllowOrigins,
        AllowMethods: a.config.Server.CORS.AllowMethods,
        AllowHeaders: a.config.Server.CORS.AllowHeaders,
        AllowCredentials: a.config.Server.CORS.AllowCredentials,
    }))
    
    // Rate limiting middleware if enabled
    if a.config.Security.RateLimit.Enabled {
        a.server.Use(limiter.New(limiter.Config{
            Next: func(c *fiber.Ctx) bool {
                return c.IP() == "127.0.0.1" // Don't rate limit localhost
            },
            Max:        a.config.Security.RateLimit.Requests,
            Duration:   a.config.Security.RateLimit.Window,
            Message:    a.config.Security.RateLimit.Message,
            StatusCode: a.config.Security.RateLimit.StatusCode,
        }))
    }
    
    // Security headers
    a.server.Use(helmet.New())
}
```

### Route Registration

```go
// Register all routes for the application
func (a *Application) registerRoutes() {
    // Health check endpoint
    a.server.Get("/health", func(c *fiber.Ctx) error {
        return c.JSON(fiber.Map{
            "status": "ok",
            "timestamp": time.Now().Unix(),
        })
    })
    
    // API routes
    api := a.server.Group("/api")
    
    // User routes
    userRoutes := api.Group("/users")
    RegisterUserRoutes(userRoutes, a.services.UserService)
    
    // Artist routes
    artistRoutes := api.Group("/artists") 
    RegisterArtistRoutes(artistRoutes, a.services.ArtistService, a.services.DirectoryCodeService)
    
    // Album routes
    albumRoutes := api.Group("/albums")
    RegisterAlbumRoutes(albumRoutes, a.services.AlbumService)
    
    // Song routes
    songRoutes := api.Group("/songs")
    RegisterSongRoutes(songRoutes, a.services.SongService)
    
    // Playlist routes
    playlistRoutes := api.Group("/playlists")
    RegisterPlaylistRoutes(playlistRoutes, a.services.PlaylistService)
    
    // Library routes
    libraryRoutes := api.Group("/libraries")
    RegisterLibraryRoutes(libraryRoutes, a.services.LibraryService, a.services.FileService)
    
    // Directory code routes
    dirCodeRoutes := api.Group("/directory-codes")
    RegisterDirectoryCodeRoutes(dirCodeRoutes, a.services.DirectoryCodeService)
    
    // OpenSubsonic API routes
    if a.config.Features.OpenSubsonicAPI {
        openSubsonicAPI := a.server.Group("/rest")
        RegisterOpenSubsonicRoutes(openSubsonicAPI, a.services)
    }
}
```

### Environment Variable Map and Precedence
- Precedence: CLI flags > env vars > config file > defaults.
- Required env vars: `MELODEE_DATABASE_PASSWORD`, `MELODEE_JWT_SECRET`, `MELODEE_REDIS_PASSWORD` (if applicable), `MELODEE_BOOTSTRAP_ADMIN_PASSWORD`.
- Common service vars:
  - `MELODEE_API_ADDR` (default `:3000`)
  - `MELODEE_WEB_ADDR` (default `:8080`)
  - `MELODEE_REDIS_ADDR` (default `redis:6379`)
  - `MELODEE_DB_NAME` (default `melodee`)
  - `MELODEE_LOG_LEVEL` (default `info`)
  - `MELODEE_FEATURE_OPENSUBSONIC_API` (default `true`)
  - `MELODEE_FFMPEG_PATH` (default `/usr/bin/ffmpeg`)
  - External metadata: `MELODEE_MUSICBRAINZ_TOKEN`, `MELODEE_LASTFM_KEY`, `MELODEE_SPOTIFY_CLIENT_ID/SECRET` (optional; if set, enable fetching)
- Example `config.yaml`:
```yaml
server:
  host: 0.0.0.0
  port: 3000
database:
  host: db
  port: 5432
  user: melodee
  dbname: melodee
  sslmode: disable
redis:
  addr: redis:6379
logging:
  level: info
paths:
  storage_dir: /melodee/storage
  inbound_dir: /melodee/inbound
  staging_dir: /melodee/staging
features:
  open_subsonic_api: true
```

### Per-service Sample Configs
- API (`config.api.yaml`):
```yaml
server:
  host: 0.0.0.0
  port: 3000
database:
  host: db
  user: melodee
  dbname: melodee
redis:
  addr: redis:6379
features:
  open_subsonic_api: true
processing:
  ffmpeg_path: /usr/bin/ffmpeg
```
- Worker (`config.worker.yaml`):
```yaml
database:
  host: db
  user: melodee
redis:
  addr: redis:6379
processing:
  concurrency: 4
  ffmpeg_path: /usr/bin/ffmpeg
queues:
  critical: 5
  default: 10
  bulk: 2
```
- Web (`config.web.yaml`):
```yaml
server:
  host: 0.0.0.0
  port: 8080
api:
  base_url: http://localhost:3000
logging:
  level: info
```

### Environment Matrix (required vars)
| Env | Required Vars | Notes |
| --- | --- | --- |
| development | `MELODEE_DATABASE_PASSWORD`, `MELODEE_JWT_SECRET`, `MELODEE_BOOTSTRAP_ADMIN_PASSWORD`, `MELODEE_FFMPEG_PATH` | external API keys optional |
| staging | dev vars + `MELODEE_REDIS_PASSWORD` (if auth), `MELODEE_MUSICBRAINZ_TOKEN` | feature flags match prod |
| production | staging vars + `MELODEE_LASTFM_KEY`, `MELODEE_SPOTIFY_CLIENT_ID/SECRET` (if enabled) | secrets from vault/secret manager; no defaults |

Secret guidance: prefer env vars injected by secret manager; avoid committing secrets to config files. Mount service-specific YAML with non-secret defaults only.

### Application Start and Shutdown

```go
// Start the application server
func (a *Application) Start() error {
    a.logger.Info().Int("port", a.config.Server.Port).Str("host", a.config.Server.Host).Msg("Starting server")
    
    // Start the server in a goroutine
    go func() {
        addr := fmt.Sprintf("%s:%d", a.config.Server.Host, a.config.Server.Port)
        if a.config.Server.TLS.Enabled {
            // Start with TLS
            if err := a.server.ListenTLS(addr, a.config.Server.TLS.CertFile, a.config.Server.TLS.KeyFile); err != nil {
                a.logger.Error().Err(err).Msg("Failed to start HTTPS server")
            }
        } else {
            // Start without TLS
            if err := a.server.Listen(addr); err != nil {
                a.logger.Error().Err(err).Msg("Failed to start HTTP server")
            }
        }
    }()
    
    // Register for shutdown signals
    signal.Notify(a.shutdown, os.Interrupt, syscall.SIGTERM)
    
    return nil
}

// WaitForShutdown waits for shutdown signal and handles cleanup
func (a *Application) WaitForShutdown() {
    a.logger.Info().Msg("Waiting for shutdown signal...")
    <-a.shutdown
    
    a.logger.Info().Msg("Shutdown signal received, cleaning up...")
    
    // Create context with timeout for graceful shutdown
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Shutdown server gracefully
    if err := a.server.ShutdownWithContext(ctx); err != nil {
        a.logger.Error().Err(err).Msg("Error shutting down server")
    }
    
    // Close database connection
    if a.dbManager != nil && a.dbManager.sqlDB != nil {
        if err := a.dbManager.sqlDB.Close(); err != nil {
            a.logger.Error().Err(err).Msg("Error closing database connection")
        }
    }
    
    // Close Redis connection
    if a.redisClient != nil {
        if err := a.redisClient.Close(); err != nil {
            a.logger.Error().Err(err).Msg("Error closing Redis connection")
        }
    }
    
    // Shutdown scheduler
    if a.scheduler != nil {
        a.scheduler.Shutdown()
    }
    
    a.logger.Info().Msg("Application shutdown completed")
}
```

## Environment Configuration Files

### Development Configuration
```yaml
# config/development.yaml
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: "30s"
  write_timeout: "60s"
  idle_timeout: "120s"

database:
  host: "localhost"
  port: 5432
  user: "melodee_dev"
  password: "melodee_dev_password"  # In production, this would come from environment
  dbname: "melodee_dev"
  sslmode: "disable"
  max_open_conns: 10
  max_idle_conns: 5
  conn_max_lifetime: "30m"
  conn_max_idle_time: "15m"

redis:
  addr: "localhost:6379"
  db: 0
  pool_size: 10
  timeout: "5s"

logging:
  level: "debug"
  format: "console"  # Use console format for development
  output: "stdout"

features:
  open_subsonic_api: true
  user_registration: true
  sharing_enabled: true
  scrobbling_enabled: true
  transcoding_enabled: true
  metadata_editing: true

paths:
  data_dir: "./data"
  storage_dir: "./storage/music"
  inbound_dir: "./storage/inbound"
  staging_dir: "./storage/staging"
  user_images_dir: "./storage/user_images"
  playlist_dir: "./storage/playlists"
  temp_dir: "./temp"
```

### Production Configuration Template
```yaml
# config/production.yaml
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: "30s"
  write_timeout: "60s"
  idle_timeout: "120s"

database:
  host: "melodee-db"
  port: 5432
  user: "melodee"
  # Password loaded from environment variable MELODEE_DATABASE_PASSWORD
  dbname: "melodee"
  sslmode: "require"
  max_open_conns: 50
  max_idle_conns: 25
  conn_max_lifetime: "30m"
  conn_max_idle_time: "15m"

redis:
  addr: "melodee-redis:6379"
  db: 0
  pool_size: 20
  timeout: "5s"

logging:
  level: "info"
  format: "json"
  output: "stdout"

security:
  jwt:
    # Secret loaded from environment variable MELODEE_SECURITY_JWT_SECRET
    issuer: "Melodee"
    audience: "MelodeeAPI"
    expires_in: "24h"
    refresh_expiry: "168h"  # 7 days
  rate_limit:
    enabled: true
    requests: 1000
    window: "15m"
    message: "Rate limit exceeded"
    status_code: 429

features:
  open_subsonic_api: true
  user_registration: true
  sharing_enabled: true
  scrobbling_enabled: true
  transcoding_enabled: true
  metadata_editing: true

paths:
  data_dir: "/var/lib/melodee"
  storage_dir: "/storage/music"
  inbound_dir: "/storage/inbound"
  staging_dir: "/storage/staging"
  user_images_dir: "/storage/user_images"
  playlist_dir: "/storage/playlists"
  temp_dir: "/tmp/melodee"

processing:
  concurrency: 20
  batch_size: 500
  max_file_size: 524288000  # 500MB
  cleanup_after: "24h"
  conversion:
    enabled: true
    bitrate: 320
    sample_rate: 44100
    format: "mp3"
    ffmpeg_path: "/usr/bin/ffmpeg"
```

This configuration and entry point structure plan provides a robust, secure, and maintainable foundation for the Go-based Melodee system. The design emphasizes flexibility through configuration, security through environment variable handling, and maintainability through clear separation of concerns.
