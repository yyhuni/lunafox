package activity

import (
	"embed"
	"fmt"
	"regexp"
	"sync"
	"text/template"

	"github.com/yyhuni/lunafox/worker/internal/pkg"
	"github.com/yyhuni/lunafox/worker/internal/workflow"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// TemplateFile defines the complete structure of a template file
type TemplateFile struct {
	Metadata workflow.WorkflowMetadata  `yaml:"metadata"`
	Tools    map[string]CommandTemplate `yaml:"tools"`
}

func isValidIdentifier(name string) bool {
	identifier := regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	return identifier.MatchString(name)
}
func isValidSemanticID(name string) bool {
	// Allow identifier or kebab-style names (letters/digits with hyphens)
	if isValidIdentifier(name) {
		return true
	}
	kebabLike := regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*(?:-[A-Za-z0-9]+)*$`)
	return kebabLike.MatchString(name)
}

func isValidConfigKey(key string) bool {
	kebab := regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)
	return kebab.MatchString(key)
}

func isValidDisplay(value string) bool {
	switch value {
	case "value", "comment", "hidden":
		return true
	default:
		return false
	}
}

// TemplateLoader loads and caches command templates from embedded YAML
type TemplateLoader struct {
	fs       embed.FS
	filename string
	once     sync.Once
	cache    map[string]CommandTemplate
	metadata workflow.WorkflowMetadata
	err      error
}

// NewTemplateLoader creates a new template loader
func NewTemplateLoader(fs embed.FS, filename string) *TemplateLoader {
	return &TemplateLoader{
		fs:       fs,
		filename: filename,
	}
}

// Load loads templates (cached with sync.Once)
func (l *TemplateLoader) Load() (map[string]CommandTemplate, error) {
	l.once.Do(func() {
		data, err := l.fs.ReadFile(l.filename)
		if err != nil {
			l.err = fmt.Errorf("failed to read %s: %w", l.filename, err)
			pkg.Logger.Error("Failed to load templates",
				zap.String("file", l.filename),
				zap.Error(l.err))
			return
		}

		var templateFile TemplateFile
		if err := yaml.Unmarshal(data, &templateFile); err != nil {
			l.err = fmt.Errorf("failed to parse %s: %w", l.filename, err)
			pkg.Logger.Error("Failed to parse templates",
				zap.String("file", l.filename),
				zap.Error(l.err))
			return
		}

		l.cache = templateFile.Tools
		l.metadata = templateFile.Metadata

		if err := l.validate(); err != nil {
			l.err = err
			pkg.Logger.Error("Failed to validate templates",
				zap.String("file", l.filename),
				zap.Error(l.err))
			return
		}

		pkg.Logger.Info("Templates loaded",
			zap.String("file", l.filename),
			zap.String("workflow", l.metadata.Name),
			zap.Int("count", len(l.cache)))
	})

	return l.cache, l.err
}

// Get returns a specific template by name
func (l *TemplateLoader) Get(name string) (CommandTemplate, error) {
	templates, err := l.Load()
	if err != nil {
		return CommandTemplate{}, fmt.Errorf("templates not loaded: %w", err)
	}
	tmpl, ok := templates[name]
	if !ok {
		return CommandTemplate{}, fmt.Errorf("template not found: %s", name)
	}
	return tmpl, nil
}

// GetMetadata returns the workflow metadata
func (l *TemplateLoader) GetMetadata() (workflow.WorkflowMetadata, error) {
	_, err := l.Load()
	if err != nil {
		return workflow.WorkflowMetadata{}, fmt.Errorf("templates not loaded: %w", err)
	}
	return l.metadata, nil
}

// validate checks all templates for syntax errors
func (l *TemplateLoader) validate() error {
	// Validate metadata
	if l.metadata.Name == "" {
		return fmt.Errorf("workflow metadata: name is required")
	}
	if l.metadata.Version == "" {
		return fmt.Errorf("workflow metadata: version is required")
	}

	// Build stage ID set
	stageIDs := make(map[string]bool)
	for _, stage := range l.metadata.Stages {
		if stage.ID == "" {
			return fmt.Errorf("stage metadata: id is required")
		}
		stageIDs[stage.ID] = true
	}

	// Create funcMap for validation (consistent with CommandBuilder)
	funcMap := template.FuncMap{
		"quote": func(s string) string { return fmt.Sprintf("%q", s) },
		"lower": func(s string) string { return s },
		"upper": func(s string) string { return s },
		"join":  func(sep string, elems []string) string { return "" },
	}

	// Validate tool templates
	for name, tmpl := range l.cache {
		if tmpl.BaseCommand == "" {
			return fmt.Errorf("template %s: base_command is required", name)
		}

		// Validate Go Template syntax (provide funcMap)
		if _, err := template.New(name).Funcs(funcMap).Parse(tmpl.BaseCommand); err != nil {
			return fmt.Errorf("template %s: invalid base_command syntax: %w", name, err)
		}

		// Validate tool metadata
		if tmpl.Metadata.DisplayName == "" {
			return fmt.Errorf("template %s: metadata.display_name is required", name)
		}
		if tmpl.Metadata.Stage == "" {
			return fmt.Errorf("template %s: metadata.stage is required", name)
		}

		// Validate that the referenced stage exists
		if !stageIDs[tmpl.Metadata.Stage] {
			return fmt.Errorf("template %s: stage %s not found in workflow metadata", name, tmpl.Metadata.Stage)
		}

		// Validate parameters
		paramNames := make(map[string]struct{})
		semanticIDs := make(map[string]struct{})
		configKeys := make(map[string]struct{})
		validateParams := func(params []Parameter, kind string) error {
			for _, param := range params {
				paramVar := param.Var
				if paramVar == "" {
					return fmt.Errorf("template %s: %s param var is required", name, kind)
				}
				if !isValidIdentifier(paramVar) {
					return fmt.Errorf("template %s, %s param %s: invalid var (must be valid identifier)", name, kind, paramVar)
				}
				if _, exists := paramNames[paramVar]; exists {
					return fmt.Errorf("template %s: duplicate var %s", name, paramVar)
				}
				paramNames[paramVar] = struct{}{}
				semanticID := param.SemanticID
				if semanticID == "" {
					return fmt.Errorf("template %s, %s param %s: semantic_id is required", name, kind, paramVar)
				}
				if !isValidSemanticID(semanticID) {
					return fmt.Errorf("template %s, %s param %s: invalid semantic_id %s (must be valid identifier or kebab-case)", name, kind, paramVar, semanticID)
				}
				if _, exists := semanticIDs[semanticID]; exists {
					return fmt.Errorf("template %s: duplicate semantic_id %s", name, semanticID)
				}
				semanticIDs[semanticID] = struct{}{}

				configKey := param.ConfigSchema.Key
				if configKey == "" {
					return fmt.Errorf("template %s, %s param %s: config_schema.key is required", name, kind, paramVar)
				}
				if !isValidConfigKey(configKey) {
					return fmt.Errorf("template %s, %s param %s: invalid config_schema.key %s (must be kebab-case)", name, kind, paramVar, configKey)
				}
				if _, exists := configKeys[configKey]; exists {
					return fmt.Errorf("template %s: duplicate config_schema.key %s", name, configKey)
				}
				configKeys[configKey] = struct{}{}

				// Validate type
				if param.ConfigSchema.Type == "" {
					return fmt.Errorf("template %s, %s param %s: config_schema.type is required", name, kind, paramVar)
				}
				if param.ConfigSchema.Type != "string" && param.ConfigSchema.Type != "integer" && param.ConfigSchema.Type != "boolean" {
					return fmt.Errorf("template %s, %s param %s: invalid config_schema.type %s (must be string/integer/boolean)",
						name, kind, paramVar, param.ConfigSchema.Type)
				}

				if param.ConfigSchema.Required && param.ConfigSchema.Default != nil {
					return fmt.Errorf("template %s, %s param %s: required cannot be true when default is set", name, kind, paramVar)
				}

				// Validate default value type
				if param.ConfigSchema.Default != nil {
					if err := validateType(param.ConfigSchema.Default, param.ConfigSchema.Type); err != nil {
						return fmt.Errorf("template %s, %s param %s: %w", name, kind, paramVar, err)
					}
				}

				if param.ConfigExample.ShowAs == "" {
					return fmt.Errorf("template %s, %s param %s: config_example.show_as is required", name, kind, paramVar)
				}
				if !isValidDisplay(param.ConfigExample.ShowAs) {
					return fmt.Errorf("template %s, %s param %s: invalid config_example.show_as %s (must be value/comment/hidden)", name, kind, paramVar, param.ConfigExample.ShowAs)
				}
				if param.ConfigExample.Value != nil {
					if err := validateType(param.ConfigExample.Value, param.ConfigSchema.Type); err != nil {
						return fmt.Errorf("template %s, %s param %s: example.value %w", name, kind, paramVar, err)
					}
				}

				if param.Documentation.Description == "" {
					return fmt.Errorf("template %s, %s param %s: documentation.description is required", name, kind, paramVar)
				}

				if kind == "runtime" && param.Arg != "" {
					return fmt.Errorf("template %s, runtime param %s: arg must be empty", name, paramVar)
				}

				// Validate arg's Go Template syntax (provide funcMap)
				if kind == "cli" && param.Arg != "" {
					if _, err := template.New(name + "_" + paramVar).Funcs(funcMap).Parse(param.Arg); err != nil {
						return fmt.Errorf("template %s, cli param %s: invalid arg syntax: %w", name, paramVar, err)
					}
				}
			}
			return nil
		}
		if err := validateParams(tmpl.RuntimeParams, "runtime"); err != nil {
			return err
		}
		if err := validateParams(tmpl.CLIParams, "cli"); err != nil {
			return err
		}

		// Validate internal params (optional)
		for key, value := range tmpl.InternalParams {
			if key == "" {
				return fmt.Errorf("template %s: internal_params key is required", name)
			}
			if key == "enabled" {
				return fmt.Errorf("template %s: internal_params key %s is reserved", name, key)
			}
			if !isValidConfigKey(key) {
				return fmt.Errorf("template %s: invalid internal_params key %s (must be kebab-case)", name, key)
			}
			if _, exists := configKeys[key]; exists {
				return fmt.Errorf("template %s: internal_params key %s conflicts with config_schema.key", name, key)
			}
			if _, exists := semanticIDs[key]; exists {
				return fmt.Errorf("template %s: internal_params key %s conflicts with semantic_id", name, key)
			}
			if value == nil {
				return fmt.Errorf("template %s: internal_params key %s has nil value", name, key)
			}
		}
	}

	// Validate that each stage has at least one tool
	stageTools := make(map[string]int)
	for _, tmpl := range l.cache {
		stageTools[tmpl.Metadata.Stage]++
	}
	for _, stage := range l.metadata.Stages {
		if stageTools[stage.ID] == 0 {
			pkg.Logger.Warn("Stage has no tools defined",
				zap.String("stage", stage.ID))
		}
	}

	return nil
}

// validateType validates that the value type matches
func validateType(value interface{}, expectedType string) error {
	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case "integer":
		switch value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return nil
		default:
			return fmt.Errorf("expected integer, got %T", value)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}
	}
	return nil
}
