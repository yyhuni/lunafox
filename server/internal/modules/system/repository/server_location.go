package repository

import (
	systemdomain "github.com/yyhuni/lunafox/server/internal/modules/system/domain"
	"gorm.io/gorm"
)

const serverLocationSingletonID int16 = 1

type serverLocationRepository struct {
	db *gorm.DB
}

// NewServerLocationRepository creates the system-owned singleton location repository.
func NewServerLocationRepository(db *gorm.DB) systemdomain.ServerLocationRepository {
	return &serverLocationRepository{db: db}
}
