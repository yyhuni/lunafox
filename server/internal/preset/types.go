// Package preset provides preset engine configuration management.
package preset

// PresetFile represents the YAML file structure for parsing.
// Also used as the internal Preset type since they're identical now.
type PresetFile struct {
	ID            string `yaml:"id" json:"id"`
	Name          string `yaml:"name" json:"name"`
	Description   string `yaml:"description,omitempty" json:"description,omitempty"`
	Configuration string `yaml:"configuration" json:"configuration"`
}

// Preset is an alias for PresetFile.
type Preset = PresetFile
