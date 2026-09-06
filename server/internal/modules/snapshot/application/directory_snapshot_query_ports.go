package application

import (
	"context"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
)

type DirectorySnapshotQueryStore interface {
	ListByScanID(scanID int, page, pageSize int, filter, orderBy string) ([]snapshotdomain.DirectorySnapshot, int64, error)
	ListFilterOptionsByScanID(scanID int, field string) ([]snapshotdomain.FilterOption, error)
	ForEachByScanID(ctx context.Context, scanID int, visit func(snapshotdomain.DirectorySnapshot) error) error
	CountByScanID(scanID int) (int64, error)
}
