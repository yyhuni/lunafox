package subdomain_discovery

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type generatedSchemaIdentity struct {
	ID         string                 `json:"$id"`
	Properties map[string]interface{} `json:"properties"`
}

type generatedManifestIdentity struct {
	WorkflowID string `json:"workflowId"`
}

type generatedProfileIdentity struct {
	ID            string                 `yaml:"id"`
	Name          string                 `yaml:"name"`
	Description   string                 `yaml:"description"`
	Configuration map[string]interface{} `yaml:"configuration"`
}

func TestContractDefinitionMatchesGeneratedArtifactsAndDocs(t *testing.T) {
	definition := GetContractDefinition()
	expectedSchemaID := fmt.Sprintf("lunafox://schemas/workflows/%s", definition.WorkflowID)

	root := repoRootFromWorkflowDir(t)
	serverSchema := loadGeneratedSchema(t, filepath.Join(root, "server", "internal", "workflow", "schema", fmt.Sprintf("%s.schema.json", definition.WorkflowID)))
	require.Equal(t, expectedSchemaID, serverSchema.ID)
	_, hasAPIVersion := serverSchema.Properties["apiVersion"]
	_, hasSchemaVersion := serverSchema.Properties["schemaVersion"]
	require.False(t, hasAPIVersion)
	require.False(t, hasSchemaVersion)

	serverManifest := loadGeneratedManifest(t, filepath.Join(root, "server", "internal", "workflow", "manifest", fmt.Sprintf("%s.manifest.json", definition.WorkflowID)))
	require.Equal(t, definition.WorkflowID, serverManifest.WorkflowID)

	profile := loadGeneratedProfile(t, filepath.Join(root, "server", "internal", "workflow", "profile", "profiles", fmt.Sprintf("%s.yaml", definition.WorkflowID)))
	require.Equal(t, definition.WorkflowID, profile.ID)
	require.Equal(t, definition.DisplayName, profile.Name)
	require.Equal(t, definition.Description, profile.Description)
	docsPath := filepath.Join(root, "docs", "config-reference", definition.WorkflowID+".md")
	docsBytes, err := os.ReadFile(docsPath)
	require.NoError(t, err)
	docs := string(docsBytes)
	require.Contains(t, docs, fmt.Sprintf("<!-- Source: %s -->", ContractSourcePath))
	require.Contains(t, docs, fmt.Sprintf("- Workflow: `%s`", definition.WorkflowID))
	require.NotContains(t, docs, "API Version")
	require.NotContains(t, docs, "Schema Version")
}

func loadGeneratedSchema(t *testing.T, path string) generatedSchemaIdentity {
	t.Helper()
	payload, err := os.ReadFile(path)
	require.NoErrorf(t, err, "schema file not found: %s", path)
	var schema generatedSchemaIdentity
	require.NoError(t, json.Unmarshal(payload, &schema))
	return schema
}

func loadGeneratedManifest(t *testing.T, path string) generatedManifestIdentity {
	t.Helper()
	payload, err := os.ReadFile(path)
	require.NoErrorf(t, err, "manifest file not found: %s", path)
	var manifest generatedManifestIdentity
	require.NoError(t, json.Unmarshal(payload, &manifest))
	return manifest
}

func loadGeneratedProfile(t *testing.T, path string) generatedProfileIdentity {
	t.Helper()
	payload, err := os.ReadFile(path)
	require.NoErrorf(t, err, "profile file not found: %s", path)
	var profile generatedProfileIdentity
	require.NoError(t, yaml.Unmarshal(payload, &profile))
	return profile
}

func repoRootFromWorkflowDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Clean(filepath.Join(wd, "..", "..", "..", ".."))
}
