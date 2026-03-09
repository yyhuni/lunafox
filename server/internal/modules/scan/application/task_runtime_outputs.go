package application

// TaskAssignment is the execution payload assigned to an agent after a task is
// pulled from the scheduler. It is not a persistence model; it is the runtime
// handoff contract used by the gRPC task-assignment path.
type TaskAssignment struct {
	TaskID         int
	ScanID         int
	Stage          int
	WorkflowID     string
	TargetID       int
	TargetName     string
	TargetType     string
	WorkspaceDir   string
	WorkflowConfig map[string]any
}
