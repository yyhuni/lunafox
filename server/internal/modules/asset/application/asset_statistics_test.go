package application

import (
	"context"
	"errors"
	"testing"
	"time"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

type assetStatisticsStoreStub struct {
	current            assetdomain.AssetStatistics
	history            []assetdomain.AssetStatisticsHistoryItem
	currentErr         error
	historyErr         error
	currentChangeSince time.Time
	historyStart       time.Time
	historyDays        int
}

func (stub *assetStatisticsStoreStub) GetCurrentAssetStatistics(_ context.Context, changeSince time.Time) (assetdomain.AssetStatistics, error) {
	stub.currentChangeSince = changeSince
	return stub.current, stub.currentErr
}

func (stub *assetStatisticsStoreStub) ListAssetStatisticsHistory(_ context.Context, start time.Time, days int) ([]assetdomain.AssetStatisticsHistoryItem, error) {
	stub.historyStart = start
	stub.historyDays = days
	return stub.history, stub.historyErr
}

func TestAssetStatisticsQueryServiceGetCurrentSetsObservationTime(t *testing.T) {
	now := time.Date(2026, 8, 7, 15, 30, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	store := &assetStatisticsStoreStub{current: assetdomain.AssetStatistics{TotalAssets: 5}}
	service := NewAssetStatisticsQueryService(store, func() time.Time { return now })

	statistics, err := service.GetCurrent(context.Background())
	if err != nil {
		t.Fatalf("get current: %v", err)
	}
	if !statistics.UpdatedAt.Equal(now.UTC()) {
		t.Fatalf("updated at = %s, want %s", statistics.UpdatedAt, now.UTC())
	}
	if !store.currentChangeSince.Equal(now.UTC().Add(-24 * time.Hour)) {
		t.Fatalf("change since = %s, want %s", store.currentChangeSince, now.UTC().Add(-24*time.Hour))
	}
}

func TestAssetStatisticsQueryServiceListHistoryValidatesWindowAndAnchorsUTCDate(t *testing.T) {
	now := time.Date(2026, 8, 7, 1, 30, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	store := &assetStatisticsStoreStub{}
	service := NewAssetStatisticsQueryService(store, func() time.Time { return now })

	for _, days := range []int{0, 31} {
		if _, err := service.ListHistory(context.Background(), days); !errors.Is(err, ErrInvalidAssetStatisticsHistoryDays) {
			t.Fatalf("days=%d error = %v, want invalid history days", days, err)
		}
	}
	if store.historyDays != 0 {
		t.Fatalf("invalid request called history store with %d days", store.historyDays)
	}

	if _, err := service.ListHistory(context.Background(), 7); err != nil {
		t.Fatalf("list history: %v", err)
	}
	if store.historyDays != 7 {
		t.Fatalf("history days = %d, want 7", store.historyDays)
	}
	wantStart := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	if !store.historyStart.Equal(wantStart) {
		t.Fatalf("history start = %s, want %s", store.historyStart, wantStart)
	}
}
