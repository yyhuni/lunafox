package workflowmanifest

import "regexp"

type Manifest struct {
	ManifestVersion string           `json:"manifestVersion"`
	WorkflowID      string           `json:"workflowId"`
	DisplayName     string           `json:"displayName"`
	Description     string           `json:"description,omitempty"`
	Executor        ManifestExecutor `json:"executor"`
	Stages          []ManifestStage  `json:"stages"`
}

type ManifestExecutor struct {
	Type string `json:"type"`
	Ref  string `json:"ref"`
}

type ManifestStage struct {
	StageID     string         `json:"stageId"`
	DisplayName string         `json:"displayName"`
	Description string         `json:"description,omitempty"`
	Required    bool           `json:"required"`
	Parallel    bool           `json:"parallel"`
	Notes       []string       `json:"notes,omitempty"`
	Tools       []ManifestTool `json:"tools"`
}

type ManifestTool struct {
	ToolID      string          `json:"toolId"`
	Description string          `json:"description,omitempty"`
	Params      []ManifestParam `json:"params,omitempty"`
}

type ManifestParam struct {
	ConfigKey           string `json:"configKey"`
	ValueType           string `json:"valueType"`
	Description         string `json:"description,omitempty"`
	RequiredWhenEnabled bool   `json:"requiredWhenEnabled"`
}

var (
	workflowIDPattern       = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
	componentIDPattern      = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)
	configKeyPattern        = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)
	reservedWorkflowIDNames = map[string]struct{}{"all": {}, "default": {}}
)
