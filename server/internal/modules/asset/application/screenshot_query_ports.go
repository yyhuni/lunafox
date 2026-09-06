package application

import (
	"context"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

type ScreenshotQueryStore interface {
	ListByTargetID(targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Screenshot, int64, error)
	ListFilterOptionsByTargetID(targetID int, field string) ([]assetdomain.FilterOption, error)
	GetByID(id int) (*assetdomain.Screenshot, error)
}

// ScreenshotQueryStoreContext is the request-aware extension used by MCP.
type ScreenshotQueryStoreContext interface {
	ListByTargetIDContext(context.Context, int, int, int, string, string) ([]assetdomain.Screenshot, int64, error)
	GetByIDContext(context.Context, int) (*assetdomain.Screenshot, error)
}
