package database

import (
	"fmt"

	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"melodee/internal/media"
	"melodee/internal/models"
)

// MigrationManager manages database migrations
type MigrationManager struct {
	db     *gorm.DB
	logger *zerolog.Logger
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(db *gorm.DB, logger *zerolog.Logger) *MigrationManager {
	return &MigrationManager{
		db:     db,
		logger: logger,
	}
}

// Migrate runs database migrations
func (m *MigrationManager) Migrate() error {
	// Create extensions first
	if err := m.createExtensions(); err != nil {
		return fmt.Errorf("failed to create extensions: %w", err)
	}

	// Let GORM handle all table creation from models
	if err := m.migrateTables(); err != nil {
		return fmt.Errorf("failed to migrate tables: %w", err)
	}

	if m.logger != nil {
		m.logger.Info().Msg("Database migrations completed successfully")
	}
	return nil
}

// SeedDefaultLibraries creates default library entries if none exist
func (m *MigrationManager) SeedDefaultLibraries(storagePath, inboundPath, stagingPath string) error {
	// Check if libraries already exist
	var count int64
	if err := m.db.Model(&models.Library{}).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to count libraries: %w", err)
	}

	// Only seed if no libraries exist
	if count > 0 {
		if m.logger != nil {
			m.logger.Info().Int64("count", count).Msg("Libraries already exist, skipping seed")
		}
		return nil
	}

	// Create default libraries
	libraries := []models.Library{
		{
			Name:     "Production Library",
			Path:     storagePath,
			Type:     "production",
			BasePath: storagePath,
			IsLocked: false,
		},
		{
			Name:     "Inbound Library",
			Path:     inboundPath,
			Type:     "inbound",
			BasePath: inboundPath,
			IsLocked: false,
		},
		{
			Name:     "Staging Library",
			Path:     stagingPath,
			Type:     "staging",
			BasePath: stagingPath,
			IsLocked: false,
		},
	}

	for _, lib := range libraries {
		if err := m.db.Create(&lib).Error; err != nil {
			return fmt.Errorf("failed to create library %s: %w", lib.Name, err)
		}
		if m.logger != nil {
			m.logger.Info().Str("name", lib.Name).Str("path", lib.Path).Msg("Created library")
		}
	}

	return nil
}

// migrateTables handles migration of all tables via GORM
func (m *MigrationManager) migrateTables() error {
	// Let GORM create all tables from Go models
	// This ensures schema consistency and avoids manual SQL/GORM conflicts
	if err := m.db.AutoMigrate(
		&models.User{},
		&models.Library{},
		&models.Artist{},
		&models.Album{},
		&models.Song{},
		&models.Playlist{},
		&models.PlaylistSong{},
		&models.UserSong{},
		&models.UserAlbum{},
		&models.UserArtist{},
		&models.UserPin{},
		&models.Bookmark{},
		&models.Player{},
		&models.PlayQueue{},
		&models.SearchHistory{},
		&models.Share{},
		&models.ShareActivity{},
		&models.LibraryScanHistory{},
		&models.Setting{},
		&models.ArtistRelation{},
		&models.RadioStation{},
		&models.Contributor{},
		&models.CapacityStatus{},
		&media.QuarantineRecord{},
	); err != nil {
		return fmt.Errorf("failed to auto-migrate tables: %w", err)
	}

	return nil
}

// createExtensions creates PostgreSQL extensions needed by the application
func (m *MigrationManager) createExtensions() error {
	extensions := []string{"uuid-ossp", "pg_trgm", "btree_gin"}

	for _, ext := range extensions {
		// Quote extension name to handle names with hyphens like "uuid-ossp"
		if err := m.db.Exec(fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS \"%s\"", ext)).Error; err != nil {
			return fmt.Errorf("failed to create extension %s: %w", ext, err)
		}
	}

	return nil
}
