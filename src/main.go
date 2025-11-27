package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/hibiken/asynq"

	"melodee/internal/capacity"
	"melodee/internal/config"
	"melodee/internal/database"
	"melodee/internal/directory"
	"melodee/internal/handlers"
	"melodee/internal/media"
	"melodee/internal/middleware"
	"melodee/internal/services"
	open_subsonic_handlers "melodee/open_subsonic/handlers"
	open_subsonic_middleware "melodee/open_subsonic/middleware"
)

// Server represents the main application server that handles both internal and OpenSubsonic APIs
type Server struct {
	app            *fiber.App
	cfg            *config.AppConfig
	dbManager      *database.DatabaseManager
	repo           *services.Repository
	authService    *services.AuthService
	asynqClient    *asynq.Client
	asynqScheduler *asynq.Scheduler
	asynqInspector *asynq.Inspector
	capacityProbe  *capacity.CapacityProbe
}

// NewServer creates a new combined server instance
func NewServer() (*Server, error) {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize database manager
	dbManager, err := database.NewDatabaseManager(
		&config.DatabaseConfig{
			Host:            cfg.Database.Host,
			Port:            cfg.Database.Port,
			User:            cfg.Database.User,
			Password:        cfg.Database.Password,
			DBName:          cfg.Database.DBName,
			SSLMode:         cfg.Database.SSLMode,
			MaxOpenConns:    cfg.Database.MaxOpenConns,
			MaxIdleConns:    cfg.Database.MaxIdleConns,
			ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
			ConnMaxIdleTime: cfg.Database.ConnMaxIdleTime,
		},
		nil, // logger
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Initialize repository and services
	repo := services.NewRepository(dbManager.GetGormDB())
	authService := services.NewAuthService(dbManager.GetGormDB(), cfg.JWT.Secret)

	// Initialize Asynq client and scheduler
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{
		Addr: cfg.Redis.Address,
	})

	asynqScheduler := asynq.NewScheduler(
		asynq.RedisClientOpt{Addr: cfg.Redis.Address},
		&asynq.SchedulerOpts{},
	)

	asynqInspector := asynq.NewInspector(asynq.RedisClientOpt{
		Addr: cfg.Redis.Address,
	})

	// Initialize capacity probe
	capacityConfig := &config.CapacityConfig{
		Interval:         10 * time.Minute, // Default 10 minutes
		WarningThreshold: 80.0,             // 80% warning
		AlertThreshold:   90.0,             // 90% alert
		ProbeCommand:     "df --output=pcent /storage",
	}

	capacityProbe := capacity.NewCapacityProbe(
		capacityConfig,
		dbManager.GetGormDB(),
		asynqClient,
		asynqScheduler,
		nil, // logger placeholder
	)

	// Create the server instance
	server := &Server{
		cfg:            cfg,
		dbManager:      dbManager,
		repo:           repo,
		authService:    authService,
		asynqClient:    asynqClient,
		asynqScheduler: asynqScheduler,
		asynqInspector: asynqInspector,
		capacityProbe:  capacityProbe,
	}

	// Initialize Fiber app
	server.app = fiber.New(fiber.Config{
		AppName:      "Melodee",
		ServerHeader: "Melodee",
	})

	// Setup middleware
	server.setupMiddleware()

	// Setup routes
	server.setupRoutes()

	return server, nil
}

// setupMiddleware configures the application middleware
func (s *Server) setupMiddleware() {
	s.app.Use(recover.New())
	s.app.Use(logger.New())
	s.app.Use(helmet.New())

	// Request metrics middleware - apply to all routes
	metricsMiddleware := middleware.MetricsMiddleware()
	s.app.Use(metricsMiddleware)

	// Rate limiting middleware - apply to all routes
	rateLimiter := middleware.RateLimiterForPublicAPI()
	s.app.Use(rateLimiter)

	// CORS middleware with configuration
	corsConfig := cors.Config{
		AllowOrigins:     strings.Join(s.cfg.Server.CORS.AllowOrigins, ","),
		AllowMethods:     strings.Join(s.cfg.Server.CORS.AllowMethods, ","),
		AllowHeaders:     strings.Join(s.cfg.Server.CORS.AllowHeaders, ","),
		AllowCredentials: s.cfg.Server.CORS.AllowCredentials,
	}
	s.app.Use(cors.New(corsConfig))
}

