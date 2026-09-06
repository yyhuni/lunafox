package application

import (
	"context"
	"time"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

// AssetStatisticsStore reads the current-state inventory projections that power Overview.
type AssetStatisticsStore interface {
	GetCurrentAssetStatistics(ctx context.Context, changeSince time.Time) (assetdomain.AssetStatistics, error)
	ListAssetStatisticsHistory(ctx context.Context, start time.Time, days int) ([]assetdomain.AssetStatisticsHistoryItem, error)
}
