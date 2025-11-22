package services

import (
	"gorm.io/gorm"
)

// Repository handles database operations for models
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new repository instance
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		db: db,
	}
}