package repository

import (
	"gorm.io/gorm"
)

// TaskProgressLogRepository handles task progress log database operations.
type TaskProgressLogRepository struct {
	db *gorm.DB
}

// NewTaskProgressLogRepository creates a new task progress log repository.
func NewTaskProgressLogRepository(db *gorm.DB) *TaskProgressLogRepository {
	return &TaskProgressLogRepository{db: db}
}
