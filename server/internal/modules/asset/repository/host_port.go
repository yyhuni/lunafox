package repository

import "gorm.io/gorm"

// HostPortRepository handles host-port mapping (host_port_mapping) database operations
type HostPortRepository struct {
	db *gorm.DB
}

// NewHostPortRepository creates a new host-port repository
func NewHostPortRepository(db *gorm.DB) *HostPortRepository {
	return &HostPortRepository{db: db}
}
