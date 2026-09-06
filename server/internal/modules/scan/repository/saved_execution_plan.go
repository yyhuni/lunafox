package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/yyhuni/lunafox/contracts/agentexecution"
	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"gorm.io/gorm"
)

// SavedExecutionPlanReader is an additive v2 claim boundary. It reads only
// the immutable bytes saved at scan creation; it never recompiles package or
// workflow facts.
type SavedExecutionPlanReader interface {
	GetSavedExecutionPlan(ctx context.Context, taskID int) (*agentexecutionv1.ResolvedEngineExecutionPlan, error)
	GetSavedExecutionPlanLease(ctx context.Context, taskID int) (*scandomain.SavedExecutionPlanLease, error)
}

var _ SavedExecutionPlanReader = (*scanTaskRepository)(nil)

type savedExecutionPlanRow struct {
	ID                    int
	ScanID                int    `gorm:"column:scan_id"`
	InputSource           string `gorm:"column:input_source"`
	Status                string
	AgentID               *int    `gorm:"column:assigned_agent_id"`
	AssignedSessionID     *string `gorm:"column:assigned_session_id"`
	AssignedSessionEpoch  *int64  `gorm:"column:assigned_session_epoch"`
	ResolvedExecutionPlan []byte  `gorm:"column:resolved_execution_plan"`
}

func (r *scanTaskRepository) GetSavedExecutionPlanLease(ctx context.Context, taskID int) (*scandomain.SavedExecutionPlanLease, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("scan task repository is not initialized")
	}
	if ctx == nil {
		return nil, fmt.Errorf("context is required")
	}
	if taskID <= 0 {
		return nil, fmt.Errorf("task id must be positive")
	}
	var row savedExecutionPlanRow
	err := r.db.WithContext(ctx).
		Table("scan_task AS st").
		Joins("JOIN scan AS s ON s.id = st.scan_id").
		Select("st.id, st.scan_id, s.input_source, st.status, st.assigned_agent_id, st.assigned_session_id, st.assigned_session_epoch, st.resolved_execution_plan").
		Where("st.id = ? AND s.deleted_at IS NULL", taskID).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: task %d", scandomain.ErrSavedExecutionPlanLeaseNotFound, taskID)
		}
		return nil, err
	}
	if err := validateSavedExecutionPlanLeaseRow(row); err != nil {
		return nil, fmt.Errorf("validate saved execution plan lease for task %d: %w", taskID, err)
	}
	inputSource, ok := scandomain.ParseDatabaseInputSource(row.InputSource)
	if !ok {
		return nil, fmt.Errorf("%w: persisted value %q", scandomain.ErrInvalidInputSource, row.InputSource)
	}
	return &scandomain.SavedExecutionPlanLease{
		TaskID:                row.ID,
		ScanID:                row.ScanID,
		InputSource:           inputSource,
		Status:                row.Status,
		AgentID:               row.AgentID,
		AssignedSessionID:     row.AssignedSessionID,
		AssignedSessionEpoch:  row.AssignedSessionEpoch,
		ResolvedExecutionPlan: append([]byte(nil), row.ResolvedExecutionPlan...),
	}, nil
}

func validateSavedExecutionPlanLeaseRow(row savedExecutionPlanRow) error {
	if row.Status == taskStatusSkipped {
		if len(row.ResolvedExecutionPlan) != 0 {
			return fmt.Errorf("skipped task must not have a saved execution plan")
		}
		return nil
	}
	plan, err := agentexecution.UnmarshalResolvedEngineExecutionPlan(row.ResolvedExecutionPlan)
	if err != nil {
		return fmt.Errorf("saved execution plan is invalid: %w", err)
	}
	// Closed-plan validation proves internal consistency; these checks bind that
	// otherwise-valid scope to the authoritative task row selected for the lease.
	if plan.GetWorkflowStep().GetScan() != resourcenames.Scan(row.ScanID) {
		return fmt.Errorf("saved execution plan scan scope does not match persisted task")
	}
	if plan.GetTask() != resourcenames.Task(row.ScanID, row.ID) {
		return fmt.Errorf("saved execution plan task scope does not match persisted task")
	}
	return nil
}

func (r *scanTaskRepository) GetSavedExecutionPlan(ctx context.Context, taskID int) (*agentexecutionv1.ResolvedEngineExecutionPlan, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("scan task repository is not initialized")
	}
	if taskID <= 0 {
		return nil, fmt.Errorf("task id must be positive")
	}
	lease, err := r.GetSavedExecutionPlanLease(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if lease.Status == taskStatusSkipped {
		if len(lease.ResolvedExecutionPlan) != 0 {
			return nil, fmt.Errorf("skipped task %d must not have a saved execution plan", taskID)
		}
		return nil, nil
	}
	if len(lease.ResolvedExecutionPlan) == 0 {
		return nil, fmt.Errorf("task %d has no saved execution plan", taskID)
	}
	plan, err := agentexecution.UnmarshalResolvedEngineExecutionPlan(lease.ResolvedExecutionPlan)
	if err != nil {
		return nil, fmt.Errorf("read saved execution plan for task %d: %w", taskID, err)
	}
	return plan, nil
}
