package repository

import (
	"gorm.io/gorm"
)

// ScanLogRepository handles scan log database operations
type ScanLogRepository struct {
	db *gorm.DB
}

// NewScanLogRepository creates a new scan log repository
func NewScanLogRepository(db *gorm.DB) *ScanLogRepository {
	return &ScanLogRepository{db: db}
}
