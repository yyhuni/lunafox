package repository

import (
	"gorm.io/gorm"
)

// WordlistRepository handles wordlist database operations
type WordlistRepository struct {
	db *gorm.DB
}

// NewWordlistRepository creates a new wordlist repository
func NewWordlistRepository(db *gorm.DB) *WordlistRepository {
	return &WordlistRepository{db: db}
}
