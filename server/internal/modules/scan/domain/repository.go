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
	GetByID(ctx context.Context, id ScanID) (*Scan, error)
	List(ctx context.Context, filter ScanFilter) ([]Scan, int64, error)
	SaveScanState(ctx context.Context, scan *Scan) error
}
