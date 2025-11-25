package database

import (
	"log"
	"time"

	"melodee/internal/models"

	"gorm.io/gorm"
)

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
			BasePath:   "/melodee",
			CreatedAt:  time.Now(),
			SongCount:  0,
			AlbumCount: 0,
			Duration:   0,
		},
		{
			Name:       "Staging",
			Path:       "/melodee/staging",
			Type:       "staging",
			IsLocked:   false,
			BasePath:   "/melodee",
			CreatedAt:  time.Now(),
			SongCount:  0,
			AlbumCount: 0,
			Duration:   0,
		},
		{
			Name:       "Production",
			Path:       "/melodee/storage",
			Type:       "production",
			IsLocked:   false,
			BasePath:   "/melodee",
			CreatedAt:  time.Now(),
			SongCount:  0,
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
