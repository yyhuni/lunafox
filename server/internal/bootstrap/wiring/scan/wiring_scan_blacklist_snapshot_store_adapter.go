package scanwiring

import (
	"context"
	"errors"

	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
)

// NewScanBlacklistSnapshotStoreAdapter exposes only Scan-owned immutable
// snapshot reads to the application layer.
func NewScanBlacklistSnapshotStoreAdapter(repo *scanrepo.ScanRepository) scanapp.ScanBlacklistSnapshotStore {
	if repo == nil {
		return nil
	}
	return scanBlacklistSnapshotStoreAdapter{repo: repo}
}

type scanBlacklistSnapshotStoreAdapter struct {
	repo *scanrepo.ScanRepository
}

func (adapter scanBlacklistSnapshotStoreAdapter) LoadBlacklistSnapshot(ctx context.Context, scanID int) ([]string, error) {
	patterns, err := adapter.repo.LoadBlacklistSnapshot(ctx, scanID)
	if err != nil {
		switch {
		case errors.Is(err, scanrepo.ErrScanBlacklistSnapshotNotFound):
			return nil, scanapp.ErrScanBlacklistSnapshotNotFound
		case errors.Is(err, scanrepo.ErrScanBlacklistSnapshotDataIntegrity):
			return nil, scanapp.ErrScanBlacklistSnapshotDataIntegrity
		default:
			return nil, err
		}
	}
	return append([]string{}, patterns...), nil
}

var _ scanapp.ScanBlacklistSnapshotStore = (*scanBlacklistSnapshotStoreAdapter)(nil)
