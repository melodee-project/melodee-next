package main

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"gorm.io/gorm"

	"melodee/internal/config"
	"melodee/internal/database"
	"melodee/internal/handlers"
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

	// Setup routes
	server.setupRoutes()

	// Register custom metrics
	handlers.RegisterCustomMetrics()

	return server
}

// setupRoutes configures the API routes
func (s *APIServer) setupRoutes() {
	// Create handlers
	authHandler := handlers.NewAuthHandler(s.authService)
	userHandler := handlers.NewUserHandler(s.repo, s.authService)
	playlistHandler := handlers.NewPlaylistHandler(s.repo)
	healthHandler := handlers.NewHealthHandler(s.dbManager) // Pass the dbManager

	// Health check route
	s.app.Get("/healthz", healthHandler.HealthCheck)

	// Metrics route
	metricsHandler := handlers.NewMetricsHandler()
	s.app.Get("/metrics", metricsHandler.Metrics())

	// Auth routes
	auth := s.app.Group("/api/auth")
	auth.Post("/login", authHandler.Login)
	auth.Post("/refresh", authHandler.Refresh)
	auth.Post("/request-reset", authHandler.RequestReset)
	auth.Post("/reset", authHandler.ResetPassword)

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

	// Create and start server
	server := NewAPIServer(cfg, dbManager)
	if err := server.Start(); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}