package application

import (
	"context"
	"errors"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

var (
	ErrScreenshotNotFound = errors.New("screenshot not found")
)

type ScreenshotQueryService struct {
	store        ScreenshotQueryStore
	targetLookup ScreenshotTargetLookup
}

func NewScreenshotQueryService(store ScreenshotQueryStore, targetLookup ScreenshotTargetLookup) *ScreenshotQueryService {
	return &ScreenshotQueryService{store: store, targetLookup: targetLookup}
}

func (service *ScreenshotQueryService) ListByTargetID(ctx context.Context, targetID, page, pageSize int, filter string) ([]assetdomain.Screenshot, int64, error) {
	_ = ctx

	if _, err := service.targetLookup.GetActiveByID(targetID); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, 0, ErrTargetNotFound
		}
		return nil, 0, err
	}

	return service.store.FindByTargetID(targetID, page, pageSize, filter)
}

func (service *ScreenshotQueryService) GetByID(ctx context.Context, id int) (*assetdomain.Screenshot, error) {
	_ = ctx

	item, err := service.store.GetByID(id)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrScreenshotNotFound
		}
		return nil, err
	}
	return item, nil
}
