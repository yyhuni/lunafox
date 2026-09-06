package assetwiring

import (
	"context"
	"time"

	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	assetrepo "github.com/yyhuni/lunafox/server/internal/modules/asset/repository"
)

type assetStatisticsStoreAdapter struct {
	repo *assetrepo.AssetStatisticsRepository
}

func newAssetStatisticsStoreAdapter(repo *assetrepo.AssetStatisticsRepository) *assetStatisticsStoreAdapter {
	return &assetStatisticsStoreAdapter{repo: repo}
}

func (adapter *assetStatisticsStoreAdapter) GetCurrentAssetStatistics(ctx context.Context, changeSince time.Time) (assetdomain.AssetStatistics, error) {
	return adapter.repo.GetCurrentAssetStatistics(ctx, changeSince)
}

func (adapter *assetStatisticsStoreAdapter) ListAssetStatisticsHistory(ctx context.Context, start time.Time, days int) ([]assetdomain.AssetStatisticsHistoryItem, error) {
	return adapter.repo.ListAssetStatisticsHistory(ctx, start, days)
}

var _ assetapp.AssetStatisticsStore = (*assetStatisticsStoreAdapter)(nil)
