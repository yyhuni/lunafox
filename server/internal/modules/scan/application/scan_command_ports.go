package application

import "context"

type ScanCommandStore interface {
	GetLifecycleRefByID(id int) (*QueryScan, error)
	FindByIDs(ids []int) ([]QueryScan, error)
	BatchSoftDelete(ids []int) (int64, []string, error)
	UpdateScanStatus(id int, status string, failure *FailureDetail) error
}

// ScanCreateCommandStore is the only scan-create transaction boundary. The
// repository allocates identities, invokes the finalizer, and commits only
// after every executable task owns a validated plan.
type ScanCreateCommandStore interface {
	CreateWithScanTasksAndPlans(ctx context.Context, scan *CreateScan, finalize ScanCreateTaskFinalizer) error
}

type ScanApplicationCommandStore interface {
	ScanCommandStore
	ScanCreateCommandStore
}
