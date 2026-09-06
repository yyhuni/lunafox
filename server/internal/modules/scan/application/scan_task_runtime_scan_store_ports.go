package application

type ScanTaskRuntimeScanStore interface {
	ScanTaskRuntimeScanQueryStore
	ScanTaskRuntimeScanCommandStore
}
