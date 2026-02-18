package repository

import (
	"errors"

	"github.com/yyhuni/lunafox/server/internal/modules/identity/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
)

// ErrTargetNotFound indicates one or more target IDs do not exist.
var ErrTargetNotFound = errors.New("one or more target IDs do not exist")

// PostgreSQL error codes.
const (
	pgForeignKeyViolation = "23503"
)

// OrganizationFilterMapping defines filter fields for organization.
var OrganizationFilterMapping = scope.FilterMapping{
	"name": {Column: "organization.name", IsArray: false},
}

var organizationFilterMappingNormalized = scope.NormalizeFilterMapping(OrganizationFilterMapping)

// OrganizationWithCount represents organization with target count.
type OrganizationWithCount struct {
	model.Organization
	TargetCount int64 `gorm:"column:target_count"`
}

// OrganizationRepository handles organization database operations.
type OrganizationRepository struct {
	db *gorm.DB
}

// NewOrganizationRepository creates a new organization repository.
func NewOrganizationRepository(db *gorm.DB) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}
