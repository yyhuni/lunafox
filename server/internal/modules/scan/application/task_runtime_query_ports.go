package application

type TaskRuntimeQueryStore interface {
	GetTaskRuntimeByID(id int) (*TaskScanRecord, error)
}
