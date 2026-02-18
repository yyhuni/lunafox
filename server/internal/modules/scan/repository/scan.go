package repository

import (
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
)

// ScanRepository handles scan database operations.
type ScanRepository struct {
	db *gorm.DB
}

// NewScanRepository creates a new scan repository.
func NewScanRepository(db *gorm.DB) *ScanRepository {
	return &ScanRepository{db: db}
}

// ScanFilterMapping defines field mapping for scan filtering.
var ScanFilterMapping = scope.FilterMapping{
	"status":   {Column: "status"},
	"target":   {Column: "target_id"},
	"targetId": {Column: "target_id"},
}

var scanFilterMappingNormalized = scope.NormalizeFilterMapping(ScanFilterMapping)

// ScanStatistics holds scan statistics.
type ScanStatistics = scandomain.QueryStatistics
