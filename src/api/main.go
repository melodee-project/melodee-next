package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"melodee/internal/config"
	"melodee/internal/database"
	"melodee/internal/directory"
	"melodee/internal/handlers"
	"melodee/internal/media"
	"melodee/internal/middleware"
	"melodee/internal/services"
)

// APIServer represents the API server
type APIServer struct {
	app         *fiber.App
	cfg         *config.AppConfig
	db          *gorm.DB
	repo        *services.Repository
	authService *services.AuthService
	dbManager   *database.DatabaseManager
}

// NewAPIServer creates a new API server
func NewAPIServer(cfg *config.AppConfig, dbManager *database.DatabaseManager) *APIServer {
	db := dbManager.GetGormDB()

	server := &APIServer{
		cfg:         cfg,
		db:          db,
		dbManager:   dbManager,
		repo:        services.NewRepository(db),
		authService: services.NewAuthService(db, cfg.JWT.Secret),
	}

	// Initialize Fiber app
	server.app = fiber.New(fiber.Config{
		AppName:      "Melodee API Server",
		ServerHeader: "Melodee",
	})

	// Add middleware
	server.app.Use(recover.New())
	server.app.Use(logger.New())
	server.app.Use(cors.New())

	// Request metrics middleware
	server.app.Use(middleware.MetricsMiddleware())

	// Setup routes
	server.setupRoutes()

	// Register custom metrics
	handlers.RegisterCustomMetrics()

	return server
}

