package application

import (
	"context"
	"database/sql"
	"errors"

	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

type HostPortFacade struct {
	queryService *HostPortQueryService
	cmdService   *HostPortCommandService
}

func NewHostPortFacade(store HostPortStore, targetLookup HostPortTargetLookup) *HostPortFacade {
	return &HostPortFacade{
		queryService: NewHostPortQueryService(store, targetLookup),
		cmdService:   NewHostPortCommandService(store, targetLookup),
	}
}

func (service *HostPortFacade) ListByTarget(targetID, page, pageSize int, filter string) ([]HostPortResponse, int64, error) {
	items, total, err := service.queryService.ListByTarget(context.Background(), targetID, page, pageSize, filter)
	if err != nil {
		if errors.Is(err, ErrTargetNotFound) || dberrors.IsRecordNotFound(err) {
			return nil, 0, ErrTargetNotFound
		}
		return nil, 0, err
	}
	return items, total, nil
}

func (service *HostPortFacade) StreamByTarget(targetID int) (*sql.Rows, error) {
	rows, err := service.queryService.StreamByTarget(context.Background(), targetID)
	if err != nil {
		if errors.Is(err, ErrTargetNotFound) || dberrors.IsRecordNotFound(err) {
			return nil, ErrTargetNotFound
		}
		return nil, err
	}
	return rows, nil
}

func (service *HostPortFacade) StreamByTargetAndIPs(targetID int, ips []string) (*sql.Rows, error) {
	rows, err := service.queryService.StreamByTargetAndIPs(context.Background(), targetID, ips)
	if err != nil {
		if errors.Is(err, ErrTargetNotFound) || dberrors.IsRecordNotFound(err) {
			return nil, ErrTargetNotFound
		}
		return nil, err
	}
	return rows, nil
}

func (service *HostPortFacade) CountByTarget(targetID int) (int64, error) {
	count, err := service.queryService.CountByTarget(context.Background(), targetID)
	if err != nil {
		if errors.Is(err, ErrTargetNotFound) || dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}
	return count, nil
}

func (service *HostPortFacade) ScanRow(rows *sql.Rows) (*HostPort, error) {
	item, err := service.queryService.ScanRow(context.Background(), rows)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (service *HostPortFacade) BulkUpsert(targetID int, items []HostPortItem) (int64, error) {
	count, err := service.cmdService.BulkUpsert(context.Background(), targetID, items)
	if err != nil {
		if errors.Is(err, ErrTargetNotFound) || dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}
	return count, nil
}

func (service *HostPortFacade) BulkDeleteByIPs(ips []string) (int64, error) {
	return service.cmdService.BulkDeleteByIPs(context.Background(), ips)
}
