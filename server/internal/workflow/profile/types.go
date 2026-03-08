// Package profile provides workflow profile configuration management.
package profile

// WorkflowConfig is the canonical structured workflow configuration object.
type WorkflowConfig map[string]any

// Profile represents the YAML file structure for profile parsing.
type Profile struct {
	ID            string         `yaml:"id" json:"id"`
	Name          string         `yaml:"name" json:"name"`
	Description   string         `yaml:"description,omitempty" json:"description,omitempty"`
	Configuration WorkflowConfig `yaml:"configuration" json:"configuration"`
}
