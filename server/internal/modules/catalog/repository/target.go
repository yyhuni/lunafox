package repository

import (
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
)

// TargetRepository handles target database operations.
type TargetRepository struct {
	db *gorm.DB
}

// NewTargetRepository creates a new target repository.
func NewTargetRepository(db *gorm.DB) *TargetRepository {
	return &TargetRepository{db: db}
}

// TargetFilterMapping defines field mapping for target filtering.
var TargetFilterMapping = scope.FilterMapping{
	"name": {Column: "name"},
	"type": {Column: "type"},
}

var targetFilterMappingNormalized = scope.NormalizeFilterMapping(TargetFilterMapping)
