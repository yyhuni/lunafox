package repository

import "gorm.io/gorm"

// ScanWorkflowRepository persists management-plane scan workflow aggregates.
type ScanWorkflowRepository struct{ db *gorm.DB }

func NewScanWorkflowRepository(db *gorm.DB) *ScanWorkflowRepository {
	return &ScanWorkflowRepository{db: db}
}
