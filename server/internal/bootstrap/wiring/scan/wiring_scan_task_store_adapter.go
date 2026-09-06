package scanwiring

import (
	"context"

	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
)

func (adapter *scanTaskStoreAdapter) ClaimNextCompatibleSavedExecutionPlan(ctx context.Context, agentID int, sessionID string, sessionEpoch int64, requestID string, supportedEngineAPIMajors []uint32) (*agentexecutionv1.ResolvedEngineExecutionPlan, error) {
	return adapter.repo.ClaimNextCompatibleSavedExecutionPlan(ctx, agentID, sessionID, sessionEpoch, requestID, supportedEngineAPIMajors)
}

type scanTaskStoreAdapter struct {
	repo scanrepo.ScanTaskRepository
}

func newScanTaskStoreAdapter(repo scanrepo.ScanTaskRepository) *scanTaskStoreAdapter {
	return &scanTaskStoreAdapter{repo: repo}
}

func (adapter *scanTaskStoreAdapter) GetByID(ctx context.Context, id int) (*scanapp.ScanTaskRecord, error) {
	return adapter.repo.GetByID(ctx, id)
}

func (adapter *scanTaskStoreAdapter) RequireCurrentAgentExecutionSession(ctx context.Context, agentID int, sessionID string, sessionEpoch int64) error {
	return adapter.repo.RequireCurrentAgentExecutionSession(ctx, agentID, sessionID, sessionEpoch)
}

func (adapter *scanTaskStoreAdapter) CommitScanTaskTerminalStatusForSession(ctx context.Context, id, agentID int, sessionID string, sessionEpoch int64, status string, failure *scanapp.FailureDetail) (bool, error) {
	return adapter.repo.CommitScanTaskTerminalStatusForSession(ctx, id, agentID, sessionID, sessionEpoch, status, failure)
}

func (adapter *scanTaskStoreAdapter) CommitScanTaskTerminalStatusWithDiagnosticsForSession(ctx context.Context, id, agentID int, sessionID string, sessionEpoch int64, status string, failure *scanapp.FailureDetail, diagnostics *scanapp.EngineExecutionDiagnostics) (bool, error) {
	return adapter.repo.CommitScanTaskTerminalStatusWithDiagnosticsForSession(ctx, id, agentID, sessionID, sessionEpoch, status, failure, diagnostics)
}

func (adapter *scanTaskStoreAdapter) ListTerminalTasksPendingReconciliation(ctx context.Context, afterTaskID, limit int) ([]scanapp.ScanTaskRecord, error) {
	return adapter.repo.ListTerminalTasksPendingReconciliation(ctx, afterTaskID, limit)
}

func (adapter *scanTaskStoreAdapter) ListSupersededSessionTerminalTasksPendingReconciliation(ctx context.Context, agentID int, currentSessionEpoch int64, limit int) ([]scanapp.ScanTaskRecord, error) {
	return adapter.repo.ListSupersededSessionTerminalTasksPendingReconciliation(ctx, agentID, currentSessionEpoch, limit)
}

func (adapter *scanTaskStoreAdapter) ClearTerminalTaskReconciliationPending(ctx context.Context, taskID int) error {
	return adapter.repo.ClearTerminalTaskReconciliationPending(ctx, taskID)
}

func (adapter *scanTaskStoreAdapter) FailClaimedTask(ctx context.Context, id int, failure *scanapp.FailureDetail) error {
	return adapter.repo.FailClaimedTask(ctx, id, failure)
}

func (adapter *scanTaskStoreAdapter) SkipUnstartedTasksByScanID(ctx context.Context, scanID int, reason string) (int64, error) {
	return adapter.repo.SkipUnstartedTasksByScanID(ctx, scanID, reason)
}

func (adapter *scanTaskStoreAdapter) CancelUnstartedTasksByScanID(ctx context.Context, scanID int) (int64, error) {
	return adapter.repo.CancelUnstartedTasksByScanID(ctx, scanID)
}

func (adapter *scanTaskStoreAdapter) ListFailedByScanID(ctx context.Context, scanID int) ([]scanapp.ScanTaskRecord, error) {
	return adapter.repo.ListFailedByScanID(ctx, scanID)
}

func (adapter *scanTaskStoreAdapter) CountByStatusForScanID(ctx context.Context, scanID int) (pending, running, completed, failed, cancelled, skipped int, err error) {
	return adapter.repo.CountByStatusForScanID(ctx, scanID)
}

func (adapter *scanTaskStoreAdapter) CountActiveByScanAndStageOrder(ctx context.Context, scanID, scanWorkflowStageOrder int) (int, error) {
	return adapter.repo.CountActiveByScanAndStageOrder(ctx, scanID, scanWorkflowStageOrder)
}

func (adapter *scanTaskStoreAdapter) UnlockNextStageOrder(ctx context.Context, scanID, scanWorkflowStageOrder int) (int64, error) {
	return adapter.repo.UnlockNextStageOrder(ctx, scanID, scanWorkflowStageOrder)
}

func (adapter *scanTaskStoreAdapter) FailTasksForSupersededAgentSession(ctx context.Context, agentID int, currentSessionEpoch int64) ([]int, error) {
	return adapter.repo.FailTasksForSupersededAgentSession(ctx, agentID, currentSessionEpoch)
}
