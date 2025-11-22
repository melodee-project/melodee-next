package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"melodee/internal/config"
	"melodee/internal/database"
	"melodee/internal/directory"
	"melodee/internal/media"
)

// WorkerServer handles background job processing
type WorkerServer struct {
	srv        *asynq.Server
	db         *gorm.DB
	config     *config.AppConfig
	mediaSvc   *media.MediaService
	directorySvc *directory.DirectoryCodeGenerator
	pathResolver *directory.PathTemplateResolver
}

// NewWorkerServer creates a new worker server
func NewWorkerServer() (*WorkerServer, error) {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
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
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Initialize directory code generator
	directoryConfig := directory.DefaultDirectoryCodeConfig()
	directorySvc := directory.NewDirectoryCodeGenerator(directoryConfig, dbManager.GetGormDB())

	// Initialize path template resolver
	pathConfig := directory.DefaultPathTemplateConfig()
	pathResolver := directory.NewPathTemplateResolver(pathConfig)

	// Initialize media service
	mediaSvc := media.NewMediaService(dbManager.GetGormDB(), directorySvc, pathResolver)

	// Initialize Asynq server with Redis connection
	redisAddr := fmt.Sprintf("%s:%d", cfg.Redis.Address, cfg.Redis.Port)
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			// Specify queues and their priorities
			Queues: map[string]int{
				"critical":  6, // stream-serving support tasks
				"default":   3, // library scans, metadata write-backs
				"bulk":      1, // large backfills
				"maintenance": 2, // partition/index management
			},
			// Set concurrency for each worker
			Concurrency: 10,
			// Enable periodic task scheduler
			PeriodicTaskConfig: &asynq.PeriodicTaskConfig{
				CleanupInterval: 24 * time.Hour,
			},
		},
	)

	// Register task handlers
	srv.HandleFunc(media.TypeLibraryScan, media.HandleLibraryScan)
	srv.HandleFunc(media.TypeLibraryProcess, media.HandleLibraryProcess)
	srv.HandleFunc(media.TypeLibraryMoveOK, media.HandleLibraryMoveOK)
	srv.HandleFunc(media.TypeDirectoryRecalculate, media.HandleDirectoryRecalculate)
	srv.HandleFunc(media.TypeMetadataWriteback, media.HandleMetadataWriteback)
	srv.HandleFunc(media.TypeMetadataEnhance, media.HandleMetadataEnhance)

	return &WorkerServer{
		srv:          srv,
		db:           dbManager.GetGormDB(),
		config:       cfg,
		mediaSvc:     mediaSvc,
		directorySvc: directorySvc,
		pathResolver: pathResolver,
	}, nil
}

// Start starts the worker server
func (w *WorkerServer) Start() error {
	log.Println("Starting worker server...")

	// Start the server
	if err := w.srv.Run(); err != nil {
		return fmt.Errorf("failed to start worker server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the worker server
func (w *WorkerServer) Shutdown() {
	log.Println("Shutting down worker server...")
	w.srv.Shutdown()
}

// Main entry point for the worker service
func main() {
	worker, err := NewWorkerServer()
	if err != nil {
		log.Fatal("Failed to create worker server:", err)
	}

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	// Start the worker in a goroutine
	go func() {
		if err := worker.Start(); err != nil {
			log.Fatal("Worker server error:", err)
		}
	}()

	// Wait for shutdown signal
	<-sigCh
	log.Println("Received shutdown signal")
	worker.Shutdown()
}