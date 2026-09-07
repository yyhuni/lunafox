package application

import (
	"context"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
)

type HostPortSnapshotQueryStore interface {
	GetIPAggregation(scanID int, page, pageSize int, filter, orderBy string) ([]snapshotdomain.HostPortIPAggregationRow, int64, error)
	GetHostsAndPortsByIP(scanID int, ip string, filter string) ([]string, []int, error)
	ListPortOptionsByScanID(scanID int) ([]snapshotdomain.FilterOption, error)
	ForEachByScanID(ctx context.Context, scanID int, visit func(snapshotdomain.HostPortSnapshot) error) error
	CountByScanID(scanID int) (int64, error)
}
