package application

type TaskRuntimeCommandStore interface {
	UpdateStatus(id int, status string, failure *FailureDetail) error
}
