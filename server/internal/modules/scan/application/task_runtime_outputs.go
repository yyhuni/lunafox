package application

type TaskAssignment struct {
	TaskID       int
	ScanID       int
	Stage        int
	WorkflowName string
	TargetID     int
	TargetName   string
	TargetType   string
	WorkspaceDir string
	Config       string
}
