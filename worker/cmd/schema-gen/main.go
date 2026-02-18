package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/yyhuni/lunafox/worker/internal/activity"
	"github.com/yyhuni/lunafox/worker/internal/workflow"
	"gopkg.in/yaml.v3"
)

// JSONSchema represents a JSON Schema Draft 7 structure
type JSONSchema struct {
	Schema string `json:"$schema"`
	ID     string `json:"$id,omitempty"`

	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`

	// Engine identity (custom schema annotations)
	Engine        string `json:"x-engine,omitempty"`
	EngineVersion string `json:"x-engine-version,omitempty"`

	Type                 string                     `json:"type"`
	Properties           map[string]*PropertySchema `json:"properties,omitempty"`
	Required             []string                   `json:"required,omitempty"`
	AdditionalProperties bool                       `json:"additionalProperties"`
	Metadata             map[string]interface{}     `json:"x-metadata,omitempty"`
}

// PropertySchema represents a property in JSON Schema
//
// Note: We intentionally model only the subset of JSON Schema Draft 7 features
// we need for config validation.
type PropertySchema struct {
	Type        string                     `json:"type,omitempty"`
	Description string                     `json:"description,omitempty"`
	Default     interface{}                `json:"default,omitempty"`
	Const       interface{}                `json:"const,omitempty"`
	Properties  map[string]*PropertySchema `json:"properties,omitempty"`
	Required    []string                   `json:"required,omitempty"`

	// Object strictness
	AdditionalProperties *bool `json:"additionalProperties,omitempty"`

	// Conditional validation (Draft 7)
	If   *PropertySchema `json:"if,omitempty"`
	Then *PropertySchema `json:"then,omitempty"`
	Else *PropertySchema `json:"else,omitempty"`

	// Tool metadata extension fields
	Stage   string `json:"x-stage,omitempty"`
	Warning string `json:"x-warning,omitempty"`
}

// TemplateFile represents the structure of templates.yaml
type TemplateFile struct {
	Metadata workflow.WorkflowMetadata           `yaml:"metadata"`
	Tools    map[string]activity.CommandTemplate `yaml:"tools"`
}

func main() {
	inputFile := flag.String("input", "", "Input templates.yaml file")
	outputFile := flag.String("output", "", "Output JSON Schema file")
	flag.Parse()

	if *inputFile == "" || *outputFile == "" {
		fmt.Fprintf(os.Stderr, "Usage: schema-gen -input <templates.yaml> -output <schema.json>\n")
		os.Exit(1)
	}

	// Read template file
	data, err := os.ReadFile(*inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input file: %v\n", err)
		os.Exit(1)
	}

	// Parse YAML
	var templateFile TemplateFile
	if err := yaml.Unmarshal(data, &templateFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing YAML: %v\n", err)
		os.Exit(1)
	}

	// Generate JSON Schema
	schema := generateSchema(templateFile)

	// Output JSON
	output, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating JSON: %v\n", err)
		os.Exit(1)
	}

	// Write to file
	if err := os.WriteFile(*outputFile, output, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Schema generated successfully: %s\n", *outputFile)
}

func generateSchema(templateFile TemplateFile) *JSONSchema {
	schema := &JSONSchema{
		Schema:      "http://json-schema.org/draft-07/schema#",
		ID:          schemaID(templateFile.Metadata.Name, templateFile.Metadata.Version),
		Title:       templateFile.Metadata.DisplayName,
		Description: templateFile.Metadata.Description,

		Engine:        templateFile.Metadata.Name,
		EngineVersion: templateFile.Metadata.Version,

		Type:                 "object",
		Properties:           make(map[string]*PropertySchema),
		AdditionalProperties: false,
		Metadata: map[string]interface{}{
			"name":         templateFile.Metadata.Name,
			"version":      templateFile.Metadata.Version,
			"target_types": templateFile.Metadata.TargetTypes,
			"stages":       templateFile.Metadata.Stages,
		},
	}
	// Group tools by stage
	toolsByStage := make(map[string][]string)
	for toolName, tool := range templateFile.Tools {
		stage := tool.Metadata.Stage
		toolsByStage[stage] = append(toolsByStage[stage], toolName)
	}

	// Generate schema for each stage
	var requiredStages []string
	for _, stage := range templateFile.Metadata.Stages {
		stageSchema := &PropertySchema{
			Type:       "object",
			Properties: make(map[string]*PropertySchema),
		}
		// Stage config should be strict: only {enabled, tools}
		stageSchema.AdditionalProperties = boolPtr(false)

		stageSchema.Properties["enabled"] = &PropertySchema{
			Type:        "boolean",
			Description: "Whether to enable this stage",
		}

		toolsSchema := &PropertySchema{
			Type:       "object",
			Properties: make(map[string]*PropertySchema),
		}
		// Tools object should be strict: only known tool keys for this stage
		toolsSchema.AdditionalProperties = boolPtr(false)

		tools := toolsByStage[stage.ID]
		if len(tools) > 0 {
			sort.Strings(tools)
		}

		var requiredTools []string
		for _, toolName := range tools {
			tool := templateFile.Tools[toolName]
			toolSchema := &PropertySchema{
				Type:        "object",
				Description: tool.Metadata.Description,
				Properties:  make(map[string]*PropertySchema),
				Stage:       tool.Metadata.Stage,
				Warning:     tool.Metadata.Warning,
			}

			// Tool config should be strict: only {enabled, <known param keys>}
			toolSchema.AdditionalProperties = boolPtr(false)

			toolSchema.Properties["enabled"] = &PropertySchema{
				Type:        "boolean",
				Description: "Whether to enable this tool",
			}
			// Explicit config: tool.enabled must always be present (even when false)
			toolSchema.Required = []string{"enabled"}

			var requiredParams []string
			for _, param := range append(append([]activity.Parameter{}, tool.RuntimeParams...), tool.CLIParams...) {
				paramSchema := &PropertySchema{
					Description: param.Documentation.Description,
				}

				switch param.ConfigSchema.Type {
				case "string":
					paramSchema.Type = "string"
				case "integer":
					paramSchema.Type = "integer"
				case "boolean":
					paramSchema.Type = "boolean"
				default:
					paramSchema.Type = "string"
				}
				toolSchema.Properties[param.ConfigSchema.Key] = paramSchema
				if param.ConfigSchema.Required {
					requiredParams = append(requiredParams, param.ConfigSchema.Key)
				}
			}

			// Conditional required params: only required when enabled=true.
			// This matches worker runtime behavior (disabled tools do not need full params).
			if len(requiredParams) > 0 {
				toolSchema.If = &PropertySchema{
					Type: "object",
					Properties: map[string]*PropertySchema{
						"enabled": {Const: true},
					},
					Required: []string{"enabled"},
				}
				toolSchema.Then = &PropertySchema{
					Type:     "object",
					Required: requiredParams,
				}
			}

			toolsSchema.Properties[toolName] = toolSchema
			requiredTools = append(requiredTools, toolName)
		}

		if len(requiredTools) > 0 {
			toolsSchema.Required = requiredTools
		}

		stageSchema.Properties["tools"] = toolsSchema
		stageSchema.Required = []string{"enabled", "tools"}

		schema.Properties[stage.ID] = stageSchema
		requiredStages = append(requiredStages, stage.ID)
	}

	if len(requiredStages) > 0 {
		schema.Required = requiredStages
	}

	return schema
}

func schemaID(name, version string) string {
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)
	if name == "" {
		return ""
	}
	if version == "" {
		return "lunafox://schemas/engines/" + name
	}
	return "lunafox://schemas/engines/" + name + "/" + version
}

func boolPtr(v bool) *bool { return &v }
