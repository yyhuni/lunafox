package application

import (
	"context"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

func (service *TargetCommandService) BatchCreateTargets(ctx context.Context, names []string, organizationID *int) *BatchCreateResult {
	_ = ctx

	failedTargets := []FailedTarget{}
	validTargets := make([]catalogdomain.Target, 0, len(names))
	validNames := make([]string, 0, len(names))
	seen := make(map[string]bool)

	for _, rawName := range names {
		if catalogdomain.NormalizeBatchTargetName(rawName) == "" {
			continue
		}

		target, err := catalogdomain.BuildBatchTarget(rawName)
		if err != nil {
			failedTargets = append(failedTargets, FailedTarget{Name: catalogdomain.NormalizeBatchTargetName(rawName), Reason: "unrecognized target format"})
			continue
		}

		if seen[target.Name] {
			continue
		}
		seen[target.Name] = true

		validTargets = append(validTargets, *target)
		validNames = append(validNames, target.Name)
	}

	if len(validTargets) == 0 {
		return &BatchCreateResult{
			CreatedCount:  0,
			FailedCount:   len(failedTargets),
			FailedTargets: failedTargets,
			Message:       "no valid targets",
		}
	}

	if organizationID != nil {
		if service.organizationRef == nil {
			return &BatchCreateResult{
				CreatedCount:  0,
				FailedCount:   len(names),
				FailedTargets: failedTargets,
				Message:       "organization repository not configured",
			}
		}

		exists, err := service.organizationRef.ExistsByID(*organizationID)
		if err != nil {
			return &BatchCreateResult{
				CreatedCount:  0,
				FailedCount:   len(names),
				FailedTargets: failedTargets,
				Message:       "failed to validate organization: " + err.Error(),
			}
		}
		if !exists {
			return &BatchCreateResult{
				CreatedCount:  0,
				FailedCount:   len(names),
				FailedTargets: failedTargets,
				Message:       "organization not found",
			}
		}
	}

	createdCount, err := service.store.BulkCreateIgnoreConflicts(validTargets)
	if err != nil {
		return &BatchCreateResult{
			CreatedCount:  0,
			FailedCount:   len(names),
			FailedTargets: failedTargets,
			Message:       "batch create failed: " + err.Error(),
		}
	}

	if organizationID != nil {
		targets, err := service.store.FindByNames(validNames)
		if err != nil {
			return &BatchCreateResult{
				CreatedCount:  createdCount,
				FailedCount:   len(failedTargets),
				FailedTargets: failedTargets,
				Message:       "targets created, but failed to associate with organization: " + err.Error(),
			}
		}

		targetIDs := make([]int, len(targets))
		for index, target := range targets {
			targetIDs[index] = target.ID
		}

		if err := service.organizationRef.BulkAddTargets(*organizationID, targetIDs); err != nil {
			return &BatchCreateResult{
				CreatedCount:  createdCount,
				FailedCount:   len(failedTargets),
				FailedTargets: failedTargets,
				Message:       "targets created, but failed to associate with organization: " + err.Error(),
			}
		}
	}

	return &BatchCreateResult{
		CreatedCount:  createdCount,
		FailedCount:   len(failedTargets),
		FailedTargets: failedTargets,
		Message:       "successfully created targets",
	}
}
