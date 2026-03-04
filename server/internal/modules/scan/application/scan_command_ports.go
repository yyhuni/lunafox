package application

type ScanCommandStore interface {
	GetByIDNotDeleted(id int) (*QueryScan, error)
	FindByIDs(ids []int) ([]QueryScan, error)
	CreateWithTasks(scan *CreateScan, tasks []CreateScanTask) error
	BulkSoftDelete(ids []int) (int64, []string, error)
	UpdateStatus(id int, status string, errorMessage ...string) error
}

type ScanCreateCommandStore interface {
	CreateWithTasks(scan *CreateScan, tasks []CreateScanTask) error
}
