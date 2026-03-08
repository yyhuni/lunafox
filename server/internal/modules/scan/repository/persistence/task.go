package model

import (
	"time"

	"github.com/yyhuni/lunafox/contracts/runtimecontract"
	"gorm.io/datatypes"
)

// ScanTask represents a task in the queue supporting priority scheduling.
type ScanTask struct {
	ID             int            `gorm:"primaryKey;autoIncrement" json:"id"`
	ScanID         int            `gorm:"not null;index:idx_scan_task_scan_id" json:"scanId"`
	Stage          int            `gorm:"not null;default:0;index:idx_scan_task_pending_order,priority:2" json:"stage"`
	WorkflowID     string         `gorm:"column:workflow_id;type:varchar(100);not null" json:"workflowId"`
	Status         string         `gorm:"type:varchar(20);default:'pending';index:idx_scan_task_pending_order,priority:1" json:"status"`
	AgentID        *int           `gorm:"index:idx_scan_task_agent_id" json:"agentId,omitempty"`
	WorkflowConfig datatypes.JSON `gorm:"column:workflow_config;type:jsonb" json:"workflowConfig"`
	ErrorMessage   string         `gorm:"type:varchar(4096)" json:"errorMessage,omitempty"`
	CreatedAt      time.Time      `gorm:"type:timestamptz;default:now();index:idx_scan_task_pending_order,priority:3" json:"createdAt"`
	StartedAt      *time.Time     `gorm:"type:timestamptz" json:"startedAt,omitempty"`
	CompletedAt    *time.Time     `gorm:"type:timestamptz" json:"completedAt,omitempty"`
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
	Scan      *Scan     `gorm:"foreignKey:ScanID" json:"scan,omitempty"`
}

func (ScanLog) TableName() string {
	return "scan_log"
}

const (
	ScanLogLevelInfo    = "info"
	ScanLogLevelWarning = "warning"
	ScanLogLevelError   = "error"
)
