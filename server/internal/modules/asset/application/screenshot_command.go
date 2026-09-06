package application

import (
	"context"
	"fmt"

	contractresults "github.com/yyhuni/lunafox/contracts/results"
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

type ScreenshotItem struct {
	URL        string
	StatusCode *int16
	Image      []byte
}

type BatchUpsertScreenshotRequest struct {
	Screenshots []ScreenshotItem
}

type ScreenshotCommandService struct {
	store        ScreenshotCommandStore
	targetLookup ScreenshotTargetLookup
}

func NewScreenshotCommandService(store ScreenshotCommandStore, targetLookup ScreenshotTargetLookup) *ScreenshotCommandService {
	return &ScreenshotCommandService{store: store, targetLookup: targetLookup}
}

func (service *ScreenshotCommandService) BatchDelete(ctx context.Context, ids []int) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	return service.store.BatchDelete(ids)
}

func (service *ScreenshotCommandService) BatchUpsert(ctx context.Context, targetID int, req *BatchUpsertScreenshotRequest) (int64, error) {
	_ = ctx

	target, err := service.targetLookup.GetActiveByID(targetID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}

	screenshots := make([]assetdomain.Screenshot, 0, len(req.Screenshots))
	for index, item := range req.Screenshots {
		if _, err := contractresults.ValidateObservedAssetURL(item.URL); err != nil {
			return 0, fmt.Errorf("screenshots[%d]: %w", index, err)
		}
		if _, err := contractresults.DeriveObservedAssetURLHost(item.URL); err != nil {
			return 0, fmt.Errorf("screenshots[%d]: %w", index, err)
		}
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

	if contextual, ok := service.store.(interface {
		BatchUpsertContext(context.Context, []assetdomain.Screenshot) (int64, error)
	}); ok {
		return contextual.BatchUpsertContext(ctx, screenshots)
	}
	return service.store.BatchUpsert(screenshots)
}
