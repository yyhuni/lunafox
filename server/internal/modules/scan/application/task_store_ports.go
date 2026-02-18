package application

type TaskStore interface {
	TaskQueryStore
	TaskCommandStore
}
