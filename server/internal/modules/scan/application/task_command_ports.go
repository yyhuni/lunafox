package application

import "context"

type TaskCommandStore interface {
	UpdateStatus(ctx context.Context, id int, status string, errorMessage string) error
	UnlockNextStage(ctx context.Context, scanID, stage int) (int64, error)
}