// setupRoutes configures all API routes (both internal and OpenSubsonic)
func (s *Server) setupRoutes() {
	// Health check endpoint (for kubernetes readiness/liveness probes)
	healthHandler := handlers.NewHealthHandler(s.dbManager)
	s.app.Get("/healthz", healthHandler.HealthCheck)

	// Metrics endpoint for Prometheus
	metricsHandler := handlers.NewMetricsHandler()
	s.app.Get("/metrics", metricsHandler.Metrics())

	// Start capacity monitoring
	if err := s.capacityProbe.Start(); err != nil {
		log.Printf("Warning: Failed to start capacity probe: %v", err)
	}

	// Setup internal API routes
	s.setupInternalRoutes()

	// Setup OpenSubsonic API routes
	s.setupOpenSubsonicRoutes()
}

// setupInternalRoutes configures internal API routes
func (s *Server) setupInternalRoutes() {
	// Create internal API group
	internalAPI := s.app.Group("/api")

	// Authentication middleware for internal API
	authMiddleware := middleware.NewAuthMiddleware(s.authService)

	// Auth routes with stricter rate limiting
	authHandler := handlers.NewAuthHandler(s.authService)
	authRateLimiter := middleware.RateLimiterForAuth()
	auth := internalAPI.Group("/auth")
	auth.Post("/login", authRateLimiter, authHandler.Login)
	auth.Post("/refresh", authRateLimiter, authHandler.Refresh)
	auth.Post("/request-reset", authRateLimiter, authHandler.RequestReset)
	auth.Post("/reset", authRateLimiter, authHandler.ResetPassword)

	// Protected routes (require authentication)
	protected := internalAPI.Use(authMiddleware.JWTProtected())

	// User management (admin only for some operations)
	userHandler := handlers.NewUserHandler(s.repo, s.authService)
	users := protected.Group("/users")
	users.Get("/", authMiddleware.AdminOnly(), userHandler.GetUsers)
	users.Post("/", authMiddleware.AdminOnly(), userHandler.CreateUser)
	users.Get("/:id", userHandler.GetUser)
	users.Put("/:id", userHandler.UpdateUser)
	users.Delete("/:id", authMiddleware.AdminOnly(), userHandler.DeleteUser)

	// Playlist management
	playlistHandler := handlers.NewPlaylistHandler(s.repo)
	playlists := protected.Group("/playlists")
	playlists.Get("/", playlistHandler.GetPlaylists)
	playlists.Post("/", playlistHandler.CreatePlaylist)
	playlists.Get("/:id", playlistHandler.GetPlaylist)
	playlists.Put("/:id", playlistHandler.UpdatePlaylist)
	playlists.Delete("/:id", playlistHandler.DeletePlaylist)

	// Admin endpoints
	admin := protected.Group("/admin")
	admin.Use(authMiddleware.AdminOnly())

	// DLQ management
	dlqHandler := handlers.NewDLQHandler(s.asynqInspector, s.asynqClient)
	admin.Get("/jobs/dlq", dlqHandler.GetDLQItems)
	admin.Post("/jobs/dlq/requeue", dlqHandler.RequeueDLQItems)
	admin.Post("/jobs/dlq/purge", dlqHandler.PurgeDLQItems)
	admin.Get("/jobs/:id", dlqHandler.GetJobById)

	// Capacity monitoring and health
	healthMetricsHandler := handlers.NewHealthMetricsHandler(s.dbManager.GetGormDB(), s.cfg, s.capacityProbe, s.asynqInspector)
	admin.Get("/capacity", healthMetricsHandler.CapacityStatus)
	admin.Get("/capacity/:id", healthMetricsHandler.CapacityStatusForLibrary)
	admin.Post("/capacity/probe-now", healthMetricsHandler.ProbeCapacityNow)

	// Initialize directory services for media processing
	directorySvc := directory.NewDirectoryCodeGenerator(&directory.DirectoryCodeConfig{
		FormatPattern: "consonant_vowel",
		MaxLength:     10,
		MinLength:     2,
		UseSuffixes:   true,
		SuffixPattern: "-%d",
	}, s.dbManager.GetGormDB())

	pathResolver := directory.NewPathTemplateResolver(&directory.PathTemplateConfig{
		DefaultTemplate: "{library}/{artist_dir_code}/{artist}/{year} - {album}",
	})

	// Initialize checksum service for media processing
	checksumSvc := media.NewChecksumService(
		s.dbManager.GetGormDB(),
		&media.ChecksumConfig{
			Algorithm:     "SHA256",
			EnableCaching: true,
			CacheTTL:      24 * time.Hour,
			StoreLocation: "DB",
		},
	)

	// Initialize quarantine service
	quarantineSvc := media.NewDefaultQuarantineService(s.dbManager.GetGormDB())

	// Initialize media service
	mediaSvc := media.NewMediaService(
		s.dbManager.GetGormDB(),
		directorySvc,
		pathResolver,
		quarantineSvc,
	)

	// Initialize media processor with all required services
	mediaProcessor := media.NewMediaProcessor(
		media.DefaultProcessingConfig(),
		s.dbManager.GetGormDB(),
		pathResolver,
		quarantineSvc, // Use the same quarantine service instance
		media.NewMediaFileValidator(&media.ValidationConfig{}),
		media.NewFFmpegProcessor(&media.FFmpegConfig{
			FFmpegPath: s.cfg.Processing.FFmpegPath,
			Profiles:   make(map[string]media.FFmpegProfile),
		}), // FFmpeg processor
		checksumSvc,
	)
	_ = mediaProcessor // Will be used in future endpoints

	// Library management
	libraryHandler := handlers.NewLibraryHandler(s.repo, mediaSvc, s.asynqClient, mediaSvc.QuarantineService)
	libraries := admin.Group("/libraries")
	libraries.Get("/", libraryHandler.GetLibraryStates)
	libraries.Get("/:id", libraryHandler.GetLibraryState)
	libraries.Get("/stats", libraryHandler.GetLibrariesStats)
	libraries.Post("/scan", libraryHandler.TriggerLibraryScan)
	libraries.Post("/process", libraryHandler.TriggerLibraryProcess)
	libraries.Post("/move-ok", libraryHandler.TriggerLibraryMoveOK)
	libraries.Get("/quarantine", libraryHandler.GetQuarantineItems)
	libraries.Post("/quarantine/:id/resolve", libraryHandler.ResolveQuarantineItem)
	libraries.Post("/quarantine/:id/requeue", libraryHandler.RequeueQuarantineItem)

	// Settings management
	settingsHandler := handlers.NewSettingsHandler(s.repo)
	admin.Get("/settings", settingsHandler.GetSettings)
	admin.Put("/settings/:key", settingsHandler.UpdateSetting)

	// Shares management
	sharesHandler := handlers.NewSharesHandler(s.repo)
	admin.Get("/shares", sharesHandler.GetShares)
	admin.Post("/shares", sharesHandler.CreateShare)
	admin.Put("/shares/:id", sharesHandler.UpdateShare)
	admin.Delete("/shares/:id", sharesHandler.DeleteShare)

	// Image management
	imageHandler := handlers.NewImageHandler(s.repo)
	images := protected.Group("/images")
	images.Post("/avatar", imageHandler.UploadAvatar)
	images.Get("/:id", imageHandler.GetImage)

	// V1 API endpoints (with pagination)
	albumsV1Handler := handlers.NewAlbumsV1Handler(s.repo)
	albumsV1 := protected.Group("/v1/Albums")
	albumsV1.Get("/", albumsV1Handler.GetAlbums)
	albumsV1.Get("/:id", albumsV1Handler.GetAlbum)
	albumsV1.Get("/recent", albumsV1Handler.GetRecentAlbums)
	albumsV1.Get("/:id/songs", albumsV1Handler.GetAlbumSongs)

	artistsV1Handler := handlers.NewArtistsV1Handler(s.repo)
	artistsV1 := protected.Group("/v1/Artists")
	artistsV1.Get("/", artistsV1Handler.GetArtists)
	artistsV1.Get("/:id", artistsV1Handler.GetArtist)
	artistsV1.Get("/recent", artistsV1Handler.GetRecentArtists)
	artistsV1.Get("/:id/albums", artistsV1Handler.GetArtistAlbums)
	artistsV1.Get("/:id/songs", artistsV1Handler.GetArtistSongs)

	tracksV1Handler := handlers.NewTracksV1Handler(s.repo)
	tracksV1 := protected.Group("/v1/Tracks")
	tracksV1.Get("/", tracksV1Handler.GetTracks)
	tracksV1.Get("/:id", tracksV1Handler.GetTrack)
	tracksV1.Get("/recent", tracksV1Handler.GetRecentTracks)
	tracksV1.Post("/starred/:id/:isStarred", tracksV1Handler.ToggleTrackStarred)
	tracksV1.Post("/setrating/:id/:rating", tracksV1Handler.SetTrackRating)

	// Search
	searchHandler := handlers.NewSearchHandler(s.repo)
	protected.Get("/search", middleware.NewExpensiveEndpointRateLimiter(), searchHandler.Search)
}

