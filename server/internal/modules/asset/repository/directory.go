package repository

import (
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
)

// DirectoryRepository handles directory database operations
type DirectoryRepository struct {
	db *gorm.DB
}

// NewDirectoryRepository creates a new directory repository
func NewDirectoryRepository(db *gorm.DB) *DirectoryRepository {
	return &DirectoryRepository{db: db}
}

// DirectoryFilterMapping defines field mapping for directory filtering
var DirectoryFilterMapping = scope.FilterMapping{
	"url":    {Column: "url"},
	"status": {Column: "status", IsNumeric: true},
}

var directoryFilterMappingNormalized = scope.NormalizeFilterMapping(DirectoryFilterMapping)
