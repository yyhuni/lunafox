package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

func (service *ScanFacade) CreateQuick(ctx context.Context, req *CreateQuickRequest) (*QuickScanResult, error) {
	if service.createService == nil {
		return nil, ErrScanInvalidConfig
	}
	if req == nil {
		return nil, ErrCreateInvalidConfig
	}
	if err := validateScanTriggerType(req.TriggerType); err != nil {
		return nil, err
	}
	if err := validateInputSource(req.InputSource); err != nil {
		return nil, err
	}

	result, err := service.createService.CreateQuick(ctx, &CreateQuickInput{
		Targets:       append([]string(nil), req.Targets...),
		ScanWorkflow:  req.ScanWorkflow,
		Configuration: req.Configuration,
		AgentID:       cloneIntPtr(req.AgentID),
		InputSource:   req.InputSource,
		TriggerType:   req.TriggerType,
	})
	if err != nil {
		if _, ok := AsTaskExecutionError(err); ok {
			return nil, err
		}
		if isConfigResourceValidationError(err) {
			return nil, err
		}
		if isPlanTaskConfigError(err) {
			return nil, fmt.Errorf("%w: %v", ErrScanEngineConfigInvalid, err)
		}
		if dberrors.IsRecordNotFound(err) || errors.Is(err, ErrCreateTargetNotFound) {
			return nil, ErrTargetNotFound
		}
		if errors.Is(err, ErrNoTargetsForScan) {
			return nil, ErrNoTargetsForScan
		}
		if errors.Is(err, ErrCreateInvalidConfig) {
			return nil, fmt.Errorf("%w: %v", ErrScanEngineConfigInvalid, err)
		}
		if errors.Is(err, ErrCreateTargetLookupNotReady) {
			return nil, ErrScanInvalidConfig
		}
		if errors.Is(err, ErrCreateAgentNotFound) {
			return nil, ErrScanAgentNotFound
		}
		if errors.Is(err, ErrCreateScanWorkflowEngineUnavailable) {
			return nil, ErrScanWorkflowEngineUnavailable
		}
		if errors.Is(err, ErrCreateInvalidScanWorkflow) {
			return nil, wrapScanInvalidScanWorkflow(err)
		}
		if errors.Is(err, ErrCreateNoScanWorkflows) {
			return nil, ErrScanNoScanWorkflows
		}
		return nil, err
	}

	out := &QuickScanResult{
		Scans:       make([]QueryScan, 0, len(result.Scans)),
		TargetStats: result.TargetStats,
		Errors:      append([]QuickTargetError(nil), result.Errors...),
	}
	for index := range result.Scans {
		out.Scans = append(out.Scans, *createScanToQueryScan(&result.Scans[index]))
	}
	return out, nil
}

func (service *ScanFacade) CreateBatch(ctx context.Context, req *CreateBatchRequest) (*BatchScanResult, error) {
	if service.createService == nil {
		return nil, ErrScanInvalidConfig
	}
	if req == nil {
		return nil, ErrCreateInvalidConfig
	}
	if err := validateScanTriggerType(req.TriggerType); err != nil {
		return nil, err
	}
	if err := validateInputSource(req.InputSource); err != nil {
		return nil, err
	}

	result, err := service.createService.CreateBatch(ctx, &CreateBatchInput{
		Requests:      append([]CreateBatchItem(nil), req.Requests...),
		ScanWorkflow:  req.ScanWorkflow,
		Configuration: req.Configuration,
		AgentID:       cloneIntPtr(req.AgentID),
		InputSource:   req.InputSource,
		TriggerType:   req.TriggerType,
	})
	out := createBatchResultToBatchScanResult(result)
	if err != nil {
		if _, ok := AsTaskExecutionError(err); ok {
			return out, err
		}
		if isConfigResourceValidationError(err) {
			return out, err
		}
		if isPlanTaskConfigError(err) {
			return out, fmt.Errorf("%w: %v", ErrScanEngineConfigInvalid, err)
		}
		if errors.Is(err, ErrNoTargetsForScan) {
			return out, ErrNoTargetsForScan
		}
		if errors.Is(err, ErrCreateInvalidConfig) {
			return out, fmt.Errorf("%w: %v", ErrScanEngineConfigInvalid, err)
		}
		if errors.Is(err, ErrCreateTargetLookupNotReady) {
			return out, ErrScanInvalidConfig
		}
		if errors.Is(err, ErrCreateAgentNotFound) {
			return out, ErrScanAgentNotFound
		}
		if errors.Is(err, ErrCreateScanWorkflowEngineUnavailable) {
			return out, ErrScanWorkflowEngineUnavailable
		}
		if errors.Is(err, ErrCreateInvalidScanWorkflow) {
			return out, wrapScanInvalidScanWorkflow(err)
		}
		if errors.Is(err, ErrCreateNoScanWorkflows) {
			return out, ErrScanNoScanWorkflows
		}
		return out, err
	}

	return out, nil
}

func createBatchResultToBatchScanResult(result *CreateBatchResult) *BatchScanResult {
	if result == nil {
		return nil
	}
	out := &BatchScanResult{
		Scans:        make([]QueryScan, 0, len(result.Scans)),
		CreatedCount: result.CreatedCount,
		Skipped:      append([]CreateBatchItemOutcome(nil), result.Skipped...),
		Failed:       append([]CreateBatchItemOutcome(nil), result.Failed...),
	}
	for index := range result.Scans {
		out.Scans = append(out.Scans, *createScanToQueryScan(&result.Scans[index]))
	}
	return out
}

func isPlanTaskConfigError(err error) bool {
	var planErr *PlanTaskError
	return errors.As(err, &planErr) && planErr.Kind == PlanTaskConfigError
}

func isConfigResourceValidationError(err error) bool {
	var resourceErr *ConfigResourceValidationError
	return errors.As(err, &resourceErr)
}

func createScanToQueryScan(scan *CreateScan) *QueryScan {
	if scan == nil {
		return nil
	}

	result := &QueryScan{
		ID:             scan.ID,
		TargetID:       scan.TargetID,
		ScanWorkflowID: scan.ScanWorkflowID,
		Configuration:  scan.Configuration,
		InputSource:    scan.InputSource,
		TriggerType:    scan.TriggerType,
		AssignmentMode: scan.AssignmentMode,
		AgentID:        cloneIntPtr(scan.AgentID),
		Status:         scan.Status,
		CreatedAt:      scan.CreatedAt,
	}
	if scan.Target != nil {
		result.Target = &QueryTargetRef{
			ID:   scan.Target.ID,
			Name: scan.Target.Name,
			Type: scan.Target.Type,
		}
	}

	return result
}

func wrapScanInvalidScanWorkflow(err error) error {
	if err == nil {
		return ErrScanInvalidScanWorkflow
	}

	detail := strings.TrimSpace(strings.TrimPrefix(err.Error(), ErrCreateInvalidScanWorkflow.Error()))
	detail = strings.TrimSpace(strings.TrimPrefix(detail, ":"))
	if detail == "" {
		return ErrScanInvalidScanWorkflow
	}

	return fmt.Errorf("%w: %s", ErrScanInvalidScanWorkflow, detail)
}
