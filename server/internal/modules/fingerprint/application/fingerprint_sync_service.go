package application

import (
	"context"
	"fmt"

	"github.com/yyhuni/lunafox/server/internal/modules/fingerprint/domain"
)

// FingerprintSyncService coordinates full-file parsing and one transactional
// store synchronization. It deliberately parses all records before invoking
// the store so an invalid later record cannot produce a partial import.
type FingerprintSyncService struct {
	store FingerprintStore
}

func NewFingerprintSyncService(store FingerprintStore) *FingerprintSyncService {
	return &FingerprintSyncService{store: store}
}

func (service *FingerprintSyncService) Import(ctx context.Context, library domain.Library, contents []byte) (ImportCounts, error) {
	records, err := domain.ParseNativeImport(library, contents)
	if err != nil {
		return ImportCounts{}, err
	}
	return service.store.Upsert(ctx, library, domain.CollapseLastWins(records))
}

func (service *FingerprintSyncService) Export(ctx context.Context, library domain.Library) ([]byte, error) {
	records, err := service.store.ListAll(ctx, library)
	if err != nil {
		return nil, err
	}
	return domain.EncodeNativeExport(library, records)
}

func (service *FingerprintSyncService) Delete(ctx context.Context, library domain.Library, names []string) (int64, error) {
	if len(names) == 0 || len(names) > MaxBatchDeleteNames {
		return 0, fmt.Errorf("%w: names must contain 1 to %d values", ErrInvalidBatchDelete, MaxBatchDeleteNames)
	}
	resourceIDs := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		nameLibrary, resourceID, err := domain.ParseCanonicalName(name)
		if err != nil || nameLibrary != library {
			return 0, fmt.Errorf("%w: invalid resource name", ErrInvalidBatchDelete)
		}
		if _, exists := seen[name]; exists {
			return 0, fmt.Errorf("%w: duplicate resource name", ErrInvalidBatchDelete)
		}
		seen[name] = struct{}{}
		resourceIDs = append(resourceIDs, resourceID)
	}
	return service.store.DeleteByResourceIDs(ctx, library, resourceIDs)
}

func (service *FingerprintSyncService) Clear(ctx context.Context, library domain.Library) (int64, error) {
	if !library.IsSupported() {
		return 0, fmt.Errorf("%w: unsupported library", ErrInvalidBatchDelete)
	}
	return service.store.Clear(ctx, library)
}

func (service *FingerprintSyncService) Statistics(ctx context.Context) (LibraryStatistics, error) {
	return service.store.Statistics(ctx)
}
