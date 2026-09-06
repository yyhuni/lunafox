package repository

import "gorm.io/gorm"

// HostPortSnapshotRepository handles host-port mapping snapshot database operations
type HostPortSnapshotRepository struct {
	db *gorm.DB
}

// NewHostPortSnapshotRepository creates a new host-port mapping snapshot repository
func NewHostPortSnapshotRepository(db *gorm.DB) *HostPortSnapshotRepository {
	return &HostPortSnapshotRepository{db: db}
}
