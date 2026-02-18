package activity

// CommandTemplate defines a tool's command template (using Go Template syntax)
type CommandTemplate struct {
	Metadata       ToolMetadata   `yaml:"metadata" json:"metadata"`
	BaseCommand    string         `yaml:"base_command" json:"baseCommand"` // Uses {{.Var}} placeholders
	RuntimeParams  []Parameter    `yaml:"runtime_params,omitempty" json:"runtimeParams,omitempty"`
	CLIParams      []Parameter    `yaml:"cli_params,omitempty" json:"cliParams,omitempty"`
	InternalParams map[string]any `yaml:"internal_params,omitempty" json:"internalParams,omitempty"`
}

// Parameter defines all properties of a single parameter
type Parameter struct {
	SemanticID    string        `yaml:"semantic_id" json:"semanticId"`      // Stable semantic identifier
	Var           string        `yaml:"var" json:"var"`                     // Internal name (Go template variable)
	Arg           string        `yaml:"arg,omitempty" json:"arg,omitempty"` // Uses {{.Var}} placeholders
	ConfigSchema  ConfigSchema  `yaml:"config_schema" json:"configSchema"`
	ConfigExample ConfigExample `yaml:"config_example" json:"configExample"`
	Documentation Documentation `yaml:"documentation" json:"documentation"`
}

// ConfigSchema defines user config contract
type ConfigSchema struct {
	Key      string      `yaml:"key" json:"key"`                             // External config key (kebab-case)
	Type     string      `yaml:"type" json:"type"`                           // "string", "integer", "boolean"
	Required bool        `yaml:"required" json:"required"`                   // Whether required
	Default  interface{} `yaml:"default,omitempty" json:"default,omitempty"` // Default value
}

// ConfigExample defines example rendering controls
type ConfigExample struct {
	ShowAs  string      `yaml:"show_as" json:"showAs"`                      // value | comment | hidden
	Value   interface{} `yaml:"value,omitempty" json:"value,omitempty"`     // Optional example override
	Comment bool        `yaml:"comment,omitempty" json:"comment,omitempty"` // Whether to add inline comment
}

// Documentation defines parameter documentation
type Documentation struct {
	Description string `yaml:"description" json:"description"` // Parameter description
}

// ToolMetadata defines tool metadata
type ToolMetadata struct {
	DisplayName string   `yaml:"display_name" json:"displayName"`            // Display name
	Description string   `yaml:"description" json:"description"`             // Tool description
	Stage       string   `yaml:"stage" json:"stage"`                         // Stage (required)
	Warning     string   `yaml:"warning,omitempty" json:"warning,omitempty"` // Warning message
	Notes       []string `yaml:"notes,omitempty" json:"notes,omitempty"`     // Optional notes
}

func allParams(tmpl CommandTemplate) []Parameter {
	params := make([]Parameter, 0, len(tmpl.RuntimeParams)+len(tmpl.CLIParams))
	params = append(params, tmpl.RuntimeParams...)
	params = append(params, tmpl.CLIParams...)
	return params
}
