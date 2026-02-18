package domain

type Task struct {
	ID           int    `json:"taskId"`
	ScanID       int    `json:"scanId"`
	Stage        int    `json:"stage"`
	WorkflowName string `json:"workflowName"`
	TargetID     int    `json:"targetId"`
	TargetName   string `json:"targetName"`
	TargetType   string `json:"targetType"`
	WorkspaceDir string `json:"workspaceDir"`
	Config       string `json:"config"`
}
