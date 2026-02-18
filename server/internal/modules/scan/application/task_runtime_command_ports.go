package application

type TaskRuntimeCommandStore interface {
	UpdateStatus(id int, status string, errorMessage string) error
}
