package scanwiring

import (
	"context"

	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
)

type scanQueryStoreAdapter struct {
	repo *scanrepo.ScanRepository
}

func newScanQueryStoreAdapter(repo *scanrepo.ScanRepository) *scanQueryStoreAdapter {
	return &scanQueryStoreAdapter{repo: repo}
}

func (adapter *scanQueryStoreAdapter) List(page, pageSize int, targetID int, status, filter, orderBy string) ([]scanapp.QueryScan, int64, error) {
	return adapter.repo.List(page, pageSize, targetID, status, filter, orderBy)
}

func (adapter *scanQueryStoreAdapter) ListContext(ctx context.Context, page, pageSize int, targetID int, status, filter, orderBy string) ([]scanapp.QueryScan, int64, error) {
	return adapter.repo.ListContext(ctx, page, pageSize, targetID, status, filter, orderBy)
}

func (adapter *scanQueryStoreAdapter) GetDetailByID(id int) (*scanapp.QueryScan, error) {
	return adapter.repo.GetDetailByID(id)
}

func (adapter *scanQueryStoreAdapter) GetDetailByIDContext(ctx context.Context, id int) (*scanapp.QueryScan, error) {
	return adapter.repo.GetDetailByIDContext(ctx, id)
}

func (adapter *scanQueryStoreAdapter) GetGlobalStatsSummary() (*scanapp.QueryStatistics, error) {
	return adapter.repo.GetGlobalStatsSummary()
}
