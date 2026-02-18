package application

import (
	"context"
	"errors"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

func (service *ScanFacade) Delete(id int) (int64, []string, error) {
	if service.lifecycleService == nil {
		return 0, nil, errors.New("scan lifecycle service not initialized")
	}
	deletedCount, deletedNames, err := service.lifecycleService.DeleteScan(context.Background(), id)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, nil, ErrScanNotFound
		}
		return 0, nil, err
	}
	return deletedCount, deletedNames, nil
}

func (service *ScanFacade) BulkDelete(ids []int) (int64, []string, error) {
	if service.lifecycleService == nil {
		return 0, nil, errors.New("scan lifecycle service not initialized")
	}
	return service.lifecycleService.BulkDeleteScans(context.Background(), ids)
}

func (service *ScanFacade) HardDelete(id int) error {
	_ = id
	return ErrScanHardDeleteNotReady
}

func (service *ScanFacade) Stop(id int) (int, error) {
	if service.commandStore == nil {
		return 0, errors.New("scan command store not initialized")
	}
	scan, err := service.commandStore.GetByIDNotDeleted(id)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrScanNotFound
		}
		return 0, err
	}
	if !isScanActive(scan.Status) {
		return 0, ErrScanCannotStop
	}
	return service.stopActiveScan(scan)
}

func (service *ScanFacade) stopActiveScan(scan *QueryScan) (int, error) {
	if service.lifecycleService == nil {
		return 0, errors.New("scan lifecycle service not initialized")
	}
	count, err := service.lifecycleService.StopActiveScan(context.Background(), scan)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrScanNotFound
		}
		if errors.Is(err, scandomain.ErrScanCannotStop) || errors.Is(err, scandomain.ErrInvalidStatusChange) {
			return 0, ErrScanCannotStop
		}
		return 0, err
	}
	return count, nil
}
