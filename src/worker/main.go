package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"melodee/internal/config"
	"melodee/internal/database"
	"melodee/internal/directory"
	"melodee/internal/logging"
	"melodee/internal/media"
)

// WorkerServer handles background job processing
type WorkerServer struct {
	srv          *asynq.Server
	db           *gorm.DB
	config       *config.AppConfig
	mediaSvc     *media.MediaService
	directorySvc *directory.DirectoryCodeGenerator
	pathResolver *directory.PathTemplateResolver
}

// NewWorkerServer creates a new worker server
func NewWorkerServer() (*WorkerServer, *asynq.ServeMux, error) {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize database first
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
		nil, // logger - will be initialized below
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Initialize logging with database storage
	logStorage := logging.NewLogStorage(dbManager.GetGormDB())
	logging.InitGlobalLogger(logging.InfoLevel, "json", logStorage)
	logging.Info("Worker logging initialized with database storage")
	logging.Infof("Worker starting up - redis: %s, workers: %d, buffer: %d",
		cfg.Redis.Address, cfg.Processing.ScanWorkers, cfg.Processing.ScanBufferSize)

	// Initialize directory code generator
	directoryConfig := directory.DefaultDirectoryCodeConfig()
	directorySvc := directory.NewDirectoryCodeGenerator(directoryConfig, dbManager.GetGormDB())

	// Initialize path template resolver
	pathConfig := directory.DefaultPathTemplateConfig()
	pathResolver := directory.NewPathTemplateResolver(pathConfig)

	// Initialize quarantine and media services
	quarantineSvc := media.NewDefaultQuarantineService(dbManager.GetGormDB())
	mediaSvc := media.NewMediaService(dbManager.GetGormDB(), directorySvc, pathResolver, quarantineSvc)

	// Initialize task handler with dependencies and configuration
	taskHandler := media.NewTaskHandler(
		dbManager.GetGormDB(),
		directorySvc,
		pathResolver,
		quarantineSvc,
		cfg.Processing.ScanWorkers,
		cfg.Processing.ScanBufferSize,
	)

	// Initialize Asynq server with Redis connection
	redisAddr := cfg.Redis.Address
	logging.Infof("Connecting to Redis at %s", redisAddr)

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			Queues: map[string]int{
				"critical":    6,
				"default":     3,
				"bulk":        1,
				"maintenance": 2,
			},
			Concurrency: 10,
		},
	)

	logging.Info("Asynq server configured with concurrency=10, queues: critical:6, default:3, bulk:1, maintenance:2")

	// Register task handlers using a ServeMux with handler that has dependencies
	mux := asynq.NewServeMux()
	mux.HandleFunc(media.TypeLibraryScan, taskHandler.HandleLibraryScan)
	mux.HandleFunc(media.TypeLibraryProcess, media.HandleLibraryProcess)
	mux.HandleFunc(media.TypeLibraryMoveOK, media.HandleLibraryMoveOK)
	mux.HandleFunc(media.TypeDirectoryRecalculate, media.HandleDirectoryRecalculate)
	mux.HandleFunc(media.TypeMetadataWriteback, media.HandleMetadataWriteback)
	mux.HandleFunc(media.TypeMetadataEnhance, media.HandleMetadataEnhance)

	logging.Infof("Registered 6 task handlers: %s, %s, %s, %s, %s, %s",
		media.TypeLibraryScan, media.TypeLibraryProcess, media.TypeLibraryMoveOK,
		media.TypeDirectoryRecalculate, media.TypeMetadataWriteback, media.TypeMetadataEnhance)

	return &WorkerServer{
		srv:          srv,
		db:           dbManager.GetGormDB(),
		config:       cfg,
		mediaSvc:     mediaSvc,
		directorySvc: directorySvc,
		pathResolver: pathResolver,
	}, mux, nil
}

// Start starts the worker server
func (w *WorkerServer) Start(mux *asynq.ServeMux) error {
	logging.Info("Starting Asynq worker server...")

	// Start the server
	if err := w.srv.Run(mux); err != nil {
		logging.Errorf("Failed to start worker server: %v", err)
		return fmt.Errorf("failed to start worker server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the worker server
func (w *WorkerServer) Shutdown() {
	logging.Info("Shutting down worker server...")
	w.srv.Shutdown()
	logging.Info("Worker server shut down complete")
}

// Main entry point for the worker service
func main() {
	fmt.Println("===== Melodee Worker Starting =====")

	worker, mux, err := NewWorkerServer()
	if err != nil {
		logging.Errorf("Failed to create worker server: %v", err)
		os.Exit(1)
	}

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	// Start the worker in a goroutine
	go func() {
		if err := worker.Start(mux); err != nil {
			logging.Errorf("Worker server error: %v", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	sig := <-sigCh
	logging.Infof("Received shutdown signal: %s", sig.String())
	worker.Shutdown()
	fmt.Println("===== Melodee Worker Stopped =====")
}
