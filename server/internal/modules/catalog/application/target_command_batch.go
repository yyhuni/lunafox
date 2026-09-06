package application

import (
	"context"
	"errors"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

// BatchCreateResult is the bounded aggregate produced by the shared target
// command. Organization fields are internal so REST can retain its response
// shape while MCP can expose a deliberate summary projection.
type BatchCreateResult struct {
	CreatedCount         int
	FailedCount          int
	FailedTargets        []FailedTarget
	Message              string
	NoValidTargets       bool
	OrganizationID       int
	AssociationCompleted bool
}

type FailedTarget struct {
	Name   string
	Reason string
}

// BatchCreateTargets preserves the historical REST-friendly aggregate wrapper.
// Context-aware callers should use BatchCreateTargetsContext so cancellation
// and transaction failures remain distinguishable.
func (service *TargetCommandService) BatchCreateTargets(ctx context.Context, names []string, organizationID *int) *BatchCreateResult {
	result, err := service.BatchCreateTargetsContext(ctx, names, organizationID)
	if err == nil {
		return result
	}
	return batchCreateErrorResult(names, result, err)
}

// BatchCreateTargetsContext canonicalizes and persists a target batch. When the
// production context-aware ports and coordinator are configured, organization
// validation, target/policy writes, lookup, and relationship inserts share one
// cancellable transaction.
func (service *TargetCommandService) BatchCreateTargetsContext(ctx context.Context, names []string, organizationID *int) (*BatchCreateResult, error) {
	if ctx == nil {
		return nil, context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	failedTargets := make([]FailedTarget, 0)
	validTargets := make([]catalogdomain.Target, 0, len(names))
	validNames := make([]string, 0, len(names))
	seen := make(map[string]bool)

	for _, rawName := range names {
		if catalogdomain.NormalizeBatchTargetName(rawName) == "" {
			continue
		}

		target, err := catalogdomain.BuildBatchTarget(rawName)
		if err != nil {
			failedTargets = append(failedTargets, FailedTarget{
				Name:   catalogdomain.NormalizeBatchTargetName(rawName),
				Reason: "unrecognized target format",
			})
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
			CreatedCount:   0,
			FailedCount:    len(failedTargets),
			FailedTargets:  failedTargets,
			Message:        "no valid targets",
			NoValidTargets: true,
		}, nil
	}

	result := &BatchCreateResult{
		FailedCount:   len(failedTargets),
		FailedTargets: failedTargets,
	}
	if organizationID != nil {
		result.OrganizationID = *organizationID
	}

	var createdCount int
	execute := func(txContext context.Context) error {
		if err := txContext.Err(); err != nil {
			return err
		}
		if organizationID != nil {
			if service.organizationRef == nil {
				return ErrTargetOrgNotFound
			}
			var (
				exists bool
				err    error
			)
			if service.organizationContext != nil {
				exists, err = service.organizationContext.ExistsByIDContext(txContext, *organizationID)
			} else {
				exists, err = service.organizationRef.ExistsByID(*organizationID)
			}
			if err != nil {
				return err
			}
			if !exists {
				return ErrTargetOrgNotFound
			}
		}

		var err error
		if service.batchStore != nil {
			createdCount, err = service.batchStore.BatchCreateIgnoreConflictsContext(txContext, validTargets)
		} else {
			createdCount, err = service.store.BatchCreateIgnoreConflicts(validTargets)
		}
		if err != nil {
			return err
		}

		if organizationID == nil {
			return txContext.Err()
		}

		var targets []catalogdomain.Target
		if service.batchStore != nil {
			targets, err = service.batchStore.FindByNamesContext(txContext, validNames)
		} else {
			targets, err = service.store.FindByNames(validNames)
		}
		if err != nil {
			return err
		}
		if len(targets) != len(validNames) {
			return ErrTargetNotFound
		}
		targetIDs := make([]int, 0, len(targets))
		for _, target := range targets {
			targetIDs = append(targetIDs, target.ID)
		}
		if service.organizationContext != nil {
			err = service.organizationContext.BatchAddTargetsContext(txContext, *organizationID, targetIDs)
		} else {
			err = service.organizationRef.BatchAddTargets(*organizationID, targetIDs)
		}
		if err != nil {
			return ErrTargetOrgBindingFail
		}
		result.AssociationCompleted = true
		return txContext.Err()
	}

	var err error
	if service.transactionCoordinator != nil {
		err = service.transactionCoordinator.WithinTransaction(ctx, execute)
	} else if service.batchStore != nil || service.organizationContext != nil {
		// A context-aware repository without a transaction boundary is unsafe:
		// do not silently fall back to independently committed writes.
		err = ErrBatchTransactionUnavailable
	} else {
		// Legacy unit/test doubles and non-batch callers retain the old facade
		// behavior until they opt into the context-aware ports.
		err = execute(ctx)
	}
	if err != nil {
		return nil, err
	}

	result.CreatedCount = createdCount
	result.Message = "successfully created targets"
	return result, nil
}

func batchCreateErrorResult(names []string, result *BatchCreateResult, err error) *BatchCreateResult {
	if result == nil {
		result = &BatchCreateResult{}
	}
	result.CreatedCount = 0
	result.FailedCount = len(names)
	switch {
	case errors.Is(err, ErrTargetOrgNotFound):
		result.Message = "organization not found"
	case errors.Is(err, ErrTargetOrgBindingFail):
		result.Message = "failed to associate targets with organization"
	case errors.Is(err, context.Canceled):
		result.Message = "batch create cancelled"
	case errors.Is(err, context.DeadlineExceeded):
		result.Message = "batch create deadline exceeded"
	default:
		result.Message = "batch create failed"
	}
	return result
}
