package subdomain_discovery

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type generatedSchemaIdentity struct {
	ID            string                                `json:"$id"`
	Engine        string                                `json:"x-engine"`
	EngineVersion string                                `json:"x-engine-version"`
	Properties    map[string]generatedSchemaVersionProp `json:"properties"`
}

type generatedSchemaVersionProp struct {
	Enum []string `json:"enum"`
}

func TestContractDefinitionMatchesGeneratedSchemaAndDocs(t *testing.T) {
	definition := GetContractDefinition()
	expectedID := fmt.Sprintf("lunafox://schemas/engines/%s/%s", definition.WorkflowName, definition.SchemaVersion)

	workerSchema := loadGeneratedSchema(t, "schema_generated.json")
	require.Equal(t, expectedID, workerSchema.ID)
	require.Equal(t, definition.WorkflowName, workerSchema.Engine)
	require.Equal(t, definition.SchemaVersion, workerSchema.EngineVersion)
	require.Equal(t, []string{definition.APIVersion}, workerSchema.Properties["apiVersion"].Enum)
	require.Equal(t, []string{definition.SchemaVersion}, workerSchema.Properties["schemaVersion"].Enum)

	root := repoRootFromWorkflowDir(t)
	serverSchemaPath := filepath.Join(
		root,
		"server",
		"internal",
		"engineschema",
		fmt.Sprintf("%s-%s-%s.schema.json", definition.WorkflowName, definition.APIVersion, definition.SchemaVersion),
	)
	serverSchema := loadGeneratedSchema(t, serverSchemaPath)
	require.Equal(t, expectedID, serverSchema.ID)
	require.Equal(t, definition.WorkflowName, serverSchema.Engine)
	require.Equal(t, definition.SchemaVersion, serverSchema.EngineVersion)
	require.Equal(t, []string{definition.APIVersion}, serverSchema.Properties["apiVersion"].Enum)
	require.Equal(t, []string{definition.SchemaVersion}, serverSchema.Properties["schemaVersion"].Enum)

	docsPath := filepath.Join(root, "docs", "config-reference", definition.WorkflowName+".md")
	docsBytes, err := os.ReadFile(docsPath)
	require.NoError(t, err)
	docs := string(docsBytes)
	require.Contains(t, docs, fmt.Sprintf("<!-- Source: %s -->", ContractSourcePath))
	require.Contains(t, docs, fmt.Sprintf("- Workflow: `%s`", definition.WorkflowName))
	require.Contains(t, docs, fmt.Sprintf("- API Version: `%s`", definition.APIVersion))
	require.Contains(t, docs, fmt.Sprintf("- Schema Version: `%s`", definition.SchemaVersion))
}

func loadGeneratedSchema(t *testing.T, path string) generatedSchemaIdentity {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoErrorf(t, err, "schema file not found: %s", path)

	var schema generatedSchemaIdentity
	require.NoError(t, json.Unmarshal(data, &schema))
	return schema
}

func repoRootFromWorkflowDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Clean(filepath.Join(wd, "..", "..", "..", ".."))
}
