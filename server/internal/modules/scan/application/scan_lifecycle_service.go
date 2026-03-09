package application

import (
	"context"
	"errors"
	"expvar"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
)

var scanDeleteStopIgnoredTotal = expvar.NewInt("scan_delete_stop_ignored_total")

type LifecycleService struct {
	scanStore      ScanCommandStore
	taskCanceller  ScanTaskCanceller
	notifier       TaskCancelNotifier
	commandService *ScanCommandService
}

func NewLifecycleService(scanStore ScanCommandStore, taskCanceller ScanTaskCanceller, notifier TaskCancelNotifier, commandService *ScanCommandService) *LifecycleService {
	return &LifecycleService{scanStore: scanStore, taskCanceller: taskCanceller, notifier: notifier, commandService: commandService}
}

func (service *LifecycleService) DeleteScan(ctx context.Context, id int) (int64, []string, error) {
	scan, err := service.scanStore.GetByIDNotDeleted(id)
	if err != nil {
		return 0, nil, err
	}
	if _, err := service.stopActiveForDelete(ctx, scan); err != nil {
		return 0, nil, err
	}
	return service.scanStore.BulkSoftDelete([]int{id})
}

func (service *LifecycleService) BulkDeleteScans(ctx context.Context, ids []int) (int64, []string, error) {
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
	return service.scanStore.BulkSoftDelete(ids)
}

func (service *LifecycleService) StopActiveScan(ctx context.Context, scan *QueryScan) (int, error) {
	if scan == nil || !isScanActive(scan.Status) {
		return 0, nil
	}
	if service.taskCanceller == nil {
		if err := service.applyStopTransition(ctx, scan.ID); err != nil {
			return 0, err
		}
		return 0, nil
	}
	cancelled, err := service.taskCanceller.CancelTasksByScanID(ctx, scan.ID)
	if err != nil {
		return 0, err
	}
	if service.notifier != nil {
		for _, info := range cancelled {
			if info.AgentID == nil {
				continue
			}
			service.notifier.SendTaskCancel(*info.AgentID, info.TaskID)
		}
	}
	if err := service.applyStopTransition(ctx, scan.ID); err != nil {
		return 0, err
	}
	return len(cancelled), nil
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

func (service *LifecycleService) applyStopTransition(ctx context.Context, scanID int) error {
	if service.commandService != nil {
		return service.commandService.StopScan(ctx, scandomain.ScanID(scanID))
	}
	return service.scanStore.UpdateStatus(scanID, string(scandomain.ScanStatusCancelled), nil)
}

func isScanActive(status string) bool {
	parsed, ok := scandomain.ParseScanStatus(status)
	return ok && scandomain.IsActiveScanStatus(parsed)
}
