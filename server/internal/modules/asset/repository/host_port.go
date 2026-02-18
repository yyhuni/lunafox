package repository

import (
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
)

// HostPortRepository handles host-port mapping (host_port_mapping) database operations
type HostPortRepository struct {
	db *gorm.DB
}

// NewHostPortRepository creates a new host-port repository
func NewHostPortRepository(db *gorm.DB) *HostPortRepository {
	return &HostPortRepository{db: db}
}

// HostPortFilterMapping defines field mapping for filtering
var HostPortFilterMapping = scope.FilterMapping{
	"host": {Column: "host"},
	"ip":   {Column: "ip", NeedsCast: true},
	"port": {Column: "port", IsNumeric: true},
}

var hostPortFilterMappingNormalized = scope.NormalizeFilterMapping(HostPortFilterMapping)
