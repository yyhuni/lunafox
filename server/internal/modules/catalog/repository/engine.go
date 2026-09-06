package repository

import "gorm.io/gorm"

// EngineRepository persists the current installed engine inventory.
type EngineRepository struct{ db *gorm.DB }

func NewEngineRepository(db *gorm.DB) *EngineRepository { return &EngineRepository{db: db} }
