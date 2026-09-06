package application

import "context"

type ScanTaskQueryStore interface {
	GetByID(ctx context.Context, id int) (*ScanTaskRecord, error)
	ListFailedByScanID(ctx context.Context, scanID int) ([]ScanTaskRecord, error)
	CountByStatusForScanID(ctx context.Context, scanID int) (pending, running, completed, failed, cancelled, skipped int, err error)
	CountActiveByScanAndStageOrder(ctx context.Context, scanID, scanWorkflowStageOrder int) (int, error)
}
