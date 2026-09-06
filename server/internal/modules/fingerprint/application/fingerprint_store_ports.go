package application

import (
	"context"

	"github.com/yyhuni/lunafox/server/internal/modules/fingerprint/domain"
)

// FingerprintStore is the application boundary for the sole FingerprintHub
// corpus. The library argument preserves closed-resource validation.
type FingerprintStore interface {
	Upsert(ctx context.Context, library domain.Library, records []domain.ImportedRecord) (ImportCounts, error)
	List(ctx context.Context, library domain.Library, query ListStoreQuery) ([]domain.PersistedRecord, int64, error)
	ListFilterOptions(ctx context.Context, library domain.Library, field string) ([]domain.FilterOption, error)
	Get(ctx context.Context, library domain.Library, resourceID string) (*domain.PersistedRecord, error)
	ListAll(ctx context.Context, library domain.Library) ([]domain.PersistedRecord, error)
	DeleteByResourceIDs(ctx context.Context, library domain.Library, resourceIDs []string) (int64, error)
	Clear(ctx context.Context, library domain.Library) (int64, error)
	Statistics(ctx context.Context) (LibraryStatistics, error)
}

// ImportCounts are emitted only after the store commits the whole import.
type ImportCounts struct {
	CreatedCount   int
	UpdatedCount   int
	UnchangedCount int
}

// ListStoreQuery is the repository-facing normalized collection query.
type ListStoreQuery struct {
	Page     int
	PageSize int
	Filter   string
	OrderBy  string
}

// LibraryStatistics is a single read model for the fingerprint tab rail.
type LibraryStatistics struct {
	FingerPrintHub int64
}
