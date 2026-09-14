package bootstrap

import (
	"context"
	"fmt"

	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	upgradeapp "github.com/yyhuni/lunafox/server/internal/modules/upgrade/application"
)

type upgradeSchedulerPauser interface {
	Pause()
}

// upgradePreDispatchCoordinator pauses scheduled work and drains the active
// Scan set through the existing atomic stop lifecycle before host handoff.
type upgradePreDispatchCoordinator struct {
	scans     *scanapp.ScanFacade
	scheduler upgradeSchedulerPauser
}

func newUpgradePreDispatchCoordinator(scans *scanapp.ScanFacade, scheduler upgradeSchedulerPauser) *upgradePreDispatchCoordinator {
	return &upgradePreDispatchCoordinator{scans: scans, scheduler: scheduler}
}

func (coordinator *upgradePreDispatchCoordinator) Prepare(ctx context.Context) (upgradeapp.PreparationResult, error) {
	return coordinator.prepareLegacy(ctx)
}

// PrepareForUpgrade is the production path. It pauses the scheduler first,
// then asks the Scan lifecycle to cancel the entire active set in one database
// transaction bound to the Upgrade Operation identity.
func (coordinator *upgradePreDispatchCoordinator) PrepareForUpgrade(ctx context.Context, operationID string) (upgradeapp.PreparationResult, error) {
	if coordinator == nil || coordinator.scans == nil {
		return upgradeapp.PreparationResult{}, fmt.Errorf("upgrade scan coordinator is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if coordinator.scheduler != nil {
		coordinator.scheduler.Pause()
	}
	outcome, err := coordinator.scans.StopAllActiveForUpgrade(ctx, operationID)
	if err != nil {
		return upgradeapp.PreparationResult{}, fmt.Errorf("cancel all active scans for upgrade: %w", err)
	}
	if outcome == nil {
		return upgradeapp.PreparationResult{}, fmt.Errorf("cancel all active scans for upgrade returned no outcome")
	}
	return upgradeapp.PreparationResult{
		CancelledScanCount: outcome.CancelledScanCount,
		CancelledTaskCount: outcome.CancelledTaskCount,
	}, nil
}

// prepareLegacy retains the bounded request-stop behavior for older callers
// that do not provide an operation identity. New upgrade creation always uses
// PrepareForUpgrade above.
func (coordinator *upgradePreDispatchCoordinator) prepareLegacy(ctx context.Context) (upgradeapp.PreparationResult, error) {
	if coordinator == nil || coordinator.scans == nil {
		return upgradeapp.PreparationResult{}, fmt.Errorf("upgrade scan coordinator is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if coordinator.scheduler != nil {
		coordinator.scheduler.Pause()
	}
	result := upgradeapp.PreparationResult{}
	for _, status := range []string{"pending", "running"} {
		for {
			scans, _, err := coordinator.scans.ListContext(ctx, scanapp.ScanListFilter{
				Page: 1, PageSize: 100, Status: status, OrderBy: "createdAt asc",
			})
			if err != nil {
				return result, fmt.Errorf("list active %s scans: %w", status, err)
			}
			if len(scans) == 0 {
				break
			}
			ids := make([]int, 0, len(scans))
			for _, scan := range scans {
				if scan.ID > 0 {
					ids = append(ids, scan.ID)
				}
			}
			if len(ids) == 0 {
				break
			}
			outcome, err := coordinator.scans.BatchStop(ctx, ids)
			if err != nil {
				return result, fmt.Errorf("cancel active %s scans: %w", status, err)
			}
			if outcome == nil {
				return result, fmt.Errorf("cancel active %s scans returned no outcome", status)
			}
			result.CancelledScanCount += outcome.StoppedCount
			result.CancelledTaskCount += outcome.RevokedTaskCount
			if outcome.StoppedCount == 0 {
				// A non-empty active page with no committed progress means the
				// underlying lifecycle state is inconsistent (or another writer
				// changed it between reads). Retrying forever would leave the
				// Upgrade Operation stuck in stopping and never reach the host
				// handoff.
				return result, fmt.Errorf("cancel active %s scans made no progress", status)
			}
		}
	}
	return result, nil
}

var _ upgradeapp.PreDispatchCoordinator = (*upgradePreDispatchCoordinator)(nil)
