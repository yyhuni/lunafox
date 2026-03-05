package profile

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed profiles/*.yaml
var profilesFS embed.FS

const profilesDir = "profiles"

// Loader loads and manages workflow profiles.
type Loader struct {
	profiles    []Profile
	profilesMap map[string]*Profile
}

// NewLoader creates a new Loader and loads all profiles from embedded files.
func NewLoader() (*Loader, error) {
	l := &Loader{
		profiles:    []Profile{},
		profilesMap: make(map[string]*Profile),
	}

	if err := l.load(); err != nil {
		return nil, err
	}

	return l, nil
}

// load reads all YAML files from the embedded filesystem.
func (l *Loader) load() error {
	entries, err := profilesFS.ReadDir(profilesDir)
	if err != nil {
		return fmt.Errorf("failed to read profiles directory: %w", err)
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

		filePath := filepath.Join(profilesDir, name)
		data, err := profilesFS.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read profile file %s: %w", name, err)
		}

		var profile Profile
		if err := yaml.Unmarshal(data, &profile); err != nil {
			return fmt.Errorf("failed to parse profile file %s: %w", name, err)
		}

		// Validate required fields
		if profile.ID == "" {
			return fmt.Errorf("profile file %s: missing required field 'id'", name)
		}
		if profile.Name == "" {
			return fmt.Errorf("profile file %s: missing required field 'name'", name)
		}
		if profile.Configuration == "" {
			return fmt.Errorf("profile file %s: missing required field 'configuration'", name)
		}

		// Check for duplicate IDs
		if _, exists := l.profilesMap[profile.ID]; exists {
			return fmt.Errorf("duplicate profile id '%s' in file %s", profile.ID, name)
		}

		// Validate configuration against schemas
		if err := ValidateConfiguration(profile.Configuration); err != nil {
			return fmt.Errorf("profile file %s: %w", name, err)
		}

		l.profiles = append(l.profiles, profile)
		l.profilesMap[profile.ID] = &l.profiles[len(l.profiles)-1]
	}

	return nil
}

// List returns all loaded profiles.
func (l *Loader) List() []Profile {
	return l.profiles
}

// GetByID returns a profile by its ID, or nil if not found.
func (l *Loader) GetByID(id string) *Profile {
	return l.profilesMap[id]
}
