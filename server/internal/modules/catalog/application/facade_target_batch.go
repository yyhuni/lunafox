package application

import (
	"context"

	"github.com/yyhuni/lunafox/server/internal/modules/catalog/dto"
)

// BatchCreate creates multiple targets at once.
func (service *TargetFacade) BatchCreate(req *dto.BatchCreateTargetRequest) *dto.BatchCreateTargetResponse {
	targetNames := make([]string, 0, len(req.Targets))
	for _, item := range req.Targets {
		targetNames = append(targetNames, item.Name)
	}

	result := service.commandService.BatchCreateTargets(context.Background(), targetNames, req.OrganizationID)
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
