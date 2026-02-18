package repository

import (
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
)

// DirectorySnapshotRepository handles directory snapshot database operations
type DirectorySnapshotRepository struct {
	db *gorm.DB
}

// NewDirectorySnapshotRepository creates a new directory snapshot repository
func NewDirectorySnapshotRepository(db *gorm.DB) *DirectorySnapshotRepository {
	return &DirectorySnapshotRepository{db: db}
}

// DirectorySnapshotFilterMapping defines field mapping for directory snapshot filtering
var DirectorySnapshotFilterMapping = scope.FilterMapping{
	"url":         {Column: "url"},
	"status":      {Column: "status", IsNumeric: true},
	"contentType": {Column: "content_type"},
}

var directorySnapshotFilterMappingNormalized = scope.NormalizeFilterMapping(DirectorySnapshotFilterMapping)
