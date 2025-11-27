package database

import (
	"log"
	"time"

	"melodee/internal/models"

	"gorm.io/gorm"
)

// SeedDefaultSettings creates default settings entries if none exist
func SeedDefaultSettings(db *gorm.DB) error {
	// Check if any settings exist
	var count int64
	if err := db.Model(&models.Setting{}).Count(&count).Error; err != nil {
		return err
	}

	// Only seed if no settings exist
	if count > 0 {
		log.Println("Settings already exist, skipping seed")
		return nil
	}

	log.Println("Seeding default settings...")

	defaultSettings := []models.Setting{
		{
			Key:       "processing.scan_workers",
			Value:     "8",
			Comment:   "Number of concurrent workers for library directory scanning (1-32)",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Key:       "processing.scan_buffer_size",
			Value:     "1000",
			Comment:   "Buffer size for scan file channel (100-10000)",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Key:       "processing.scan_max_files",
			Value:     "0",
			Comment:   "Maximum number of files to scan (0 = no limit, useful for troubleshooting)",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Key:       "staging_cron.enabled",
			Value:     "false",
			Comment:   "Enable the staging cron job (true/false)",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Key:       "staging_cron.dry_run",
			Value:     "true",
			Comment:   "Dry run mode for staging cron (true/false)",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Key:       "staging_cron.schedule",
			Value:     "0 */1 * * *",
			Comment:   "Cron schedule for staging job (e.g., '0 */1 * * *' for hourly)",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Key:       "staging_cron.workers",
			Value:     "4",
			Comment:   "Number of worker goroutines for staging job (1-32)",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Key:       "staging_cron.rate_limit",
			Value:     "0",
			Comment:   "Rate limit for file operations (0 = unlimited)",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Key:       "staging_cron.scan_db_data_path",
			Value:     "/tmp/melodee-scans",
			Comment:   "Directory where temporary scan DB files are written (not a library path)",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, setting := range defaultSettings {
		if err := db.Create(&setting).Error; err != nil {
			log.Printf("Error seeding setting %s: %v", setting.Key, err)
			return err
		}
		log.Printf("Seeded setting: %s = %s", setting.Key, setting.Value)
	}

	log.Println("Default settings seeded successfully")
	return nil
}

// SeedDefaultLibraries creates default library entries if none exist
func SeedDefaultLibraries(db *gorm.DB) error {
	// Check if any libraries exist
	var count int64
	if err := db.Model(&models.Library{}).Count(&count).Error; err != nil {
		return err
	}

	// Only seed if no libraries exist
	if count > 0 {
		log.Println("Libraries already exist, skipping seed")
		return nil
	}

	log.Println("Seeding default libraries...")

	defaultLibraries := []models.Library{
		{
			Name:       "Inbound",
			Path:       "/melodee/inbound",
			Type:       "inbound",
			IsLocked:   false,
			CreatedAt:  time.Now(),
			TrackCount: 0,
			AlbumCount: 0,
			Duration:   0,
		},
		{
			Name:       "Staging",
			Path:       "/melodee/staging",
			Type:       "staging",
			IsLocked:   false,
			CreatedAt:  time.Now(),
			TrackCount: 0,
			AlbumCount: 0,
			Duration:   0,
		},
		{
			Name:       "Production",
			Path:       "/melodee/storage",
			Type:       "production",
			IsLocked:   false,
			CreatedAt:  time.Now(),
			TrackCount: 0,
			AlbumCount: 0,
			Duration:   0,
		},
	}

	for _, lib := range defaultLibraries {
		if err := db.Create(&lib).Error; err != nil {
			log.Printf("Error seeding library %s: %v", lib.Name, err)
			return err
		}
		log.Printf("Seeded library: %s (%s)", lib.Name, lib.Path)
	}

	log.Println("Default libraries seeded successfully")
	return nil
}
