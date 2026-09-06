package application

import (
	"context"
	"errors"
	"expvar"
	"time"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
)

var scanDeleteStopIgnoredTotal = expvar.NewInt("scan_delete_stop_ignored_total")
var scanStopCancelDeliveryFailedTotal = expvar.NewInt("scan_stop_cancel_delivery_failed_total")

type LifecycleService struct {
	scanStore ScanCommandStore
	stopStore ScanStopStore
	notifier  TaskCancelNotifier
	clock     Clock
}

func NewLifecycleService(scanStore ScanCommandStore, stopStore ScanStopStore, notifier TaskCancelNotifier, clocks ...Clock) *LifecycleService {
	clock := Clock(systemClock{})
	if len(clocks) > 0 && clocks[0] != nil {
		clock = clocks[0]
	}
	return &LifecycleService{scanStore: scanStore, stopStore: stopStore, notifier: notifier, clock: clock}
}

func (service *LifecycleService) DeleteScan(ctx context.Context, id int) (int64, []string, error) {
	scan, err := service.scanStore.GetLifecycleRefByID(id)
	if err != nil {
		return 0, nil, err
	}
	if _, err := service.stopActiveForDelete(ctx, scan); err != nil {
		return 0, nil, err
	}
	return service.scanStore.BatchSoftDelete([]int{id})
}

func (service *LifecycleService) BatchDeleteScans(ctx context.Context, ids []int) (int64, []string, error) {
	if len(ids) == 0 {
		return 0, nil, nil
	}
	scans, err := service.scanStore.FindByIDs(ids)
	if err != nil {
		return 0, nil, err
	}
	for index := range scans {
		if _, err := service.stopActiveForDelete(ctx, &scans[index]); err != nil {
			return 0, nil, err
		}
	}
	return service.scanStore.BatchSoftDelete(ids)
}

func (service *LifecycleService) StopActiveScan(ctx context.Context, scan *QueryScan) (int, error) {
	if scan == nil || !isScanActive(scan.Status) {
		return 0, nil
	}
	return service.stopScan(ctx, scan.ID)
}

// StopScan executes the normal synchronous Stop path. It deliberately does no
// pre-read: the repository locks and revalidates current state in its one
// cancellation transaction, so a stale read cannot split Scan and Task state.
func (service *LifecycleService) StopScan(ctx context.Context, scanID int) (int, error) {
	return service.stopScan(ctx, scanID)
}

// BatchStopScans executes one request-bound transaction for the selected Scan
// resources. Terminal rows are reported as skipped by the repository rather
// than being treated as a silent success or a per-row HTTP loop.
func (service *LifecycleService) BatchStopScans(ctx context.Context, scanIDs []int) (*BatchScanStopOutcome, error) {
	if ctx == nil {
		return nil, context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if service == nil || service.stopStore == nil {
		return nil, errors.New("scan stop store not initialized")
	}
	if len(scanIDs) == 0 {
		return nil, errors.New("batch scan stop requires at least one scan")
	}
	stoppedAt := time.Now().UTC()
	if service.clock != nil {
		stoppedAt = service.clock.Now().UTC()
	}
	outcome, err := service.stopStore.BatchStopActiveScans(ctx, scanIDs, stoppedAt)
	if err != nil {
		return nil, err
	}
	if outcome == nil {
		return nil, errors.New("scan batch stop store returned no outcome")
	}
	for _, candidate := range outcome.NotificationCandidates {
		delivered := service.notifier != nil && service.notifier.TrySendTaskCancel(candidate.AgentID, candidate.ScanID, candidate.TaskID)
		if delivered {
			continue
		}
		scanStopCancelDeliveryFailedTotal.Add(1)
		pkg.Warn("scan task cancellation notification was not accepted",
			zap.Int("agent.id", candidate.AgentID),
			zap.Int("scan.id", candidate.ScanID),
			zap.Int("task.id", candidate.TaskID),
		)
	}
	return outcome, nil
}

func (service *LifecycleService) stopScan(ctx context.Context, scanID int) (int, error) {
	if ctx == nil {
		return 0, context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if service == nil || service.stopStore == nil {
		return 0, errors.New("scan stop store not initialized")
	}
	stoppedAt := time.Now().UTC()
	if service.clock != nil {
		stoppedAt = service.clock.Now().UTC()
	}
	outcome, err := service.stopStore.StopActiveScan(ctx, scanID, stoppedAt)
	if err != nil {
		return 0, err
	}
	if outcome == nil {
		return 0, errors.New("scan stop store returned no outcome")
	}
	for _, candidate := range outcome.NotificationCandidates {
		delivered := service.notifier != nil && service.notifier.TrySendTaskCancel(candidate.AgentID, outcome.ScanID, candidate.TaskID)
		if delivered {
			continue
		}
		scanStopCancelDeliveryFailedTotal.Add(1)
		pkg.Warn("scan task cancellation notification was not accepted",
			zap.Int("agent.id", candidate.AgentID),
			zap.Int("scan.id", outcome.ScanID),
			zap.Int("task.id", candidate.TaskID),
		)
	}
	return outcome.CancelledTaskCount, nil
}

func (service *LifecycleService) stopActiveForDelete(ctx context.Context, scan *QueryScan) (int, error) {
	count, err := service.StopActiveScan(ctx, scan)
	if err == nil {
		return count, nil
	}
	if !errors.Is(err, scandomain.ErrScanCannotStop) && !errors.Is(err, scandomain.ErrInvalidStatusChange) {
		return 0, err
	}
	scanDeleteStopIgnoredTotal.Add(1)
	pkg.Warn("ignoring stop error during scan deletion",
		zap.Int("scan.id", scan.ID),
		zap.String("scan.status", scan.Status),
		zap.Error(err),
	)
	return 0, nil
}

func isScanActive(status string) bool {
	parsed, ok := scandomain.ParseScanStatus(status)
	return ok && scandomain.IsActiveScanStatus(parsed)
}
