package targetcleanupwiring

import (
	"context"
	"time"

	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
	targetcleanupapp "github.com/yyhuni/lunafox/server/internal/modules/targetcleanup/application"
)

type targetCleanupScanCancellerAdapter struct {
	repo *scanrepo.ScanRepository
}

func newTargetCleanupScanCancellerAdapter(repo *scanrepo.ScanRepository) *targetCleanupScanCancellerAdapter {
	return &targetCleanupScanCancellerAdapter{repo: repo}
}

func (adapter *targetCleanupScanCancellerAdapter) CancelNextActiveScan(ctx context.Context, targetID int, now time.Time) (*targetcleanupapp.TargetScanCancellation, error) {
	cancellation, err := adapter.repo.CancelNextActiveForDeletedTarget(ctx, targetID, now)
	if err != nil || cancellation == nil {
		return nil, err
	}
	result := &targetcleanupapp.TargetScanCancellation{
		ScanID:             cancellation.ScanID,
		CancelledTaskCount: cancellation.CancelledTaskCount,
	}
	for _, candidate := range cancellation.NotificationCandidates {
		result.NotificationCandidates = append(result.NotificationCandidates, targetcleanupapp.TargetTaskCancelNotification{
			TaskID:  candidate.TaskID,
			AgentID: candidate.AgentID,
		})
	}
	return result, nil
}
