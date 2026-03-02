package workflow

// ContractDefinition is the workflow contract source-of-truth used for schema/docs/code generation.
type ContractDefinition struct {
	WorkflowName string
	DisplayName  string
	Description  string

	APIVersion    string
	SchemaVersion string
	TargetTypes   []string
	Stages        []ContractStageDefinition
}

type ContractStageDefinition struct {
	ID          string
	Name        string
	Description string
	Required    bool
	Parallel    bool
	Notes       []string
	Tools       []ContractToolDefinition
}

type ContractToolDefinition struct {
	ID          string
	Description string
	Params      []ContractParamDefinition
}

type ContractParamDefinition struct {
	Key                 string
	Type                string
	Description         string
	RequiredWhenEnabled bool
}
