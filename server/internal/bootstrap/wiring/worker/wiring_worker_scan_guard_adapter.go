package workerwiring

import (
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

type workerScanGuardAdapter struct {
	repo *scanrepo.ScanRepository
}

func newWorkerScanGuardAdapter(repo *scanrepo.ScanRepository) *workerScanGuardAdapter {
	return &workerScanGuardAdapter{repo: repo}
}

func (adapter *workerScanGuardAdapter) EnsureActiveByID(id int) error {
	_, err := adapter.repo.GetByIDNotDeleted(id)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return catalogapp.ErrWorkerScanNotFound
		}
		return err
	}
	return nil
}
