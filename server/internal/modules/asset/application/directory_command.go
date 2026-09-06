package application

import (
	"context"
	"fmt"

	contractresults "github.com/yyhuni/lunafox/contracts/results"
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

type DirectoryUpsertItem struct {
	URL           string
	Status        *int
	ContentLength *int64
	ContentType   string
	Duration      *int64
}

type DirectoryCommandService struct {
	store        DirectoryCommandStore
	targetLookup AssetCommandTargetLookup
}

func NewDirectoryCommandService(store DirectoryCommandStore, targetLookup AssetCommandTargetLookup) *DirectoryCommandService {
	return &DirectoryCommandService{store: store, targetLookup: targetLookup}
}

func (service *DirectoryCommandService) BatchCreate(ctx context.Context, targetID int, urls []string) (int, error) {
	target, err := service.targetLookup.GetActiveByIDContext(ctx, targetID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}

	directories := make([]assetdomain.Directory, 0, len(urls))
	for index, rawURL := range urls {
		if _, err := observedAssetURLHostForBatchCreate(index, rawURL); err != nil {
			return 0, err
		}
		if assetdomain.IsURLMatchTarget(rawURL, *target) {
			directories = append(directories, assetdomain.Directory{
				TargetID: targetID,
				URL:      rawURL,
			})
		}
	}

	if len(directories) == 0 {
		return 0, nil
	}

	return service.store.BatchCreate(directories)
}

func (service *DirectoryCommandService) BatchDelete(ctx context.Context, ids []int) (int64, error) {
	_ = ctx

	if len(ids) == 0 {
		return 0, nil
	}

	return service.store.BatchDelete(ids)
}

func (service *DirectoryCommandService) BatchUpsert(ctx context.Context, targetID int, items []DirectoryUpsertItem) (int64, error) {
	target, err := service.targetLookup.GetActiveByIDContext(ctx, targetID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}

	directories := make([]assetdomain.Directory, 0, len(items))
	for index, item := range items {
		if _, err := contractresults.ValidateObservedAssetURL(item.URL); err != nil {
			return 0, fmt.Errorf("items[%d]: %w", index, err)
		}
		if _, err := contractresults.DeriveObservedAssetURLHost(item.URL); err != nil {
			return 0, fmt.Errorf("items[%d]: %w", index, err)
		}
		if !assetdomain.IsURLMatchTarget(item.URL, *target) {
			continue
		}

		directories = append(directories, assetdomain.Directory{
			TargetID:      targetID,
			URL:           item.URL,
			Status:        item.Status,
			ContentLength: item.ContentLength,
			ContentType:   item.ContentType,
			Duration:      item.Duration,
		})
	}

	if len(directories) == 0 {
		return 0, nil
	}
	return service.store.BatchUpsertContext(ctx, directories)
}
