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
	loader := &Loader{
		profiles:    []Profile{},
		profilesMap: make(map[string]*Profile),
	}

	if err := loader.load(); err != nil {
		return nil, err
	}

	return loader, nil
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
		if shouldSkipProfileEntry(name) {
			continue
		}

		profile, err := loadProfileFile(name)
		if err != nil {
			return err
		}
		if err := l.registerProfile(name, profile); err != nil {
			return err
		}
	}

	return nil
}

func shouldSkipProfileEntry(name string) bool {
	if strings.HasPrefix(name, "_") {
		return true
	}

	ext := strings.ToLower(filepath.Ext(name))
	return ext != ".yaml" && ext != ".yml"
}

func loadProfileFile(name string) (Profile, error) {
	filePath := filepath.Join(profilesDir, name)
	data, err := profilesFS.ReadFile(filePath)
	if err != nil {
		return Profile{}, fmt.Errorf("failed to read profile file %s: %w", name, err)
	}

	var profile Profile
	if err := yaml.Unmarshal(data, &profile); err != nil {
		return Profile{}, fmt.Errorf("failed to parse profile file %s: %w", name, err)
	}

	return profile, nil
}

func (l *Loader) registerProfile(name string, profile Profile) error {
	if err := validateLoadedProfile(name, profile); err != nil {
		return err
	}
	if _, exists := l.profilesMap[profile.ID]; exists {
		return fmt.Errorf("duplicate profile id '%s' in file %s", profile.ID, name)
	}

	l.profiles = append(l.profiles, profile)
	l.profilesMap[profile.ID] = &l.profiles[len(l.profiles)-1]
	return nil
}

func validateLoadedProfile(name string, profile Profile) error {
	if profile.ID == "" {
		return fmt.Errorf("profile file %s: missing required field 'id'", name)
	}
	if profile.Name == "" {
		return fmt.Errorf("profile file %s: missing required field 'name'", name)
	}
	if len(profile.Configuration) == 0 {
		return fmt.Errorf("profile file %s: missing required field 'configuration'", name)
	}
	if err := ValidateConfiguration(profile.Configuration); err != nil {
		return fmt.Errorf("profile file %s: %w", name, err)
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
