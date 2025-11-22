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
	"melodee/internal/media"
	"melodee/open_subsonic/handlers"
	"melodee/open_subsonic/middleware"
	"melodee/open_subsonic/services"
)

// OpenSubsonicServer represents the OpenSubsonic API server
type OpenSubsonicServer struct {
	app          *fiber.App
	cfg          *config.AppConfig
	db           *gorm.DB
}

// NewOpenSubsonicServer creates a new OpenSubsonic server
func NewOpenSubsonicServer(cfg *config.AppConfig, dbManager *database.DatabaseManager) *OpenSubsonicServer {
	db := dbManager.GetGormDB()

	server := &OpenSubsonicServer{
		cfg: cfg,
		db:  db,
	}

	// Initialize Fiber app
	server.app = fiber.New(fiber.Config{
		AppName:      "Melodee OpenSubsonic API Server",
		ServerHeader: "Melodee",
	})

	// Add middleware
	server.app.Use(recover.New())
	server.app.Use(logger.New())
	server.app.Use(cors.New())

	// Setup routes
	server.setupRoutes()

	return server
}

// setupRoutes configures the OpenSubsonic API routes
func (s *OpenSubsonicServer) setupRoutes() {
	// Create authentication middleware
	authMiddleware := middleware.NewOpenSubsonicAuthMiddleware(s.db, s.cfg.JWT.Secret)

	// Create handlers
	browsingHandler := handlers.NewBrowsingHandler(s.db)
	mediaHandler := handlers.NewMediaHandler(s.db, nil) // Using placeholder config
	searchHandler := handlers.NewSearchHandler(s.db)
	playlistHandler := handlers.NewPlaylistHandler(s.db)
	userHandler := handlers.NewUserHandler(s.db)
	systemHandler := handlers.NewSystemHandler(s.db)

	// Define the API routes under /rest/ prefix
	rest := s.app.Group("/rest")

	// Browsing endpoints
	rest.Get("/getMusicFolders.view", authMiddleware.Authenticate, browsingHandler.GetMusicFolders)
	rest.Get("/getIndexes.view", authMiddleware.Authenticate, browsingHandler.GetIndexes)
	rest.Get("/getArtists.view", authMiddleware.Authenticate, browsingHandler.GetArtists)
	rest.Get("/getArtist.view", authMiddleware.Authenticate, browsingHandler.GetArtist)
	rest.Get("/getAlbumInfo.view", authMiddleware.Authenticate, browsingHandler.GetAlbumInfo)
	rest.Get("/getMusicDirectory.view", authMiddleware.Authenticate, browsingHandler.GetMusicDirectory)
	rest.Get("/getAlbum.view", authMiddleware.Authenticate, browsingHandler.GetAlbum)
	rest.Get("/getSong.view", authMiddleware.Authenticate, browsingHandler.GetSong)
	rest.Get("/getGenres.view", authMiddleware.Authenticate, browsingHandler.GetGenres)

	// Media retrieval endpoints
	rest.Get("/stream.view", authMiddleware.Authenticate, mediaHandler.Stream)
	rest.Get("/download.view", authMiddleware.Authenticate, mediaHandler.Download)
	rest.Get("/getCoverArt.view", authMiddleware.Authenticate, mediaHandler.GetCoverArt)
	rest.Get("/getAvatar.view", authMiddleware.Authenticate, mediaHandler.GetAvatar)

	// Searching endpoints
	rest.Get("/search.view", authMiddleware.Authenticate, searchHandler.Search)
	rest.Get("/search2.view", authMiddleware.Authenticate, searchHandler.Search2)
	rest.Get("/search3.view", authMiddleware.Authenticate, searchHandler.Search3)

	// Playlist endpoints
	rest.Get("/getPlaylists.view", authMiddleware.Authenticate, playlistHandler.GetPlaylists)
	rest.Get("/getPlaylist.view", authMiddleware.Authenticate, playlistHandler.GetPlaylist)
	rest.Get("/createPlaylist.view", authMiddleware.Authenticate, playlistHandler.CreatePlaylist)
	rest.Get("/updatePlaylist.view", authMiddleware.Authenticate, playlistHandler.UpdatePlaylist)
	rest.Get("/deletePlaylist.view", authMiddleware.Authenticate, playlistHandler.DeletePlaylist)

	// User management endpoints
	rest.Get("/getUser.view", authMiddleware.Authenticate, userHandler.GetUser)
	rest.Get("/getUsers.view", authMiddleware.Authenticate, userHandler.GetUsers)
	rest.Get("/createUser.view", authMiddleware.Authenticate, userHandler.CreateUser)
	rest.Get("/updateUser.view", authMiddleware.Authenticate, userHandler.UpdateUser)
	rest.Get("/deleteUser.view", authMiddleware.Authenticate, userHandler.DeleteUser)

	// System endpoints
	rest.Get("/ping.view", systemHandler.Ping)
	rest.Get("/getLicense.view", systemHandler.GetLicense)
}

// Start starts the OpenSubsonic server
func (s *OpenSubsonicServer) Start() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
	log.Printf("Starting OpenSubsonic API server on %s", addr)
	return s.app.Listen(addr)
}

// Shutdown gracefully shuts down the server
func (s *OpenSubsonicServer) Shutdown() error {
	return s.app.Shutdown()
}

// Main entry point for the OpenSubsonic service
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
	server := NewOpenSubsonicServer(cfg, dbManager)
	if err := server.Start(); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}