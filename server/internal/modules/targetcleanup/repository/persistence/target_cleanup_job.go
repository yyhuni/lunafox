package persistence

import "time"

// TargetCleanupJob maps the durable internal queue table. It intentionally
// mirrors only recovery diagnostics; phase, cursor, and worker lease state are
// prohibited by the cleanup reconciliation contract.
type TargetCleanupJob struct {
	ID          int        `gorm:"primaryKey;autoIncrement"`
	TargetID    int        `gorm:"column:target_id;not null;uniqueIndex"`
	Status      string     `gorm:"column:status;not null"`
	RetryCount  int        `gorm:"column:retry_count;not null"`
	NextRetryAt time.Time  `gorm:"column:next_retry_at;not null"`
	LastError   string     `gorm:"column:last_error;not null"`
	CompletedAt *time.Time `gorm:"column:completed_at"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (TargetCleanupJob) TableName() string {
	return "target_cleanup_job"
}
