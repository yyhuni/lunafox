package application

type TaskRuntimeScanStore interface {
	TaskRuntimeQueryStore
	TaskRuntimeCommandStore
}
