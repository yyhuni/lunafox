package repository

import (
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
)

// SubdomainRepository handles subdomain database operations
type SubdomainRepository struct {
	db *gorm.DB
}

// NewSubdomainRepository creates a new subdomain repository
func NewSubdomainRepository(db *gorm.DB) *SubdomainRepository {
	return &SubdomainRepository{db: db}
}

// SubdomainFilterMapping defines field mapping for subdomain filtering
var SubdomainFilterMapping = scope.FilterMapping{
	"name": {Column: "name"},
}

var subdomainFilterMappingNormalized = scope.NormalizeFilterMapping(SubdomainFilterMapping)
