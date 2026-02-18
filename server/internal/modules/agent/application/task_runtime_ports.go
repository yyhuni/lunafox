package application

import (
	"context"

	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

// ScanTaskRuntimePort describes the scan task runtime dependency for agent task service.
type ScanTaskRuntimePort interface {
	PullTask(ctx context.Context, agentID int) (*scanapp.TaskAssignment, error)
	UpdateStatus(ctx context.Context, agentID, taskID int, status, errorMessage string) error
}
