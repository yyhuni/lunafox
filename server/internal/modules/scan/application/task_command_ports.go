package application

import "context"

type TaskCommandStore interface {
	UpdateStatus(ctx context.Context, id int, status string, failure *FailureDetail) error
	FailPulledTask(ctx context.Context, id int, failure *FailureDetail) error
	UnlockNextStage(ctx context.Context, scanID, stage int) (int64, error)
}
