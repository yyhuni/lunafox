package domain

type Task struct {
	ID                 int    `json:"taskId"`
	ScanID             int    `json:"scanId"`
	Stage              int    `json:"stage"`
	WorkflowID         string `json:"workflowId"`
	TargetID           int    `json:"targetId"`
	TargetName         string `json:"targetName"`
	TargetType         string `json:"targetType"`
	WorkspaceDir       string `json:"workspaceDir"`
	WorkflowConfigYAML string `json:"workflowConfigYAML"`
}
