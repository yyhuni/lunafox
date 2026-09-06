package scanwiring

import (
	"context"
	"time"

	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
)

type scanStopStoreAdapter struct {
	repo *scanrepo.ScanRepository
}

func newScanStopStoreAdapter(repo *scanrepo.ScanRepository) *scanStopStoreAdapter {
	return &scanStopStoreAdapter{repo: repo}
}

func (adapter *scanStopStoreAdapter) StopActiveScan(ctx context.Context, scanID int, stoppedAt time.Time) (*scanapp.ScanStopOutcome, error) {
	outcome, err := adapter.repo.StopActiveScan(ctx, scanID, stoppedAt)
	if err != nil || outcome == nil {
		return nil, err
	}
	result := &scanapp.ScanStopOutcome{
		ScanID:             outcome.ScanID,
		CancelledTaskCount: outcome.CancelledTaskCount,
	}
	for _, candidate := range outcome.NotificationCandidates {
		result.NotificationCandidates = append(result.NotificationCandidates, scanapp.ScanStopNotification{
			TaskID:  candidate.TaskID,
			AgentID: candidate.AgentID,
		})
	}
	return result, nil
}

func (adapter *scanStopStoreAdapter) BatchStopActiveScans(ctx context.Context, scanIDs []int, stoppedAt time.Time) (*scanapp.BatchScanStopOutcome, error) {
	outcome, err := adapter.repo.BatchStopActiveScans(ctx, scanIDs, stoppedAt)
	if err != nil || outcome == nil {
		return nil, err
	}
	result := &scanapp.BatchScanStopOutcome{
		StoppedCount:     outcome.StoppedCount,
		SkippedCount:     outcome.SkippedCount,
		RevokedTaskCount: outcome.RevokedTaskCount,
	}
	for _, candidate := range outcome.NotificationCandidates {
		result.NotificationCandidates = append(result.NotificationCandidates, scanapp.BatchScanStopNotification{
			ScanID:  candidate.ScanID,
			TaskID:  candidate.TaskID,
			AgentID: candidate.AgentID,
		})
	}
	return result, nil
}
