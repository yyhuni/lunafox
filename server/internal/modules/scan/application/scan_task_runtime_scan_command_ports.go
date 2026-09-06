package application

type ScanTaskRuntimeScanCommandStore interface {
	UpdateScanStatus(id int, status string, failure *FailureDetail) error
}
