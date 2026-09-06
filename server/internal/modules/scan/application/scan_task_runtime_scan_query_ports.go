package application

type ScanTaskRuntimeScanQueryStore interface {
	GetScanForScanTask(scanID int) (*ScanTaskRuntimeScanRecord, error)
}