// setupRoutes configures the API routes
func (s *APIServer) setupRoutes() {
	// Create asynq client for enqueueing background jobs from the API
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: s.cfg.Redis.Address})
	asynqInspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: s.cfg.Redis.Address})

	// Create handlers
	authHandler := handlers.NewAuthHandler(s.authService)
	userHandler := handlers.NewUserHandler(s.repo, s.authService)
	playlistHandler := handlers.NewPlaylistHandler(s.repo)
	searchHandler := handlers.NewSearchHandler(s.repo)      // Add search handler
	healthHandler := handlers.NewHealthHandler(s.dbManager) // Pass the dbManager
	settingsHandler := handlers.NewSettingsHandler(s.repo)
	sharesHandler := handlers.NewSharesHandler(s.repo)
	dlqHandler := handlers.NewDLQHandler(asynqInspector, asynqClient)
	capacityHandler := handlers.NewCapacityHandler(s.db)

	// Health check route
	s.app.Get("/healthz", healthHandler.HealthCheck)

	// Metrics route
	metricsHandler := handlers.NewMetricsHandler()
	s.app.Get("/metrics", metricsHandler.Metrics())

	// Debug endpoint to test JWT validation (outside /api to bypass middleware)
	s.app.Get("/debug/token", func(c *fiber.Ctx) error {
		token := c.Get("Authorization")
		if token == "" {
			return c.JSON(fiber.Map{"error": "no token"})
		}
		token = strings.TrimPrefix(token, "Bearer ")

		// Try to validate the token manually
		user, err := s.authService.ValidateToken(token)
		if err != nil {
			return c.JSON(fiber.Map{
				"token_received": token[:20] + "...",
				"valid":          false,
				"error":          err.Error(),
				"secret_length":  len(s.authService.GetJWTSecret()),
			})
		}
		return c.JSON(fiber.Map{
			"token_received": token[:20] + "...",
			"valid":          true,
			"user":           user,
			"secret_length":  len(s.authService.GetJWTSecret()),
		})
	})

	// Auth routes (public, requires rate limiting)
	auth := s.app.Group("/api/auth")
	auth.Post("/login", middleware.RateLimiterForAuth(), authHandler.Login)
	auth.Post("/refresh", middleware.RateLimiterForAuth(), authHandler.Refresh)
	auth.Post("/request-reset", middleware.RateLimiterForAuth(), authHandler.RequestReset)
	auth.Post("/reset", middleware.RateLimiterForAuth(), authHandler.ResetPassword)

	// Create media services EARLY so we can use libraryHandler for stats route
	directoryService := directory.NewDirectoryCodeGenerator(directory.DefaultDirectoryCodeConfig(), s.db)
	pathTemplateResolver := directory.NewPathTemplateResolver(directory.DefaultPathTemplateConfig())
	quarantineSvc := media.NewDefaultQuarantineService(s.db)
	mediaSvc := media.NewMediaService(s.db, directoryService, pathTemplateResolver, quarantineSvc)
	libraryHandler := handlers.NewLibraryHandler(s.repo, mediaSvc, asynqClient, quarantineSvc)

	// Protected routes
	protected := s.app.Group("/api", middleware.NewAuthMiddleware(s.authService).JWTProtected())

	// User routes (admin only for list/create)
	users := protected.Group("/users")
	users.Get("/", middleware.NewAuthMiddleware(s.authService).AdminOnly(), userHandler.GetUsers)
	users.Post("/", middleware.NewAuthMiddleware(s.authService).AdminOnly(), userHandler.CreateUser)
	users.Get("/:id", userHandler.GetUser)
	users.Put("/:id", userHandler.UpdateUser)
	users.Delete("/:id", middleware.NewAuthMiddleware(s.authService).AdminOnly(), userHandler.DeleteUser)

	// Playlist routes
	playlists := protected.Group("/playlists")
	playlists.Get("/", playlistHandler.GetPlaylists)
	playlists.Post("/", playlistHandler.CreatePlaylist)
	playlists.Get("/:id", playlistHandler.GetPlaylist)
	playlists.Put("/:id", playlistHandler.UpdatePlaylist)
	playlists.Delete("/:id", playlistHandler.DeletePlaylist)

	// Settings routes
	protected.Get("/settings", settingsHandler.GetSettings)
	protected.Put("/settings/:key", settingsHandler.UpdateSetting)

	// Shares routes
	shares := protected.Group("/shares")
	shares.Get("/", sharesHandler.GetShares)
	shares.Post("/", sharesHandler.CreateShare)
	shares.Put("/:id", sharesHandler.UpdateShare)
	shares.Delete("/:id", sharesHandler.DeleteShare)

	// Admin routes (admin only)
	admin := protected.Group("/admin", middleware.NewAuthMiddleware(s.authService).AdminOnly())
	admin.Get("/jobs/dlq", dlqHandler.GetDLQItems)
	admin.Post("/jobs/dlq/requeue", dlqHandler.RequeueDLQItems)
	admin.Post("/jobs/dlq/purge", dlqHandler.PurgeDLQItems)
	admin.Get("/jobs/dlq/:id", dlqHandler.GetJobById)
	admin.Get("/capacity", capacityHandler.GetAllCapacityStatuses)
	admin.Get("/capacity/:id", capacityHandler.GetCapacityForLibrary)

	// Search route (protected with auth and rate limiting)
	s.app.Post("/api/search", middleware.RateLimiterForSearch(), searchHandler.Search) // Search endpoint with rate limiting

	// Library routes - register all manually on protected group to control exact order
	adminMW := middleware.NewAuthMiddleware(s.authService).AdminOnly()
	protected.Get("/libraries", adminMW, libraryHandler.GetLibraryStates)
	// Use /library-stats instead of /libraries/stats due to Fiber routing issue with /:id matching
	protected.Get("/library-stats", adminMW, libraryHandler.GetLibrariesStats)
	protected.Get("/libraries/quarantine", adminMW, libraryHandler.GetQuarantineItems)
	protected.Get("/libraries/jobs", adminMW, libraryHandler.GetProcessingJobs)
	protected.Get("/libraries/:id", adminMW, libraryHandler.GetLibraryState)
	protected.Put("/libraries/:id", adminMW, libraryHandler.UpdateLibrary)
	protected.Get("/libraries/:id/scan", adminMW, libraryHandler.TriggerLibraryScan)
	protected.Get("/libraries/:id/process", adminMW, libraryHandler.TriggerLibraryProcess)
	protected.Get("/libraries/:id/move-ok", adminMW, libraryHandler.TriggerLibraryMoveOK)
	protected.Post("/libraries/quarantine/:id/resolve", adminMW, libraryHandler.ResolveQuarantineItem)
	protected.Post("/libraries/quarantine/:id/requeue", adminMW, libraryHandler.RequeueQuarantineItem)
}

// Start starts the API server
func (s *APIServer) Start() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
	log.Printf("Starting API server on %s", addr)
	return s.app.Listen(addr)
}

// Shutdown gracefully shuts down the server
func (s *APIServer) Shutdown() error {
	return s.app.Shutdown()
}

// Main entry point for the API service
func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Initialize database
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
		log.Fatal("Failed to connect to database:", err)
	}

	// Run migrations
	migrationManager := database.NewMigrationManager(dbManager.GetGormDB(), nil)
	if err := migrationManager.Migrate(); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	// Seed default libraries if none exist
	if err := database.SeedDefaultLibraries(dbManager.GetGormDB()); err != nil {
		log.Fatal("Failed to seed default libraries:", err)
	}

	// Create and start server
	server := NewAPIServer(cfg, dbManager)
	if err := server.Start(); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