// setupOpenSubsonicRoutes configures OpenSubsonic API routes
func (s *Server) setupOpenSubsonicRoutes() {
	// Create OpenSubsonic API group with /rest prefix (standard for OpenSubsonic)
	rest := s.app.Group("/rest")

	// OpenSubsonic authentication middleware
	openSubsonicAuth := open_subsonic_middleware.NewOpenSubsonicAuthMiddleware(s.dbManager.GetGormDB(), s.cfg.JWT.Secret)

	// Initialize FFmpeg processor
	ffmpegConfig := &media.FFmpegConfig{
		FFmpegPath: s.cfg.Processing.FFmpegPath,
		Profiles:   make(map[string]media.FFmpegProfile),
	}
	for name, cmd := range s.cfg.Processing.Profiles {
		ffmpegConfig.Profiles[name] = media.FFmpegProfile{
			Name:        name,
			CommandLine: cmd,
		}
	}
	ffmpegProcessor := media.NewFFmpegProcessor(ffmpegConfig)

	// Initialize transcode service with caching
	transcodeService := media.NewTranscodeService(
		ffmpegProcessor,
		s.cfg.Processing.TranscodeCache.CacheDir,
		s.cfg.Processing.TranscodeCache.MaxSize*1024*1024, // Convert MB to bytes
	)

	// Create handlers for OpenSubsonic endpoints
	browsingHandler := open_subsonic_handlers.NewBrowsingHandler(s.repo.GetDB())
	mediaHandler := open_subsonic_handlers.NewMediaHandler(s.repo.GetDB(), s.cfg, transcodeService)
	searchHandler := open_subsonic_handlers.NewSearchHandler(s.repo.GetDB())
	playlistHandler := open_subsonic_handlers.NewPlaylistHandler(s.repo.GetDB())
	userHandler := open_subsonic_handlers.NewUserHandler(s.repo.GetDB())
	systemHandler := open_subsonic_handlers.NewSystemHandler(s.repo)

	// Browsing endpoints
	rest.Get("/getMusicFolders.view", openSubsonicAuth.Authenticate, browsingHandler.GetMusicFolders)
	rest.Get("/getIndexes.view", openSubsonicAuth.Authenticate, browsingHandler.GetIndexes)
	rest.Get("/getArtists.view", openSubsonicAuth.Authenticate, browsingHandler.GetArtists)
	rest.Get("/getArtist.view", openSubsonicAuth.Authenticate, browsingHandler.GetArtist)
	rest.Get("/getAlbumInfo.view", openSubsonicAuth.Authenticate, browsingHandler.GetAlbumInfo)
	rest.Get("/getMusicDirectory.view", openSubsonicAuth.Authenticate, browsingHandler.GetMusicDirectory)
	rest.Get("/getAlbum.view", openSubsonicAuth.Authenticate, browsingHandler.GetAlbum)
	rest.Get("/getSong.view", openSubsonicAuth.Authenticate, browsingHandler.GetSong)
	rest.Get("/getGenres.view", openSubsonicAuth.Authenticate, browsingHandler.GetGenres)

	// Media retrieval endpoints
	rest.Get("/stream.view", openSubsonicAuth.Authenticate, mediaHandler.Stream)
	rest.Get("/download.view", openSubsonicAuth.Authenticate, mediaHandler.Download)
	rest.Get("/getCoverArt.view", openSubsonicAuth.Authenticate, mediaHandler.GetCoverArt)
	rest.Get("/getAvatar.view", openSubsonicAuth.Authenticate, mediaHandler.GetAvatar)

	// Searching endpoints
	rest.Get("/search.view", openSubsonicAuth.Authenticate, middleware.NewExpensiveEndpointRateLimiter(), searchHandler.Search)
	rest.Get("/search2.view", openSubsonicAuth.Authenticate, middleware.NewExpensiveEndpointRateLimiter(), searchHandler.Search2)
	rest.Get("/search3.view", openSubsonicAuth.Authenticate, middleware.NewExpensiveEndpointRateLimiter(), searchHandler.Search3)

	// Playlist endpoints
	rest.Get("/getPlaylists.view", openSubsonicAuth.Authenticate, playlistHandler.GetPlaylists)
	rest.Get("/getPlaylist.view", openSubsonicAuth.Authenticate, playlistHandler.GetPlaylist)
	rest.Get("/createPlaylist.view", openSubsonicAuth.Authenticate, playlistHandler.CreatePlaylist)
	rest.Get("/updatePlaylist.view", openSubsonicAuth.Authenticate, playlistHandler.UpdatePlaylist)
	rest.Get("/deletePlaylist.view", openSubsonicAuth.Authenticate, playlistHandler.DeletePlaylist)

	// User management endpoints
	rest.Get("/getUser.view", openSubsonicAuth.Authenticate, userHandler.GetUser)
	rest.Get("/getUsers.view", openSubsonicAuth.Authenticate, userHandler.GetUsers)
	rest.Get("/createUser.view", openSubsonicAuth.Authenticate, userHandler.CreateUser)
	rest.Get("/updateUser.view", openSubsonicAuth.Authenticate, userHandler.UpdateUser)
	rest.Get("/deleteUser.view", openSubsonicAuth.Authenticate, userHandler.DeleteUser)

	// System endpoints
	rest.Get("/ping.view", systemHandler.Ping)
	rest.Get("/getLicense.view", systemHandler.GetLicense)
}

