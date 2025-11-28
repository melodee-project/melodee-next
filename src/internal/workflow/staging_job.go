package workflow

import (
	"context"
	"fmt"
	"os"
	"time"

	"melodee/internal/config"
	"melodee/internal/logging"
	"melodee/internal/models"
	"melodee/internal/processor"
	"melodee/internal/scanner"

	"gorm.io/gorm"
)

// StagingJobConfig holds all knobs for one staging job run.
type StagingJobConfig struct {
	Workers        int
	RateLimit      int
	DryRun         bool
	ScanDBDataPath string
}

// StagingJobResult contains the results of a staging job run
type StagingJobResult struct {
	InboundPath   string
	StagingPath   string
	ScanDBPath    string
	AlbumsTotal   int
	AlbumsSuccess int
	AlbumsFailed  int
	Duration      time.Duration
	ProcessedAt   time.Time
	DryRun        bool
	Error         error
}

// StagingJobService handles the staging workflow
type StagingJobService struct {
	db     *gorm.DB
	logger Logger
}

// Logger interface to allow different logging implementations
type Logger interface {
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
}

// LoggerAdapter adapts the global logger to the workflow logger interface
type LoggerAdapter struct {
	logger *logging.Logger
}

// NewLoggerAdapter creates a new logger adapter
func NewLoggerAdapter(logger *logging.Logger) *LoggerAdapter {
	return &LoggerAdapter{
		logger: logger,
	}
}

func (la *LoggerAdapter) Info(args ...interface{}) {
	la.logger.Info(fmt.Sprint(args...))
}

func (la *LoggerAdapter) Infof(format string, args ...interface{}) {
	la.logger.Info(fmt.Sprintf(format, args...))
}

func (la *LoggerAdapter) Warn(args ...interface{}) {
	la.logger.Warn(fmt.Sprint(args...))
}

func (la *LoggerAdapter) Warnf(format string, args ...interface{}) {
	la.logger.Warn(fmt.Sprintf(format, args...))
}

func (la *LoggerAdapter) Error(args ...interface{}) {
	la.logger.Error(fmt.Sprint(args...))
}

func (la *LoggerAdapter) Errorf(format string, args ...interface{}) {
	la.logger.Error(fmt.Sprintf(format, args...))
}

func (la *LoggerAdapter) Debug(args ...interface{}) {
	la.logger.Debug(fmt.Sprint(args...))
}

func (la *LoggerAdapter) Debugf(format string, args ...interface{}) {
	la.logger.Debug(fmt.Sprintf(format, args...))
}

// NewStagingJobService creates a new staging job service
func NewStagingJobService(db *gorm.DB, logger Logger) *StagingJobService {
	return &StagingJobService{
		db:     db,
		logger: logger,
	}
}

