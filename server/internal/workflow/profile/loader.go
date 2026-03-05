package preset

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed presets/*.yaml
var presetsFS embed.FS

const presetsDir = "presets"

// Loader loads and manages preset engine configurations.
type Loader struct {
	presets    []Preset
	presetsMap map[string]*Preset
}

// NewLoader creates a new Loader and loads all presets from embedded files.
func NewLoader() (*Loader, error) {
	l := &Loader{
		presets:    []Preset{},
		presetsMap: make(map[string]*Preset),
	}

	if err := l.load(); err != nil {
		return nil, err
	}

	return l, nil
}

// load reads all YAML files from the embedded filesystem.
func (l *Loader) load() error {
	entries, err := presetsFS.ReadDir(presetsDir)
	if err != nil {
		return fmt.Errorf("failed to read presets directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Skip files starting with underscore (templates/examples)
		if strings.HasPrefix(name, "_") {
			continue
		}

		// Only process .yaml and .yml files
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		filePath := filepath.Join(presetsDir, name)
		data, err := presetsFS.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read preset file %s: %w", name, err)
		}

		var preset Preset
		if err := yaml.Unmarshal(data, &preset); err != nil {
			return fmt.Errorf("failed to parse preset file %s: %w", name, err)
		}

		// Validate required fields
		if preset.ID == "" {
			return fmt.Errorf("preset file %s: missing required field 'id'", name)
		}
		if preset.Name == "" {
			return fmt.Errorf("preset file %s: missing required field 'name'", name)
		}
		if preset.Configuration == "" {
			return fmt.Errorf("preset file %s: missing required field 'configuration'", name)
		}

		// Check for duplicate IDs
		if _, exists := l.presetsMap[preset.ID]; exists {
			return fmt.Errorf("duplicate preset id '%s' in file %s", preset.ID, name)
		}

		// Validate configuration against schemas
		if err := ValidateConfiguration(preset.Configuration); err != nil {
			return fmt.Errorf("preset file %s: %w", name, err)
		}

		l.presets = append(l.presets, preset)
		l.presetsMap[preset.ID] = &l.presets[len(l.presets)-1]
	}

	return nil
}

// List returns all loaded presets.
func (l *Loader) List() []Preset {
	return l.presets
}

// GetByID returns a preset by its ID, or nil if not found.
func (l *Loader) GetByID(id string) *Preset {
	return l.presetsMap[id]
}
