package dto

type TaskAssignment struct {
	TaskID       int    `json:"taskId"`
	ScanID       int    `json:"scanId"`
	Stage        int    `json:"stage"`
	WorkflowID   string `json:"workflowId"`
	TargetID     int    `json:"targetId"`
	TargetName   string `json:"targetName"`
	TargetType   string `json:"targetType"`
	WorkspaceDir string `json:"workspaceDir"`
	// WorkflowConfigYAML is the workflow-scoped YAML config slice for this task.
	WorkflowConfigYAML string `json:"workflowConfigYAML"`
}

type TaskStatusUpdateRequest struct {
	Status       string `json:"status" binding:"required"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}
