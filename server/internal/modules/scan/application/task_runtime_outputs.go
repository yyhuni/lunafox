package application

type TaskAssignment struct {
	TaskID             int
	ScanID             int
	Stage              int
	WorkflowID         string
	TargetID           int
	TargetName         string
	TargetType         string
	WorkspaceDir       string
	WorkflowConfigYAML string
}
