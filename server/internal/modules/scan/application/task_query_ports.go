package application

import "context"

type TaskQueryStore interface {
	GetByID(ctx context.Context, id int) (*TaskRecord, error)
	PullTask(ctx context.Context, agentID int) (*TaskRecord, error)
	ListFailedByScanID(ctx context.Context, scanID int) ([]TaskRecord, error)
	GetStatusCountsByScanID(ctx context.Context, scanID int) (pending, running, completed, failed, cancelled int, err error)
	CountActiveByScanAndStage(ctx context.Context, scanID, stage int) (int, error)
}
