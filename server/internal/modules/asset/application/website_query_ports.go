package application

import (
	"context"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

type WebsiteQueryStore interface {
	ListByTargetID(targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Website, int64, error)
	GetByID(id int) (*assetdomain.Website, error)
	ListFilterOptionsByTargetID(targetID int, field string) ([]assetdomain.FilterOption, error)
	ForEachByTargetID(ctx context.Context, targetID int, visit func(assetdomain.Website) error) error
	CountByTargetID(targetID int) (int64, error)
}

type WebsiteScreenshotQueryStore interface {
	ListSummariesByTargetAndURLs(targetID int, urls []string) ([]assetdomain.Screenshot, error)
}

// Keep the Website read projection explicit without widening Screenshot's service port.
type WebsiteScreenshotStore interface {
	ScreenshotStore
	WebsiteScreenshotQueryStore
}
