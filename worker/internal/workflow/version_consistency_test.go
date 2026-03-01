package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

var workflowVersionRegex = regexp.MustCompile(`^\d+\.\d+\.\d+([\-+][0-9A-Za-z.-]+)?$`)

type workflowTemplate struct {
	Metadata struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	} `yaml:"metadata"`
}

type schemaIdentity struct {
	ID            string `json:"$id"`
	Engine        string `json:"x-engine"`
	EngineVersion string `json:"x-engine-version"`
}

func TestWorkflowVersionConsistency(t *testing.T) {
	templateFiles := listWorkflowTemplateFiles(t)
	root := repoRoot(t)

	for _, templatePath := range templateFiles {
		templatePath := templatePath
		workflowDir := filepath.Dir(templatePath)

		t.Run(workflowDir, func(t *testing.T) {
			template := loadWorkflowTemplate(t, templatePath)
			require.NotEmpty(t, template.Metadata.Name, "workflow metadata.name is required")
			require.NotEmpty(t, template.Metadata.Version, "workflow metadata.version is required")
			require.Truef(
				t,
				workflowVersionRegex.MatchString(template.Metadata.Version),
				"workflow metadata.version must be semver (e.g. 1.0.0), got %q",
				template.Metadata.Version,
			)

			workerSchemaPath := filepath.Join(workflowDir, "schema_generated.json")
			workerSchema := loadSchemaIdentity(t, workerSchemaPath)
			require.Equal(
				t,
				template.Metadata.Name,
				workerSchema.Engine,
				"schema_generated.json x-engine should match templates.yaml metadata.name",
			)
			require.Equal(
				t,
				template.Metadata.Version,
				workerSchema.EngineVersion,
				"schema_generated.json x-engine-version should match templates.yaml metadata.version",
			)
			expectedID := fmt.Sprintf(
				"lunafox://schemas/engines/%s/%s",
				template.Metadata.Name,
				template.Metadata.Version,
			)
			require.Equal(
				t,
				expectedID,
				workerSchema.ID,
				"schema_generated.json $id should include workflow name and version",
			)

			serverSchemaPath := filepath.Join(
				root,
				"server",
				"internal",
				"engineschema",
				fmt.Sprintf("%s-v1-%s.schema.json", template.Metadata.Name, template.Metadata.Version),
			)
			serverSchema := loadSchemaIdentity(t, serverSchemaPath)
			require.Equal(
				t,
				template.Metadata.Name,
				serverSchema.Engine,
				"server schema x-engine should match templates.yaml metadata.name",
			)
			require.Equal(
				t,
				template.Metadata.Version,
				serverSchema.EngineVersion,
				"server schema x-engine-version should match templates.yaml metadata.version",
			)
			require.Equal(
				t,
				expectedID,
				serverSchema.ID,
				"server schema $id should include workflow name and version",
			)
		})
	}
}

func listWorkflowTemplateFiles(t *testing.T) []string {
	t.Helper()
	matches, err := filepath.Glob("*/templates.yaml")
	require.NoError(t, err)
	require.NotEmpty(t, matches, "no workflow templates.yaml files found")
	return matches
}

func loadWorkflowTemplate(t *testing.T, path string) workflowTemplate {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var tmpl workflowTemplate
	require.NoError(t, yaml.Unmarshal(data, &tmpl))
	return tmpl
}

func loadSchemaIdentity(t *testing.T, path string) schemaIdentity {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoErrorf(t, err, "schema file not found: %s", path)

	var schema schemaIdentity
	require.NoError(t, json.Unmarshal(data, &schema))
	return schema
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Clean(filepath.Join(wd, "..", "..", ".."))
}
