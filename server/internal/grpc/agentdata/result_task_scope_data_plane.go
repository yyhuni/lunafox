package agentdata

import (
	"context"
	"errors"
	"strings"

	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

var (
	errResultTaskNotRunning        = errors.New("result task is not running")
	errResultTaskOwnershipMismatch = errors.New("result task is not owned by the authenticated Agent")
	errResultTaskSessionMismatch   = errors.New("result task session epoch is not current")
)

type ResultScanTargetLookup interface {
	GetTargetRefByScanIDContext(ctx context.Context, scanID int) (*scanrepo.ScanTargetRecord, error)
}

type resultTaskScopeDataPlane struct {
	tasks       scanrepo.ScanTaskRepository
	scanTargets ResultScanTargetLookup
}

func NewResultTaskScopeDataPlane(tasks scanrepo.ScanTaskRepository, scanTargets ResultScanTargetLookup) ResultTaskScopeDataPlane {
	return &resultTaskScopeDataPlane{tasks: tasks, scanTargets: scanTargets}
}

func (plane *resultTaskScopeDataPlane) GetResultTaskScope(ctx context.Context, request ResultTaskScopeRequest) (*ResultTaskScope, error) {
	if plane == nil || plane.tasks == nil || plane.scanTargets == nil {
		return nil, errDataPlaneDependencyMissing()
	}
	if request.TaskID <= 0 || request.AgentID <= 0 || request.SessionID == "" || request.SessionID != strings.TrimSpace(request.SessionID) || request.SessionEpoch <= 0 {
		return nil, errResultTaskSessionMismatch
	}
	task, err := plane.tasks.GetByID(ctx, request.TaskID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, scanapp.ErrScanTaskNotFound
		}
		return nil, err
	}
	if task == nil {
		return nil, scanapp.ErrScanTaskNotFound
	}
	if task.Status != string(scandomain.TaskStatusRunning) {
		return nil, errResultTaskNotRunning
	}
	if task.AssignedSessionID == nil || strings.TrimSpace(*task.AssignedSessionID) == "" || *task.AssignedSessionID != request.SessionID || task.AssignedSessionEpoch == nil || *task.AssignedSessionEpoch != request.SessionEpoch {
		return nil, errResultTaskSessionMismatch
	}
	if task.AssignedAgentID == nil || *task.AssignedAgentID != request.AgentID {
		return nil, errResultTaskOwnershipMismatch
	}
	target, err := plane.scanTargets.GetTargetRefByScanIDContext(ctx, task.ScanID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, scanapp.ErrScanNotFound
		}
		return nil, err
	}
	if target == nil {
		return nil, scanapp.ErrScanNotFound
	}
	return &ResultTaskScope{
		TaskID:   task.ID,
		ScanID:   task.ScanID,
		TargetID: target.ID,
	}, nil
}
