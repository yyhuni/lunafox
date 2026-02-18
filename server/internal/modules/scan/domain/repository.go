package domain

import "context"

type ScanFilter struct {
	Page     int
	PageSize int
	TargetID int
	Status   ScanStatus
	Search   string
}

type ScanRepository interface {
	GetByIDNotDeleted(ctx context.Context, id ScanID) (*Scan, error)
	FindAll(ctx context.Context, filter ScanFilter) ([]Scan, int64, error)
	Save(ctx context.Context, scan *Scan) error
}

type ScanTaskRepository interface {
	FindByScanID(ctx context.Context, scanID ScanID) ([]ScanTask, error)
	SaveBatch(ctx context.Context, tasks []ScanTask) error
}
