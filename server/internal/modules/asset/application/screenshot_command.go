package application

import (
	"context"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

type ScreenshotItem struct {
	URL        string
	StatusCode *int16
	Image      []byte
}

type BulkUpsertScreenshotRequest struct {
	Screenshots []ScreenshotItem
}

type ScreenshotCommandService struct {
	store        ScreenshotCommandStore
	targetLookup ScreenshotTargetLookup
}

func NewScreenshotCommandService(store ScreenshotCommandStore, targetLookup ScreenshotTargetLookup) *ScreenshotCommandService {
	return &ScreenshotCommandService{store: store, targetLookup: targetLookup}
}

func (service *ScreenshotCommandService) BulkDelete(ctx context.Context, ids []int) (int64, error) {
	_ = ctx

	if len(ids) == 0 {
		return 0, nil
	}

	return service.store.BulkDelete(ids)
}

func (service *ScreenshotCommandService) BulkUpsert(ctx context.Context, targetID int, req *BulkUpsertScreenshotRequest) (int64, error) {
	_ = ctx

	target, err := service.targetLookup.GetActiveByID(targetID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}

	screenshots := make([]assetdomain.Screenshot, 0, len(req.Screenshots))
	for _, item := range req.Screenshots {
		if assetdomain.IsURLMatchTarget(item.URL, *target) {
			screenshots = append(screenshots, assetdomain.Screenshot{
				TargetID:   targetID,
				URL:        item.URL,
				StatusCode: item.StatusCode,
				Image:      item.Image,
			})
		}
	}

	if len(screenshots) == 0 {
		return 0, nil
	}

	return service.store.BulkUpsert(screenshots)
}