// Start starts the server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
	log.Printf("Starting Melodee server on %s", addr)

	// Migrations are handled via init-scripts/001_schema.sql

	return s.app.Listen(addr)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	// Give some time for in-flight requests to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.app.ShutdownWithContext(ctx); err != nil {
		log.Printf("Error during server shutdown: %v", err)
		return err
	}

	// Close database connection
	if s.dbManager != nil {
		if sqlDB, err := s.dbManager.GetGormDB().DB(); err != nil {
			log.Printf("Error getting SQL DB for closing: %v", err)
		} else {
			if err := sqlDB.Close(); err != nil {
				log.Printf("Error closing database: %v", err)
			}
		}
	}

	// Close Asynq connections
	if s.asynqClient != nil {
		s.asynqClient.Close()
	}

	if s.asynqScheduler != nil {
		s.asynqScheduler.Shutdown()
	}

	return nil
}

// Main entry point for the entire application
func main() {
	// Create the server
	server, err := NewServer()
	if err != nil {
		log.Fatal("Failed to create server:", err)
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the server in a goroutine
	go func() {
		if err := server.Start(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	log.Printf("Received signal %s, shutting down...", sig)

	// Shutdown gracefully
	if err := server.Shutdown(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	} else {
		log.Println("Server shutdown completed successfully")
	}
}
