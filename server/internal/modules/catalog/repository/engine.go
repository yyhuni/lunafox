package repository

import (
	"gorm.io/gorm"
)

// EngineRepository handles scan engine database operations
type EngineRepository struct {
	db *gorm.DB
}

// NewEngineRepository creates a new engine repository
func NewEngineRepository(db *gorm.DB) *EngineRepository {
	return &EngineRepository{db: db}
}
