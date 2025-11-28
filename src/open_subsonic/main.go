package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"gorm.io/gorm"

	"melodee/internal/config"
	"melodee/internal/database"
	"melodee/internal/media"
	internal_middleware "melodee/internal/middleware"
	"melodee/open_subsonic/handlers"
	opensubsonic_middleware "melodee/open_subsonic/middleware"
	// "melodee/open_subsonic/services"
)

// OpenSubsonicServer represents the OpenSubsonic API server
type OpenSubsonicServer struct {
	app *fiber.App
	cfg *config.AppConfig
	db  *gorm.DB
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

	// Request metrics middleware
	server.app.Use(internal_middleware.MetricsMiddleware())

	server.app.Use(internal_middleware.RateLimiterForPublicAPI())

	// Middleware to handle optional .view suffix for OpenSubsonic compatibility
	server.app.Use(func(c *fiber.Ctx) error {
		path := c.Path()
		if strings.HasSuffix(path, ".view") {
			c.Path(strings.TrimSuffix(path, ".view"))
		}
		return c.Next()
	})

	// Setup routes
	server.setupRoutes()

	return server
}

// setupRoutes configures the OpenSubsonic API routes
func (s *OpenSubsonicServer) setupRoutes() {
	// Create authentication middleware
	authMiddleware := opensubsonic_middleware.NewOpenSubsonicAuthMiddleware(s.db, s.cfg.JWT.Secret)

	// Create media processing components
	ffmpegProcessor := media.NewFFmpegProcessor(media.DefaultFFmpegConfig())                                                                                    // Using default config
	transcodeService := media.NewTranscodeService(ffmpegProcessor, s.cfg.Processing.TranscodeCache.CacheDir, s.cfg.Processing.TranscodeCache.MaxSize*1024*1024) // Convert MB to bytes

	// Create handlers
	browsingHandler := handlers.NewBrowsingHandler(s.db)
	mediaHandler := handlers.NewMediaHandler(s.db, nil, transcodeService) // Pass the transcode service
	searchHandler := handlers.NewSearchHandler(s.db)
	playlistHandler := handlers.NewPlaylistHandler(s.db)
	userHandler := handlers.NewUserHandler(s.db)
	systemHandler := handlers.NewSystemHandler(s.db)
	bookmarkHandler := handlers.NewBookmarkHandler(s.db)
	playQueueHandler := handlers.NewPlayQueueHandler(s.db)

	// Define the API routes under /rest/ prefix
	rest := s.app.Group("/rest")

	// Browsing endpoints
	rest.Get("/getMusicFolders", authMiddleware.Authenticate, browsingHandler.GetMusicFolders)
	rest.Get("/getIndexes", authMiddleware.Authenticate, browsingHandler.GetIndexes)
	rest.Get("/getArtists", authMiddleware.Authenticate, browsingHandler.GetArtists)
	rest.Get("/getArtist", authMiddleware.Authenticate, browsingHandler.GetArtist)
	rest.Get("/getAlbumInfo", authMiddleware.Authenticate, browsingHandler.GetAlbumInfo)
	rest.Get("/getAlbumInfo2", authMiddleware.Authenticate, browsingHandler.GetAlbumInfo2)
	rest.Get("/getArtistInfo", authMiddleware.Authenticate, browsingHandler.GetArtistInfo)
	rest.Get("/getArtistInfo2", authMiddleware.Authenticate, browsingHandler.GetArtistInfo2)
	rest.Get("/getMusicDirectory", authMiddleware.Authenticate, browsingHandler.GetMusicDirectory)
	rest.Get("/getAlbum", authMiddleware.Authenticate, browsingHandler.GetAlbum)
	rest.Get("/getSong", authMiddleware.Authenticate, browsingHandler.GetSong)
	rest.Get("/getGenres", authMiddleware.Authenticate, browsingHandler.GetGenres)
	rest.Get("/getLyrics", authMiddleware.Authenticate, browsingHandler.GetLyrics)
	rest.Get("/getLyricsBySongId", authMiddleware.Authenticate, browsingHandler.GetLyricsBySongId)

	// Lists endpoints (Phase 1)
	rest.Get("/getAlbumList", authMiddleware.Authenticate, browsingHandler.GetAlbumList)
	rest.Get("/getAlbumList2", authMiddleware.Authenticate, browsingHandler.GetAlbumList2)
	rest.Get("/getRandomSongs", authMiddleware.Authenticate, browsingHandler.GetRandomSongs)
	rest.Get("/getSongsByGenre", authMiddleware.Authenticate, browsingHandler.GetSongsByGenre)
	rest.Get("/getNowPlaying", authMiddleware.Authenticate, browsingHandler.GetNowPlaying)
	rest.Get("/getTopSongs", authMiddleware.Authenticate, browsingHandler.GetTopSongs)
	rest.Get("/getSimilarSongs", authMiddleware.Authenticate, browsingHandler.GetSimilarSongs)
	rest.Get("/getSimilarSongs2", authMiddleware.Authenticate, browsingHandler.GetSimilarSongs2)
	rest.Get("/getStarred", authMiddleware.Authenticate, userHandler.GetStarred)
	rest.Get("/getStarred2", authMiddleware.Authenticate, userHandler.GetStarred2)

	// User Interaction endpoints (Phase 2)
	rest.Post("/star", authMiddleware.Authenticate, userHandler.Star)
	rest.Post("/unstar", authMiddleware.Authenticate, userHandler.Unstar)
	rest.Post("/setRating", authMiddleware.Authenticate, userHandler.SetRating)
	rest.Post("/scrobble", authMiddleware.Authenticate, userHandler.Scrobble)

	// Media retrieval endpoints
	rest.Get("/stream", authMiddleware.Authenticate, mediaHandler.Stream)
	rest.Get("/download", authMiddleware.Authenticate, mediaHandler.Download)
	rest.Get("/getCoverArt", authMiddleware.Authenticate, mediaHandler.GetCoverArt)
	rest.Get("/getAvatar", authMiddleware.Authenticate, mediaHandler.GetAvatar)

	// Searching endpoints
	rest.Get("/search", authMiddleware.Authenticate, searchHandler.Search)
	rest.Get("/search2", authMiddleware.Authenticate, searchHandler.Search2)
	rest.Get("/search3", authMiddleware.Authenticate, searchHandler.Search3)

	// Playlist endpoints
	rest.Get("/getPlaylists", authMiddleware.Authenticate, playlistHandler.GetPlaylists)
	rest.Get("/getPlaylist", authMiddleware.Authenticate, playlistHandler.GetPlaylist)
	rest.Get("/createPlaylist", authMiddleware.Authenticate, playlistHandler.CreatePlaylist)
	rest.Get("/updatePlaylist", authMiddleware.Authenticate, playlistHandler.UpdatePlaylist)
	rest.Get("/deletePlaylist", authMiddleware.Authenticate, playlistHandler.DeletePlaylist)

	// User management endpoints
	rest.Get("/getUser", authMiddleware.Authenticate, userHandler.GetUser)
	rest.Get("/getUsers", authMiddleware.Authenticate, userHandler.GetUsers)
	rest.Get("/createUser", authMiddleware.Authenticate, userHandler.CreateUser)
	rest.Get("/updateUser", authMiddleware.Authenticate, userHandler.UpdateUser)
	rest.Get("/deleteUser", authMiddleware.Authenticate, userHandler.DeleteUser)
	rest.Get("/changePassword", authMiddleware.Authenticate, userHandler.ChangePassword)
	rest.Get("/tokenInfo", authMiddleware.Authenticate, userHandler.TokenInfo)

	// Bookmarks
	rest.Get("/createBookmark", authMiddleware.Authenticate, bookmarkHandler.CreateBookmark)
	rest.Get("/deleteBookmark", authMiddleware.Authenticate, bookmarkHandler.DeleteBookmark)
	rest.Get("/getBookmarks", authMiddleware.Authenticate, bookmarkHandler.GetBookmarks)

	// PlayQueue
	rest.Get("/getPlayQueue", authMiddleware.Authenticate, playQueueHandler.GetPlayQueue)
	rest.Get("/savePlayQueue", authMiddleware.Authenticate, playQueueHandler.SavePlayQueue)
	rest.Get("/getPlayQueueByIndex", authMiddleware.Authenticate, playQueueHandler.GetPlayQueueByIndex)
	rest.Get("/savePlayQueueByIndex", authMiddleware.Authenticate, playQueueHandler.SavePlayQueueByIndex)

	// System endpoints
	rest.Get("/ping", systemHandler.Ping)
	rest.Get("/getLicense", systemHandler.GetLicense)
	rest.Get("/getOpenSubsonicExtensions", systemHandler.GetOpenSubsonicExtensions)
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

	// Migrations handled via init-scripts/001_schema.sql

	// Create and start server
	server := NewOpenSubsonicServer(cfg, dbManager)
	if err := server.Start(); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
