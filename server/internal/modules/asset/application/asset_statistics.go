package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

const (
	assetStatisticsMinimumHistoryDays = 1
	assetStatisticsMaximumHistoryDays = 30
)

var ErrInvalidAssetStatisticsHistoryDays = errors.New("asset statistics history days must be between 1 and 30")

// AssetStatisticsQueryService orchestrates read-only Overview statistics projections.
type AssetStatisticsQueryService struct {
	store AssetStatisticsStore
	now   func() time.Time
}

func NewAssetStatisticsQueryService(store AssetStatisticsStore, now func() time.Time) *AssetStatisticsQueryService {
	if store == nil {
		panic("asset statistics store is required")
	}
	if now == nil {
		panic("asset statistics clock is required")
	}
	return &AssetStatisticsQueryService{store: store, now: now}
}

func (service *AssetStatisticsQueryService) GetCurrent(ctx context.Context) (assetdomain.AssetStatistics, error) {
	now := service.now().UTC()
	statistics, err := service.store.GetCurrentAssetStatistics(ctx, now.Add(-24*time.Hour))
	if err != nil {
		return assetdomain.AssetStatistics{}, err
	}
	statistics.UpdatedAt = now
	return statistics, nil
}

func (service *AssetStatisticsQueryService) ListHistory(ctx context.Context, days int) ([]assetdomain.AssetStatisticsHistoryItem, error) {
	if days < assetStatisticsMinimumHistoryDays || days > assetStatisticsMaximumHistoryDays {
		return nil, fmt.Errorf("%w: %d", ErrInvalidAssetStatisticsHistoryDays, days)
	}

	now := service.now().UTC()
	start := now.Truncate(24*time.Hour).AddDate(0, 0, -(days - 1))
	return service.store.ListAssetStatisticsHistory(ctx, start, days)
}
