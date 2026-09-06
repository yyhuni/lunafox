package application

import (
	"context"
	"errors"
)

func (service *ScanFacade) Delete(id int) (int64, []string, error) {
	if service.lifecycleService == nil {
		return 0, nil, errors.New("scan lifecycle service not initialized")
	}
	deletedCount, deletedNames, err := service.lifecycleService.DeleteScan(context.Background(), id)
	if err != nil {
		return 0, nil, mapScanBoundaryError(err)
	}
	return deletedCount, deletedNames, nil
}

func (service *ScanFacade) BatchDelete(ids []int) (int64, []string, error) {
	if service.lifecycleService == nil {
		return 0, nil, errors.New("scan lifecycle service not initialized")
	}
	return service.lifecycleService.BatchDeleteScans(context.Background(), ids)
}

func (service *ScanFacade) HardDelete(id int) error {
	_ = id
	return ErrScanHardDeleteNotReady
}

func (service *ScanFacade) Stop(ctx context.Context, id int) (int, error) {
	if service.lifecycleService == nil {
		return 0, errors.New("scan lifecycle service not initialized")
	}
	count, err := service.lifecycleService.StopScan(ctx, id)
	if err != nil {
		return 0, mapScanStopBoundaryError(err)
	}
	return count, nil
}

func (service *ScanFacade) BatchStop(ctx context.Context, ids []int) (*BatchScanStopOutcome, error) {
	if service.lifecycleService == nil {
		return nil, errors.New("scan lifecycle service not initialized")
	}
	result, err := service.lifecycleService.BatchStopScans(ctx, ids)
	if err != nil {
		return nil, mapScanStopBoundaryError(err)
	}
	return result, nil
}
