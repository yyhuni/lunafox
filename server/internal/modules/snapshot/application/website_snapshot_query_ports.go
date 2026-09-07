package application

import (
	"context"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
)

type WebsiteSnapshotQueryStore interface {
	ListByScanID(scanID int, page, pageSize int, filter, orderBy string) ([]snapshotdomain.WebsiteSnapshot, int64, error)
	ListFilterOptionsByScanID(scanID int, field string) ([]snapshotdomain.FilterOption, error)
	ForEachByScanID(ctx context.Context, scanID int, visit func(snapshotdomain.WebsiteSnapshot) error) error
	CountByScanID(scanID int) (int64, error)
}
