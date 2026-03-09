package repository

import (
	"context"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"gorm.io/gorm"
)

// ScanTaskRepository defines the interface for scan task data access.
type ScanTaskRepository interface {
	GetByID(ctx context.Context, id int) (*ScanTaskRecord, error)
	PullTask(ctx context.Context, agentID int) (*ScanTaskRecord, error)
	UpdateStatus(ctx context.Context, id int, status string, failure *scandomain.FailureDetail) error
	FailPulledTask(ctx context.Context, id int, failure *scandomain.FailureDetail) error
	ListFailedByScanID(ctx context.Context, scanID int) ([]ScanTaskRecord, error)
	GetStatusCountsByScanID(ctx context.Context, scanID int) (pending, running, completed, failed, cancelled int, err error)
	CountActiveByScanAndStage(ctx context.Context, scanID, stage int) (int, error)
	UnlockNextStage(ctx context.Context, scanID, stage int) (int64, error)
	CancelTasksByScanID(ctx context.Context, scanID int) ([]CancelledTaskInfo, error)
	FailTasksForOfflineAgent(ctx context.Context, agentID int) error
}

// CancelledTaskInfo represents a cancelled task and its assigned agent (if any).
type CancelledTaskInfo struct {
	TaskID  int  `gorm:"column:id"`
	AgentID *int `gorm:"column:agent_id"`
}

type scanTaskRepository struct {
	db *gorm.DB
}

const (
	taskStatusBlocked   = string(scandomain.TaskStatusBlocked)
	taskStatusPending   = string(scandomain.TaskStatusPending)
	taskStatusRunning   = string(scandomain.TaskStatusRunning)
	taskStatusCompleted = string(scandomain.TaskStatusCompleted)
	taskStatusFailed    = string(scandomain.TaskStatusFailed)
	taskStatusCancelled = string(scandomain.TaskStatusCancelled)
)

const (
	scanStatusPending   = string(scandomain.ScanStatusPending)
	scanStatusRunning   = string(scandomain.ScanStatusRunning)
	scanStatusCompleted = string(scandomain.ScanStatusCompleted)
	scanStatusFailed    = string(scandomain.ScanStatusFailed)
	scanStatusCancelled = string(scandomain.ScanStatusCancelled)
)

// NewScanTaskRepository creates a new scan task repository.
func NewScanTaskRepository(db *gorm.DB) ScanTaskRepository {
	return &scanTaskRepository{db: db}
}