// RunStagingJobCycle runs a full staging cycle: scan inbound → process to staging → write staging_items.
// It is the canonical implementation used by cron and any other callers.
func (s *StagingJobService) RunStagingJobCycle(ctx context.Context, cfg StagingJobConfig) (*StagingJobResult, error) {
	startTime := time.Now()

	result := &StagingJobResult{
		ProcessedAt: startTime,
		DryRun:      cfg.DryRun,
	}

	s.logger.Infof("Starting staging job cycle with DryRun=%t", cfg.DryRun)

	// 0) Resolve current runtime configuration and libraries
	inboundLibrary, err := s.resolveLibraryByType("inbound")
	if err != nil {
		err := fmt.Errorf("failed to resolve inbound library: %w", err)
		s.logger.Errorf("Staging job failed: %v", err)
		return &StagingJobResult{Error: err}, err
	}

	stagingLibrary, err := s.resolveLibraryByType("staging")
	if err != nil {
		err := fmt.Errorf("failed to resolve staging library: %w", err)
		s.logger.Errorf("Staging job failed: %v", err)
		return &StagingJobResult{Error: err}, err
	}

	inboundPath := inboundLibrary.Path
	stagingPath := stagingLibrary.Path
	scanOutputDir := cfg.ScanDBDataPath

	result.InboundPath = inboundPath
	result.StagingPath = stagingPath

	s.logger.Infof("Resolved libraries - Inbound: %s, Staging: %s", inboundPath, stagingPath)

	// Check if paths exist
	if _, err := os.Stat(inboundPath); os.IsNotExist(err) {
		err := fmt.Errorf("inbound path does not exist: %s", inboundPath)
		s.logger.Errorf("Staging job failed: %v", err)
		return &StagingJobResult{Error: err}, err
	}

	if _, err := os.Stat(stagingPath); os.IsNotExist(err) {
		err := fmt.Errorf("staging path does not exist: %s", stagingPath)
		s.logger.Errorf("Staging job failed: %v", err)
		return &StagingJobResult{Error: err}, err
	}

	// 1) Create scan DB (like scan-inbound)
	s.logger.Infof("Creating scan database in %s...", scanOutputDir)
	scanDB, err := scanner.NewScanDB(scanOutputDir)
	if err != nil {
		err := fmt.Errorf("failed to create scan database: %w", err)
		s.logger.Errorf("Staging job failed: %v", err)
		return &StagingJobResult{Error: err}, err
	}
	defer scanDB.Close()

	result.ScanDBPath = scanDB.GetPath()
	s.logger.Infof("Created scan database: %s", scanDB.GetPath())

	// Create file scanner
	fileScanner := scanner.NewFileScanner(scanDB, cfg.Workers)

	// Scan the directory
	s.logger.Infof("Scanning inbound directory %s with %d workers...", inboundPath, cfg.Workers)
	if err := fileScanner.ScanDirectory(inboundPath); err != nil {
		err := fmt.Errorf("failed to scan directory: %w", err)
		s.logger.Errorf("Staging job failed: %v", err)
		return &StagingJobResult{Error: err}, err
	}

	// Compute album grouping
	s.logger.Info("Computing album grouping...")
	if err := scanDB.ComputeAlbumGrouping(); err != nil {
		err := fmt.Errorf("failed to compute album grouping: %w", err)
		s.logger.Errorf("Staging job failed: %v", err)
		return &StagingJobResult{Error: err}, err
	}

	// Get album groups for stats
	albumGroups, err := scanDB.GetAlbumGroups()
	if err != nil {
		err := fmt.Errorf("failed to get album groups: %w", err)
		s.logger.Errorf("Staging job failed: %v", err)
		return &StagingJobResult{Error: err}, err
	}
	result.AlbumsTotal = len(albumGroups)

	// Get scan statistics
	stats, err := scanDB.GetStats()
	if err != nil {
		err := fmt.Errorf("failed to get scan statistics: %w", err)
		s.logger.Errorf("Staging job failed: %v", err)
		return &StagingJobResult{Error: err}, err
	}

	s.logger.Infof("Scan completed - Total files: %d, Valid files: %d, Albums found: %d",
		stats.TotalFiles, stats.ValidFiles, stats.AlbumsFound)

	// 2) Process albums to staging (like process-scan)
	procConfig := &processor.ProcessorConfig{
		StagingRoot: stagingPath,
		Workers:     cfg.Workers,
		RateLimit:   cfg.RateLimit,
		DryRun:      cfg.DryRun,
	}

	proc := processor.NewProcessor(procConfig, scanDB)

	s.logger.Infof("Processing albums to staging (%s)... Workers: %d, RateLimit: %d",
		stagingPath, cfg.Workers, cfg.RateLimit)
	if cfg.DryRun {
		s.logger.Info("*** DRY RUN MODE - No files will be moved ***")
	}

	// Process all albums
	results, err := proc.ProcessAllAlbums()
	if err != nil {
		err := fmt.Errorf("failed to process albums: %w", err)
		s.logger.Errorf("Staging job failed: %v", err)
		return &StagingJobResult{Error: err}, err
	}

	// Calculate stats
	var successCount, failedCount int
	for _, res := range results {
		if res.Success {
			successCount++
		} else {
			failedCount++
		}
	}
	result.AlbumsSuccess = successCount
	result.AlbumsFailed = failedCount

	s.logger.Infof("Processing completed - Success: %d, Failed: %d", successCount, failedCount)

	// 3) Persist to Postgres (if not dry run and DB is available)
	if !cfg.DryRun && s.db != nil {
		s.logger.Info("Saving staging items to database...")
		savedCount := 0
		stagingRepo := processor.NewStagingRepository(s.db)

		for _, resultRes := range results {
			if resultRes.Success {
				// Read metadata file
				metadata, err := processor.ReadAlbumMetadata(resultRes.MetadataFile)
				if err != nil {
					s.logger.Warnf("Could not read metadata for %s: %v", resultRes.StagingPath, err)
					continue
				}

				// Create staging item
				if err := stagingRepo.CreateStagingItemFromResult(resultRes, metadata); err != nil {
					s.logger.Warnf("Could not save staging item for %s: %v", resultRes.StagingPath, err)
					continue
				}
				savedCount++
			}
		}
		s.logger.Infof("Saved %d staging items to database", savedCount)
	}

	// 4) Return successful result with statistics
	result.Duration = time.Since(startTime)

	// Log the summary
	s.logger.Infof("Staging job completed - Inbound: %s, Staging: %s, AlbumsTotal: %d, Success: %d, Failed: %d, Duration: %v, DryRun: %t",
		result.InboundPath, result.StagingPath, result.AlbumsTotal, result.AlbumsSuccess, result.AlbumsFailed,
		result.Duration, result.DryRun)

	return result, nil
}

// resolveLibraryByType finds a library with a specific type
func (s *StagingJobService) resolveLibraryByType(libraryType string) (*models.Library, error) {
	var library models.Library
	result := s.db.Where("type = ?", libraryType).First(&library)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no %s library found", libraryType)
		}
		return nil, result.Error
	}

	var count int64
	s.db.Model(&models.Library{}).Where("type = ?", libraryType).Count(&count)
	if count > 1 {
		return nil, fmt.Errorf("multiple %s libraries found, only one is allowed", libraryType)
	}

	return &library, nil
}

// RunStagingJobCycleWithConfig resolves configuration from app config and runs the staging job
func (s *StagingJobService) RunStagingJobCycleWithConfig(ctx context.Context, appConfig *config.AppConfig) (*StagingJobResult, error) {
	jobConfig := &StagingJobConfig{
		Workers:        appConfig.StagingScan.Workers,
		RateLimit:      appConfig.StagingScan.RateLimit,
		DryRun:         appConfig.StagingScan.DryRun,
		ScanDBDataPath: appConfig.StagingScan.ScanDBDataPath,
	}

	return s.RunStagingJobCycle(ctx, *jobConfig)
}
