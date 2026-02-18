package application

type ScanCommandStore interface {
	GetByIDNotDeleted(id int) (*QueryScan, error)
	FindByIDs(ids []int) ([]QueryScan, error)
	CreateWithInputTargetsAndTasks(scan *CreateScan, inputs []CreateScanInputTarget, tasks []CreateScanTask) error
	BulkSoftDelete(ids []int) (int64, []string, error)
	UpdateStatus(id int, status string, errorMessage ...string) error
}

type ScanCreateCommandStore interface {
	CreateWithInputTargetsAndTasks(scan *CreateScan, inputs []CreateScanInputTarget, tasks []CreateScanTask) error
}
