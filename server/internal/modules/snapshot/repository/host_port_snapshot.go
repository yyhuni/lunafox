package repository

import (
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
)

// HostPortSnapshotRepository handles host-port mapping snapshot database operations
type HostPortSnapshotRepository struct {
	db *gorm.DB
}

// NewHostPortSnapshotRepository creates a new host-port mapping snapshot repository
func NewHostPortSnapshotRepository(db *gorm.DB) *HostPortSnapshotRepository {
	return &HostPortSnapshotRepository{db: db}
}

// HostPortSnapshotFilterMapping defines field mapping for host-port snapshot filtering
var HostPortSnapshotFilterMapping = scope.FilterMapping{
	"host": {Column: "host"},
	"ip":   {Column: "ip", NeedsCast: true},
	"port": {Column: "port", IsNumeric: true},
}

var hostPortSnapshotFilterMappingNormalized = scope.NormalizeFilterMapping(HostPortSnapshotFilterMapping)
