package repository

import (
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
)

// SubdomainSnapshotRepository handles subdomain snapshot database operations
type SubdomainSnapshotRepository struct {
	db *gorm.DB
}

// NewSubdomainSnapshotRepository creates a new subdomain snapshot repository
func NewSubdomainSnapshotRepository(db *gorm.DB) *SubdomainSnapshotRepository {
	return &SubdomainSnapshotRepository{db: db}
}

// SubdomainSnapshotFilterMapping defines field mapping for subdomain snapshot filtering
var SubdomainSnapshotFilterMapping = scope.FilterMapping{
	"name": {Column: "name"},
}

var subdomainSnapshotFilterMappingNormalized = scope.NormalizeFilterMapping(SubdomainSnapshotFilterMapping)
