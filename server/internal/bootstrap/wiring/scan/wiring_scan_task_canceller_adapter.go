package scanwiring

import (
	"context"

	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
)

type scanTaskCancellerAdapter struct{ repo scanrepo.ScanTaskRepository }

func newScanTaskCancellerAdapter(repo scanrepo.ScanTaskRepository) *scanTaskCancellerAdapter {
	return &scanTaskCancellerAdapter{repo: repo}
}

func (adapter *scanTaskCancellerAdapter) CancelTasksByScanID(ctx context.Context, scanID int) ([]scanapp.CancelledTaskInfo, error) {
	items, err := adapter.repo.CancelTasksByScanID(ctx, scanID)
	if err != nil {
		return nil, err
	}
	results := make([]scanapp.CancelledTaskInfo, 0, len(items))
	for index := range items {
		results = append(results, scanapp.CancelledTaskInfo{TaskID: items[index].TaskID, AgentID: items[index].AgentID})
	}
	return results, nil
}
