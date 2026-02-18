package activity

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// CommandBuilder builds commands from templates using Go Template
type CommandBuilder struct {
	funcMap template.FuncMap
}

// NewCommandBuilder creates a new command builder
func NewCommandBuilder() *CommandBuilder {
	return &CommandBuilder{
		funcMap: template.FuncMap{
			"quote": func(s string) string {
				return fmt.Sprintf("%q", s)
			},
			"lower": strings.ToLower,
			"upper": strings.ToUpper,
			"join": func(sep string, elems []string) string {
				return strings.Join(elems, sep)
			},
		},
	}
}

// Build constructs a command from a template with the given parameters
// params: required parameters (e.g., Domain, OutputFile)
// config: user configuration (optional parameters)
func (b *CommandBuilder) Build(tmpl CommandTemplate, params map[string]any, config map[string]any) (string, error) {
	// 1. Merge data
	data, err := b.mergeParameters(tmpl, params, config)
	if err != nil {
		return "", err
	}

	// 2. Build complete template
	cmdTemplate := b.buildCommandTemplate(tmpl, data)

	// 3. Execute Go Template
	t, err := template.New("command").
		Funcs(b.funcMap).
		Option("missingkey=error").
		Parse(cmdTemplate)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return strings.TrimSpace(buf.String()), nil
}

// mergeParameters merges required parameters, default values, and user configuration
func (b *CommandBuilder) mergeParameters(
	tmpl CommandTemplate,
	params map[string]any,
	config map[string]any,
) (map[string]any, error) {
	result := make(map[string]any)

	// 1. Add required parameters
	for key, value := range params {
		result[key] = value
	}

	// 2. Add optional parameters (explicit user configuration only)
	for _, param := range allParams(tmpl) {
		paramVar := param.Var
		semanticID := param.SemanticID
		if semanticID == "" {
			return nil, fmt.Errorf("parameter %s: semantic_id is required", paramVar)
		}
		if userValue, exists := config[semanticID]; exists {
			// User configuration takes priority
			if err := validateType(userValue, param.ConfigSchema.Type); err != nil {
				return nil, fmt.Errorf("parameter %s: %w", paramVar, err)
			}
			result[paramVar] = userValue
		} else if param.ConfigSchema.Required {
			// Required parameter is missing
			return nil, fmt.Errorf("required parameter %s is missing", semanticID)
		}
		// Optional parameter with no default value: don't add to result
	}

	return result, nil
}

// buildCommandTemplate builds the complete command template string
func (b *CommandBuilder) buildCommandTemplate(tmpl CommandTemplate, data map[string]any) string {
	cmd := strings.TrimSpace(tmpl.BaseCommand)

	// Only add flags for parameters that have values
	for _, param := range tmpl.CLIParams {
		if _, exists := data[param.Var]; exists && param.Arg != "" {
			cmd += " " + param.Arg
		}
	}

	return cmd
}
