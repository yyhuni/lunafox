package application

import (
	"context"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

type DirectoryUpsertItem struct {
	URL           string
	Status        *int
	ContentLength *int
	ContentType   string
	Duration      *int
}

type DirectoryCommandService struct {
	store        DirectoryCommandStore
	targetLookup DirectoryTargetLookup
}

func NewDirectoryCommandService(store DirectoryCommandStore, targetLookup DirectoryTargetLookup) *DirectoryCommandService {
	return &DirectoryCommandService{store: store, targetLookup: targetLookup}
}

func (service *DirectoryCommandService) BulkCreate(ctx context.Context, targetID int, urls []string) (int, error) {
	_ = ctx

	target, err := service.targetLookup.GetActiveByID(targetID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}

	directories := make([]assetdomain.Directory, 0, len(urls))
	for _, rawURL := range urls {
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

	return service.store.BulkCreate(directories)
}

func (service *DirectoryCommandService) BulkDelete(ctx context.Context, ids []int) (int64, error) {
	_ = ctx

	if len(ids) == 0 {
		return 0, nil
	}

	return service.store.BulkDelete(ids)
}

func (service *DirectoryCommandService) BulkUpsert(ctx context.Context, targetID int, items []DirectoryUpsertItem) (int64, error) {
	_ = ctx

	target, err := service.targetLookup.GetActiveByID(targetID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, ErrTargetNotFound
		}
		return 0, err
	}

	directories := make([]assetdomain.Directory, 0, len(items))
	for _, item := range items {
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

	return service.store.BulkUpsert(directories)
}
