package application

import (
	"context"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
)

type SubdomainSnapshotQueryStore interface {
	ListByScanID(scanID int, page, pageSize int, filter, orderBy string) ([]snapshotdomain.SubdomainSnapshot, int64, error)
	ForEachByScanID(ctx context.Context, scanID int, visit func(snapshotdomain.SubdomainSnapshot) error) error
	CountByScanID(scanID int) (int64, error)
}
