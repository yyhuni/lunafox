package application

import (
	"context"
	"database/sql"
	"errors"

	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

type EndpointFacade struct {
	queryService *EndpointQueryService
	cmdService   *EndpointCommandService
}

func NewEndpointFacade(store EndpointStore, targetLookup EndpointTargetLookup) *EndpointFacade {
	return &EndpointFacade{
		queryService: NewEndpointQueryService(store, targetLookup),
		cmdService:   NewEndpointCommandService(store, targetLookup),
	}
}

func (service *EndpointFacade) ListByTarget(targetID, page, pageSize int, filter string) ([]Endpoint, int64, error) {
	items, total, err := service.queryService.ListByTarget(context.Background(), targetID, page, pageSize, filter)
	if err != nil {
		if errors.Is(err, ErrTargetNotFound) || dberrors.IsRecordNotFound(err) {
			return nil, 0, ErrTargetNotFound
		}
		return nil, 0, err
	}
	return items, total, nil
}

func (service *EndpointFacade) GetByID(id int) (*Endpoint, error) {
	item, err := service.queryService.GetByID(context.Background(), id)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrEndpointNotFound
		}
		return nil, err
	}
	return item, nil
}

func (service *EndpointFacade) BulkCreate(targetID int, urls []string) (int, error) {
	count, err := service.cmdService.BulkCreate(context.Background(), targetID, urls)
	if err != nil {
		if errors.Is(err, ErrTargetNotFound) || dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}
	return count, nil
}

func (service *EndpointFacade) Delete(id int) error {
	err := service.cmdService.Delete(context.Background(), id)
	if err != nil {
		if errors.Is(err, ErrEndpointNotFound) || dberrors.IsRecordNotFound(err) {
			return ErrEndpointNotFound
		}
		return err
	}
	return nil
}

func (service *EndpointFacade) BulkDelete(ids []int) (int64, error) {
	return service.cmdService.BulkDelete(context.Background(), ids)
}

func (service *EndpointFacade) StreamByTarget(targetID int) (*sql.Rows, error) {
	rows, err := service.queryService.StreamByTarget(context.Background(), targetID)
	if err != nil {
		if errors.Is(err, ErrTargetNotFound) || dberrors.IsRecordNotFound(err) {
			return nil, ErrTargetNotFound
		}
		return nil, err
	}
	return rows, nil
}

func (service *EndpointFacade) CountByTarget(targetID int) (int64, error) {
	count, err := service.queryService.CountByTarget(context.Background(), targetID)
	if err != nil {
		if errors.Is(err, ErrTargetNotFound) || dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}
	return count, nil
}

func (service *EndpointFacade) ScanRow(rows *sql.Rows) (*Endpoint, error) {
	item, err := service.queryService.ScanRow(context.Background(), rows)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (service *EndpointFacade) BulkUpsert(targetID int, items []EndpointUpsertItem) (int64, error) {
	affected, err := service.cmdService.BulkUpsert(context.Background(), targetID, items)
	if err != nil {
		if errors.Is(err, ErrTargetNotFound) || dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}
	return affected, nil
}
