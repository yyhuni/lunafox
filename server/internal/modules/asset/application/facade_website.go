package application

import (
	"context"
	"database/sql"
	"errors"

	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

type WebsiteFacade struct {
	queryService *WebsiteQueryService
	cmdService   *WebsiteCommandService
}

func NewWebsiteFacade(store WebsiteStore, targetLookup WebsiteTargetLookup) *WebsiteFacade {
	return &WebsiteFacade{
		queryService: NewWebsiteQueryService(store, targetLookup),
		cmdService:   NewWebsiteCommandService(store, targetLookup),
	}
}

func (service *WebsiteFacade) ListByTarget(targetID, page, pageSize int, filter string) ([]Website, int64, error) {
	items, total, err := service.queryService.ListByTarget(context.Background(), targetID, page, pageSize, filter)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (service *WebsiteFacade) BulkCreate(targetID int, urls []string) (int, error) {
	count, err := service.cmdService.BulkCreate(context.Background(), targetID, urls)
	if err != nil {
		if errors.Is(err, ErrTargetNotFound) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}
	return count, nil
}

func (service *WebsiteFacade) Delete(id int) error {
	err := service.cmdService.Delete(context.Background(), id)
	if err != nil {
		if errors.Is(err, ErrWebsiteNotFound) || dberrors.IsRecordNotFound(err) {
			return ErrWebsiteNotFound
		}
		return err
	}
	return nil
}

func (service *WebsiteFacade) BulkDelete(ids []int) (int64, error) {
	return service.cmdService.BulkDelete(context.Background(), ids)
}

func (service *WebsiteFacade) StreamByTarget(targetID int) (*sql.Rows, error) {
	rows, err := service.queryService.StreamByTarget(context.Background(), targetID)
	if err != nil {
		if errors.Is(err, ErrTargetNotFound) || dberrors.IsRecordNotFound(err) {
			return nil, ErrTargetNotFound
		}
		return nil, err
	}
	return rows, nil
}

func (service *WebsiteFacade) CountByTarget(targetID int) (int64, error) {
	count, err := service.queryService.CountByTarget(context.Background(), targetID)
	if err != nil {
		if errors.Is(err, ErrTargetNotFound) || dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}
	return count, nil
}

func (service *WebsiteFacade) ScanRow(rows *sql.Rows) (*Website, error) {
	item, err := service.queryService.ScanRow(context.Background(), rows)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (service *WebsiteFacade) BulkUpsert(targetID int, items []WebsiteUpsertItem) (int64, error) {
	count, err := service.cmdService.BulkUpsert(context.Background(), targetID, items)
	if err != nil {
		if errors.Is(err, ErrTargetNotFound) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}
	return count, nil
}
