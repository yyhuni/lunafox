package workflowmanifest

import "regexp"

type Manifest struct {
	ManifestVersion        string          `json:"manifestVersion"`
	WorkflowID             string          `json:"workflowId"`
	DisplayName            string          `json:"displayName"`
	Description            string          `json:"description,omitempty"`
	ConfigSchemaID         string          `json:"configSchemaId"`
	SupportedTargetTypeIDs []string        `json:"supportedTargetTypeIds"`
	DefaultProfileID       string          `json:"defaultProfileId"`
	Stages                 []ManifestStage `json:"stages"`
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
	ConfigKey           string   `json:"configKey"`
	ValueType           string   `json:"valueType"`
	Description         string   `json:"description,omitempty"`
	RequiredWhenEnabled bool     `json:"requiredWhenEnabled"`
	DefaultValue        any      `json:"defaultValue,omitempty"`
	Minimum             *int     `json:"minimum,omitempty"`
	Maximum             *int     `json:"maximum,omitempty"`
	MinLength           *int     `json:"minLength,omitempty"`
	MaxLength           *int     `json:"maxLength,omitempty"`
	Pattern             string   `json:"pattern,omitempty"`
	Enum                []string `json:"enum,omitempty"`
}

type WorkflowMetadata struct {
	WorkflowID  string
	DisplayName string
	Description string
}

var (
	workflowIDPattern       = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
	componentIDPattern      = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)
	configKeyPattern        = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)
	reservedWorkflowIDNames = map[string]struct{}{"all": {}, "default": {}}
)
