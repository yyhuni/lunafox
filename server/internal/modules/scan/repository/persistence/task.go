package model

import (
	"time"

	"github.com/yyhuni/lunafox/contracts/runtimecontract"
)

// ScanTask represents a task in the queue supporting priority scheduling.
type ScanTask struct {
	ID         int    `gorm:"primaryKey;autoIncrement" json:"id"`
	ScanID     int    `gorm:"not null;index:idx_scan_task_scan_id" json:"scan_id"`
	Stage      int    `gorm:"not null;default:0;index:idx_scan_task_pending_order,priority:2" json:"stage"`
	WorkflowID string `gorm:"column:workflow_id;type:varchar(100);not null" json:"workflow_id"`
	Status     string `gorm:"type:varchar(20);default:'pending';index:idx_scan_task_pending_order,priority:1" json:"status"`
	AgentID    *int   `gorm:"index:idx_scan_task_agent_id" json:"agent_id,omitempty"`
	// WorkflowConfigYAML stores workflow-level YAML slice (not whole scan YAML).
	WorkflowConfigYAML string     `gorm:"column:workflow_config_yaml;type:text" json:"workflow_config_yaml"`
	ErrorMessage       string     `gorm:"type:varchar(4096)" json:"error_message,omitempty"`
	CreatedAt          time.Time  `gorm:"type:timestamptz;default:now();index:idx_scan_task_pending_order,priority:3" json:"created_at"`
	StartedAt          *time.Time `gorm:"type:timestamptz" json:"started_at,omitempty"`
	CompletedAt        *time.Time `gorm:"type:timestamptz" json:"completed_at,omitempty"`
}

func (ScanTask) TableName() string {
	return "scan_task"
}

func (t *ScanTask) WorkspaceDir() string {
	return runtimecontract.BuildTaskWorkspaceDir(t.ScanID, t.ID)
}

// ScanLog represents a scan log entry.
type ScanLog struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ScanID    int       `gorm:"column:scan_id;not null;index:idx_scan_log_scan" json:"scanId"`
	Level     string    `gorm:"column:level;size:10;default:'info'" json:"level"`
	Content   string    `gorm:"column:content;type:text" json:"content"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;index:idx_scan_log_created_at" json:"createdAt"`

	Scan *Scan `gorm:"foreignKey:ScanID" json:"scan,omitempty"`
}

func (ScanLog) TableName() string {
	return "scan_log"
}

const (
	ScanLogLevelInfo    = "info"
	ScanLogLevelWarning = "warning"
	ScanLogLevelError   = "error"
)
