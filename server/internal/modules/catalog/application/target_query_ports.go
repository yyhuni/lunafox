package application

import (
	"context"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

type TargetQueryStore interface {
	GetActiveByID(id int) (*catalogdomain.Target, error)
	List(page, pageSize int, filter, orderBy string) ([]catalogdomain.Target, int64, error)
	GetAssetCountsSummary(targetID int) (*catalogdomain.TargetAssetCounts, error)
	GetVulnerabilityCountsSummary(targetID int) (*catalogdomain.VulnerabilityCounts, error)
}

// TargetQueryStoreContext is the cancellation-aware read surface used by
// transports with a caller-owned deadline. It is optional so existing local
// test stores and non-requested background readers remain source compatible.
type TargetQueryStoreContext interface {
	ListContext(context.Context, int, int, string, string) ([]catalogdomain.Target, int64, error)
	GetActiveByIDContext(context.Context, int) (*catalogdomain.Target, error)
	GetAssetCountsSummaryContext(context.Context, int) (*catalogdomain.TargetAssetCounts, error)
	GetVulnerabilityCountsSummaryContext(context.Context, int) (*catalogdomain.VulnerabilityCounts, error)
}
