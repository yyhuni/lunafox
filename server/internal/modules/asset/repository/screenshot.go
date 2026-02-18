package repository

import (
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
)

// ScreenshotRepository handles screenshot database operations
type ScreenshotRepository struct {
	db *gorm.DB
}

// NewScreenshotRepository creates a new screenshot repository
func NewScreenshotRepository(db *gorm.DB) *ScreenshotRepository {
	return &ScreenshotRepository{db: db}
}

var ScreenshotFilterMapping = scope.FilterMapping{
	"url": {Column: "url"},
}

var screenshotFilterMappingNormalized = scope.NormalizeFilterMapping(ScreenshotFilterMapping)
