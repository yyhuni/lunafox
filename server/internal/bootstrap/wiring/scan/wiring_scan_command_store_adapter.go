package scanwiring

import (
	"context"
	"fmt"

	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
)

type scanCommandStoreAdapter struct {
	repo                    *scanrepo.ScanRepository
	effectivePolicyResolver scanapp.EffectiveBlacklistPolicyResolver
}

func newScanCommandStoreAdapter(repo *scanrepo.ScanRepository, effectivePolicyResolver scanapp.EffectiveBlacklistPolicyResolver) *scanCommandStoreAdapter {
	return &scanCommandStoreAdapter{repo: repo, effectivePolicyResolver: effectivePolicyResolver}
}

func (adapter *scanCommandStoreAdapter) GetLifecycleRefByID(id int) (*scanapp.QueryScan, error) {
	return adapter.repo.GetByID(id)
}

func (adapter *scanCommandStoreAdapter) FindByIDs(ids []int) ([]scanapp.QueryScan, error) {
	return adapter.repo.FindByIDs(ids)
}

func (adapter *scanCommandStoreAdapter) CreateWithScanTasksAndPlans(ctx context.Context, scan *scanapp.CreateScan, finalize scanapp.ScanCreateTaskFinalizer) error {
	if adapter == nil || adapter.repo == nil || adapter.effectivePolicyResolver == nil {
		return fmt.Errorf("scan command store effective blacklist policy resolver is required")
	}
	return adapter.repo.CreateWithScanTasksAndPlans(ctx, scan, adapter.effectivePolicyResolver.ResolveEffectivePatternsForScan, finalize)
}

func (adapter *scanCommandStoreAdapter) CreateWithScanTasksAndPlansAndMCPOperation(ctx context.Context, scan *scanapp.CreateScan, operation *scanapp.MCPScanOperationCreate, finalize scanapp.ScanCreateTaskFinalizer) error {
	if adapter == nil || adapter.repo == nil || adapter.effectivePolicyResolver == nil || operation == nil {
		return fmt.Errorf("scan MCP operation store is not configured")
	}
	return adapter.repo.CreateWithScanTasksAndPlansAndMCPOperation(
		ctx,
		scan,
		operation.ID,
		operation.RequestID,
		operation.RequestFingerprint,
		operation.ReplayExpiresAt,
		adapter.effectivePolicyResolver.ResolveEffectivePatternsForScan,
		finalize,
	)
}

func (adapter *scanCommandStoreAdapter) FindMCPRequestReplay(ctx context.Context, requestID string) (*scanapp.MCPRequestReplay, error) {
	if adapter == nil || adapter.repo == nil {
		return nil, fmt.Errorf("scan MCP operation store is not configured")
	}
	replay, err := adapter.repo.FindMCPRequestReplay(ctx, requestID)
	if err != nil || replay == nil {
		return nil, err
	}
	return &scanapp.MCPRequestReplay{
		RequestID:          replay.RequestID,
		Action:             replay.Action,
		RequestFingerprint: replay.RequestFingerprint,
		Response:           append([]byte(nil), replay.Response...),
		CreatedAt:          replay.CreatedAt.UTC(),
		ExpiresAt:          replay.ExpiresAt.UTC(),
	}, nil
}

func (adapter *scanCommandStoreAdapter) GetMCPScanOperation(ctx context.Context, operationID string) (*scanapp.MCPScanOperation, error) {
	if adapter == nil || adapter.repo == nil {
		return nil, fmt.Errorf("scan MCP operation store is not configured")
	}
	operation, err := adapter.repo.GetMCPOperation(ctx, operationID)
	if err != nil || operation == nil {
		return nil, err
	}
	return &scanapp.MCPScanOperation{
		ID:          operation.ID,
		ScanID:      operation.ScanID,
		TargetID:    operation.TargetID,
		Status:      operation.ScanStatus,
		Phase:       operation.Phase,
		Progress:    operation.Progress,
		CurrentTask: operation.CurrentTask,
		FailureKind: operation.FailureKind,
		CreatedAt:   operation.CreatedAt.UTC(),
		UpdatedAt:   operation.UpdatedAt.UTC(),
	}, nil
}

func (adapter *scanCommandStoreAdapter) BatchSoftDelete(ids []int) (int64, []string, error) {
	return adapter.repo.BatchSoftDelete(ids)
}

func (adapter *scanCommandStoreAdapter) UpdateScanStatus(id int, status string, failure *scanapp.FailureDetail) error {
	return adapter.repo.UpdateScanStatus(id, status, failure)
}

func (adapter *scanCommandStoreAdapter) RefreshSubdomainResultSummary(ctx context.Context, scanID int, targetID int) error {
	return adapter.repo.RefreshSubdomainResultSummary(ctx, scanID, targetID)
}
