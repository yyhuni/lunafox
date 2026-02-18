package application

import (
	"context"
	"database/sql"
	"errors"

	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

var ErrDirectoryNotFound = errors.New("directory not found")

type DirectoryFacade struct {
	queryService *DirectoryQueryService
	cmdService   *DirectoryCommandService
}

func NewDirectoryFacade(store DirectoryStore, targetLookup DirectoryTargetLookup) *DirectoryFacade {
	return &DirectoryFacade{
		queryService: NewDirectoryQueryService(store, targetLookup),
		cmdService:   NewDirectoryCommandService(store, targetLookup),
	}
}

func (service *DirectoryFacade) ListByTarget(targetID, page, pageSize int, filter string) ([]Directory, int64, error) {
	items, total, err := service.queryService.ListByTarget(context.Background(), targetID, page, pageSize, filter)
	if err != nil {
		if errors.Is(err, ErrTargetNotFound) || dberrors.IsRecordNotFound(err) {
			return nil, 0, ErrTargetNotFound
		}
		return nil, 0, err
	}
	return items, total, nil
}

func (service *DirectoryFacade) BulkCreate(targetID int, urls []string) (int, error) {
	count, err := service.cmdService.BulkCreate(context.Background(), targetID, urls)
	if err != nil {
		if errors.Is(err, ErrTargetNotFound) || dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}
	return count, nil
}

func (service *DirectoryFacade) BulkDelete(ids []int) (int64, error) {
	return service.cmdService.BulkDelete(context.Background(), ids)
}

func (service *DirectoryFacade) StreamByTarget(targetID int) (*sql.Rows, error) {
	rows, err := service.queryService.StreamByTarget(context.Background(), targetID)
	if err != nil {
		if errors.Is(err, ErrTargetNotFound) || dberrors.IsRecordNotFound(err) {
			return nil, ErrTargetNotFound
		}
		return nil, err
	}
	return rows, nil
}

func (service *DirectoryFacade) CountByTarget(targetID int) (int64, error) {
	count, err := service.queryService.CountByTarget(context.Background(), targetID)
	if err != nil {
		if errors.Is(err, ErrTargetNotFound) || dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}
	return count, nil
}

func (service *DirectoryFacade) ScanRow(rows *sql.Rows) (*Directory, error) {
	item, err := service.queryService.ScanRow(context.Background(), rows)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (service *DirectoryFacade) BulkUpsert(targetID int, items []DirectoryUpsertItem) (int64, error) {
	affected, err := service.cmdService.BulkUpsert(context.Background(), targetID, items)
	if err != nil {
		if errors.Is(err, ErrTargetNotFound) || dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}
	return affected, nil
}
