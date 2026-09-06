package resultingestwiring

import (
	"context"

	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
)

type resultIngestScanResultSummaryUpdaterAdapter struct {
	repo *scanrepo.ScanRepository
}

func newResultIngestScanResultSummaryUpdaterAdapter(repo *scanrepo.ScanRepository) *resultIngestScanResultSummaryUpdaterAdapter {
	return &resultIngestScanResultSummaryUpdaterAdapter{repo: repo}
}

func (adapter *resultIngestScanResultSummaryUpdaterAdapter) RefreshScanResultSummary(ctx context.Context, scanID int, targetID int) error {
	return adapter.repo.RefreshScanResultSummary(ctx, scanID, targetID)
}
