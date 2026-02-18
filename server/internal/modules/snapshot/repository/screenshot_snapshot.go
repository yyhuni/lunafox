package repository

import (
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
)

// ScreenshotSnapshotRepository handles screenshot snapshot database operations
type ScreenshotSnapshotRepository struct {
	db *gorm.DB
}

// NewScreenshotSnapshotRepository creates a new screenshot snapshot repository
func NewScreenshotSnapshotRepository(db *gorm.DB) *ScreenshotSnapshotRepository {
	return &ScreenshotSnapshotRepository{db: db}
}

// ScreenshotSnapshotFilterMapping defines field mapping for screenshot snapshot filtering
var ScreenshotSnapshotFilterMapping = scope.FilterMapping{
	"url":    {Column: "url"},
	"status": {Column: "status_code", IsNumeric: true},
}

var screenshotSnapshotFilterMappingNormalized = scope.NormalizeFilterMapping(ScreenshotSnapshotFilterMapping)
