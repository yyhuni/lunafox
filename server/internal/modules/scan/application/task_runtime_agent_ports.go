package application

import "context"

type TaskRuntimeAgentRecord struct {
	ID            int
	AgentVersion  string
	WorkerVersion string
	Status        string
}

type TaskRuntimeAgentStore interface {
	GetTaskRuntimeAgentByID(ctx context.Context, id int) (*TaskRuntimeAgentRecord, error)
}
