package model

import (
	"time"

	"gorm.io/datatypes"
)

// ScanTask is the persisted workflow step engine execution record.
type ScanTask struct {
	ID                            int            `gorm:"primaryKey;autoIncrement" json:"id"`
	ScanID                        int            `gorm:"column:scan_id;not null;index:idx_scan_task_scan" json:"scanId"`
	StageOrder                    int            `gorm:"column:stage_order;not null;default:0" json:"stageOrder"`
	StageID                       string         `gorm:"column:stage_id;type:varchar(100);not null" json:"stageId"`
	StepOrder                     int            `gorm:"column:step_order;not null;default:0" json:"stepOrder"`
	StepID                        string         `gorm:"column:step_id;type:varchar(100);not null" json:"stepId"`
	EngineID                      string         `gorm:"column:engine_id;type:varchar(255);not null" json:"engineId"`
	EngineConfig                  datatypes.JSON `gorm:"column:engine_config;type:jsonb" json:"engineConfig"`
	TaskExecutionConfig           datatypes.JSON `gorm:"column:task_execution_config;type:jsonb" json:"taskExecutionConfig"`
	ResolvedExecutionPlan         []byte         `gorm:"column:resolved_execution_plan;type:bytea" json:"-"`
	Status                        string         `gorm:"column:status;size:20;default:'pending';index:idx_scan_task_status" json:"status"`
	AssignedAgentID               *int           `gorm:"column:assigned_agent_id;<-:update" json:"assignedAgentId,omitempty"`
	AssignedSessionID             *string        `gorm:"column:assigned_session_id;type:varchar(64);<-:update" json:"assignedSessionId,omitempty"`
	AssignedSessionEpoch          *int64         `gorm:"column:assigned_session_epoch" json:"assignedSessionEpoch,omitempty"`
	AssignedRequestID             *string        `gorm:"column:assigned_request_id;type:varchar(36);<-:update" json:"assignedRequestId,omitempty"`
	TerminalReconciliationPending bool           `gorm:"column:terminal_reconciliation_pending;not null;default:false;index:idx_scan_task_terminal_reconciliation_pending" json:"-"`
	ErrorMessage                  string         `gorm:"column:error_message;size:4096" json:"errorMessage,omitempty"`
	FailureKind                   string         `gorm:"column:failure_kind;size:100" json:"failureKind,omitempty"`
	FailureDetail                 string         `gorm:"column:failure_detail;size:500" json:"failureDetail,omitempty"`
	EngineDiagnostics             datatypes.JSON `gorm:"column:engine_diagnostics;type:jsonb" json:"-"`
	SkipReason                    string         `gorm:"column:skip_reason;size:1000" json:"skipReason,omitempty"`
	CreatedAt                     time.Time      `gorm:"column:created_at;autoCreateTime;index:idx_scan_task_created_at" json:"createdAt"`
	StartedAt                     *time.Time     `gorm:"column:started_at" json:"startedAt,omitempty"`
	CompletedAt                   *time.Time     `gorm:"column:completed_at" json:"completedAt,omitempty"`
}

func (ScanTask) TableName() string {
	return "scan_task"
}
