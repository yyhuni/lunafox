package application

import (
	"context"

	"github.com/yyhuni/lunafox/server/internal/modules/catalog/dto"
)

// BatchCreate creates multiple targets at once.
func (service *TargetFacade) BatchCreate(req *dto.BatchCreateTargetRequest, organizationID *int) *dto.BatchCreateTargetResponse {
	targetNames := make([]string, 0, len(req.Targets))
	for _, item := range req.Targets {
		targetNames = append(targetNames, item.Name)
	}
	return batchCreateResponse(service.commandService.BatchCreateTargets(context.Background(), targetNames, organizationID))
}

// BatchCreateContext preserves the caller-owned cancellation and exposes the
// shared command error so MCP can map domain failures without REST coupling.
func (service *TargetFacade) BatchCreateContext(ctx context.Context, req *dto.BatchCreateTargetRequest, organizationID *int) (*dto.BatchCreateTargetResponse, error) {
	targetNames := make([]string, 0, len(req.Targets))
	for _, item := range req.Targets {
		targetNames = append(targetNames, item.Name)
	}

	result, err := service.BatchCreateTargetsContext(ctx, targetNames, organizationID)
	if err != nil {
		return nil, err
	}
	return batchCreateResponse(result), nil
}

// BatchCreateTargetsContext exposes the shared batch command to non-HTTP
// adapters while preserving caller cancellation and transaction semantics.
func (service *TargetFacade) BatchCreateTargetsContext(ctx context.Context, targetNames []string, organizationID *int) (*BatchCreateResult, error) {
	return service.commandService.BatchCreateTargetsContext(ctx, targetNames, organizationID)
}

func batchCreateResponse(result *BatchCreateResult) *dto.BatchCreateTargetResponse {
	if result == nil {
		return &dto.BatchCreateTargetResponse{}
	}
	failedTargets := make([]dto.FailedTarget, 0, len(result.FailedTargets))
	for _, item := range result.FailedTargets {
		failedTargets = append(failedTargets, dto.FailedTarget{Name: item.Name, Reason: item.Reason})
	}

	return &dto.BatchCreateTargetResponse{
		CreatedCount:  result.CreatedCount,
		FailedCount:   result.FailedCount,
		FailedTargets: failedTargets,
		Message:       result.Message,
	}
}
