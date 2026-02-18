package application

import "context"

type CancelledTaskInfo struct {
	TaskID  int
	AgentID *int
}

type ScanTaskCanceller interface {
	CancelTasksByScanID(ctx context.Context, scanID int) ([]CancelledTaskInfo, error)
}

type TaskCancelNotifier interface {
	SendTaskCancel(agentID, taskID int)
}
