package application

import (
	"context"
	"errors"

	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

type ScreenshotFacade struct {
	queryService *ScreenshotQueryService
	cmdService   *ScreenshotCommandService
}

func NewScreenshotFacade(store ScreenshotStore, targetLookup ScreenshotTargetLookup) *ScreenshotFacade {
	return &ScreenshotFacade{
		queryService: NewScreenshotQueryService(store, targetLookup),
		cmdService:   NewScreenshotCommandService(store, targetLookup),
	}
}

func (service *ScreenshotFacade) ListByTargetID(targetID, page, pageSize int, filter string) ([]Screenshot, int64, error) {
	items, total, err := service.queryService.ListByTargetID(context.Background(), targetID, page, pageSize, filter)
	if err != nil {
		if errors.Is(err, ErrTargetNotFound) || dberrors.IsRecordNotFound(err) {
			return nil, 0, ErrTargetNotFound
		}
		return nil, 0, err
	}
	return items, total, nil
}

func (service *ScreenshotFacade) GetByID(id int) (*Screenshot, error) {
	item, err := service.queryService.GetByID(context.Background(), id)
	if err != nil {
		if errors.Is(err, ErrScreenshotNotFound) || dberrors.IsRecordNotFound(err) {
			return nil, ErrScreenshotNotFound
		}
		return nil, err
	}
	return item, nil
}

func (service *ScreenshotFacade) BulkDelete(ids []int) (int64, error) {
	return service.cmdService.BulkDelete(context.Background(), ids)
}

func (service *ScreenshotFacade) BulkUpsert(targetID int, req *BulkUpsertScreenshotRequest) (int64, error) {
	count, err := service.cmdService.BulkUpsert(context.Background(), targetID, req)
	if err != nil {
		if errors.Is(err, ErrTargetNotFound) || dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}
	return count, nil
}
