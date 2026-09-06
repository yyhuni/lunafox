package model

import (
	"time"
)

// TaskProgressLog represents a persisted task progress log entry.
type TaskProgressLog struct {
	ID        int64      `gorm:"primaryKey;autoIncrement;index:idx_task_progress_log_task,priority:2" json:"id"`
	ScanID    int        `gorm:"column:scan_id;primaryKey;not null;index:idx_task_progress_log_scan_id;index:idx_task_progress_log_task,priority:1" json:"scanId"`
	TaskID    int        `gorm:"column:task_id;not null;index:idx_task_progress_log_task,priority:2" json:"taskId"`
	RequestID string     `gorm:"column:request_id;size:100;default:''" json:"requestId"`
	Sequence  int64      `gorm:"column:sequence;default:0" json:"sequence"`
	Level     string     `gorm:"column:level;size:10;default:'info'" json:"level"`
	Content   string     `gorm:"column:content;type:text" json:"content"`
	EmittedAt *time.Time `gorm:"column:emitted_at" json:"emittedAt,omitempty"`
	CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime;index:idx_task_progress_log_created_at" json:"createdAt"`
}

func (TaskProgressLog) TableName() string {
	return "task_progress_log"
}

const (
	TaskProgressLogLevelInfo    = "info"
	TaskProgressLogLevelWarning = "warning"
	TaskProgressLogLevelError   = "error"
)
