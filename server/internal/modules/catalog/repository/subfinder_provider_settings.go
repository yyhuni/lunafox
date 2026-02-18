package repository

import (
	"gorm.io/gorm"
)

type SubfinderProviderSettingsRepository struct {
	db *gorm.DB
}

func NewSubfinderProviderSettingsRepository(db *gorm.DB) *SubfinderProviderSettingsRepository {
	return &SubfinderProviderSettingsRepository{db: db}
}
