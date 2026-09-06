package application

import (
	"context"
	"fmt"
	"time"

	cleanupdomain "github.com/yyhuni/lunafox/server/internal/modules/targetcleanup/domain"
)

type TargetCleanupRunOptions struct {
	AssetBatchSize  int
	MaxAssetBatches int
	MaxRunDuration  time.Duration
}

type TargetCleanupReconciliationResult struct {
	Completed    bool
	Deferred     bool
	AssetBatches int
	Counts       cleanupdomain.CleanupCounts
}

type TargetCleanupReconciliationService struct {
	store     TargetCleanupDataStore
	schedules TargetScheduleCleaner
	scans     TargetScanCanceller
	publisher TaskCancelPublisher
	now       func() time.Time
}

func NewTargetCleanupReconciliationService(
	store TargetCleanupDataStore,
	schedules TargetScheduleCleaner,
	scans TargetScanCanceller,
	publisher TaskCancelPublisher,
) (*TargetCleanupReconciliationService, error) {
	if store == nil || schedules == nil || scans == nil || publisher == nil {
		return nil, fmt.Errorf("target cleanup reconciliation dependencies are required")
	}
	return &TargetCleanupReconciliationService{
		store: store, schedules: schedules, scans: scans, publisher: publisher, now: time.Now,
	}, nil
}

func (service *TargetCleanupReconciliationService) WithClock(now func() time.Time) *TargetCleanupReconciliationService {
	if service != nil && now != nil {
		service.now = now
	}
	return service
}

// Reconcile always starts from the first database condition. Committed work is
// intentionally revisited as an idempotent no-op instead of inferred from a
// persisted phase or cursor after a process crash.
func (service *TargetCleanupReconciliationService) Reconcile(
	ctx context.Context,
	job cleanupdomain.CleanupJob,
	options TargetCleanupRunOptions,
) (TargetCleanupReconciliationResult, error) {
	if service == nil || service.store == nil || service.schedules == nil || service.scans == nil || service.publisher == nil {
		return TargetCleanupReconciliationResult{}, fmt.Errorf("target cleanup reconciliation is not configured")
	}
	if job.ID <= 0 || job.TargetID <= 0 {
		return TargetCleanupReconciliationResult{}, fmt.Errorf("target cleanup job scope is invalid")
	}
	if err := validateTargetCleanupRunOptions(options); err != nil {
		return TargetCleanupReconciliationResult{}, err
	}

	startedAt := service.now().UTC()
	result := TargetCleanupReconciliationResult{}
	if err := service.store.ConfirmTombstone(ctx, job.TargetID); err != nil {
		return result, err
	}
	deletedSchedules, err := service.schedules.DeleteTargetScopedSchedules(ctx, job.TargetID)
	if err != nil {
		return result, err
	}
	result.Counts.Schedules = deletedSchedules
	relationships, policies, err := service.store.DeleteTargetControlPlane(ctx, job.TargetID)
	if err != nil {
		return result, err
	}
	result.Counts.OrganizationRelations = relationships
	result.Counts.TargetPolicies = policies

	for {
		cancellation, err := service.scans.CancelNextActiveScan(ctx, job.TargetID, service.now().UTC())
		if err != nil {
			return result, err
		}
		if cancellation == nil {
			break
		}
		result.Counts.Scans++
		result.Counts.Tasks += cancellation.CancelledTaskCount
		for _, notification := range cancellation.NotificationCandidates {
			result.Counts.AgentNotifications++
			if !service.publisher.TrySendTaskCancel(notification.AgentID, cancellation.ScanID, notification.TaskID) {
				result.Counts.AgentNotificationFails++
			}
		}
	}

	for _, resource := range cleanupdomain.OrderedAssetResources() {
		for {
			if targetCleanupAssetBudgetReached(startedAt, service.now().UTC(), result.AssetBatches, options) {
				result.Deferred = true
				return result, nil
			}
			deleted, err := service.store.DeleteCurrentAssetBatch(ctx, job.TargetID, resource, options.AssetBatchSize)
			if err != nil {
				return result, err
			}
			if deleted == 0 {
				break
			}
			result.AssetBatches++
			result.Counts.AddAssetRows(resource, deleted)
		}
	}

	completed, err := service.store.MarkCompletedIfClear(ctx, job.ID, job.TargetID, service.now().UTC())
	if err != nil {
		return result, err
	}
	result.Completed = completed
	result.Deferred = !completed
	return result, nil
}

func validateTargetCleanupRunOptions(options TargetCleanupRunOptions) error {
	if options.AssetBatchSize <= 0 {
		return fmt.Errorf("target cleanup asset batch size must be positive")
	}
	if options.MaxAssetBatches <= 0 {
		return fmt.Errorf("target cleanup maximum batches must be positive")
	}
	if options.MaxRunDuration <= 0 {
		return fmt.Errorf("target cleanup maximum run duration must be positive")
	}
	return nil
}

func targetCleanupAssetBudgetReached(startedAt, now time.Time, batches int, options TargetCleanupRunOptions) bool {
	return batches >= options.MaxAssetBatches || !now.Before(startedAt.Add(options.MaxRunDuration))
}
