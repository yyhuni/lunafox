package application

type ScanTaskStore interface {
	ScanTaskQueryStore
	ScanTaskCommandStore
}
