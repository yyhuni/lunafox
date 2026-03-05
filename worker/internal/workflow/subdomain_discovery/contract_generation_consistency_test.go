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
	ID         string                 `json:"$id"`
	Workflow   string                 `json:"x-workflow"`
	Properties map[string]interface{} `json:"properties"`
}

func TestContractDefinitionMatchesGeneratedSchemaAndDocs(t *testing.T) {
	definition := GetContractDefinition()
	expectedID := fmt.Sprintf("lunafox://schemas/workflows/%s", definition.WorkflowName)

	workerSchema := loadGeneratedSchema(
		t,
		fmt.Sprintf("generated/%s.schema.json", definition.WorkflowName),
	)
	require.Equal(t, expectedID, workerSchema.ID)
	require.Equal(t, definition.WorkflowName, workerSchema.Workflow)
	_, hasAPIVersion := workerSchema.Properties["apiVersion"]
	_, hasSchemaVersion := workerSchema.Properties["schemaVersion"]
	require.False(t, hasAPIVersion)
	require.False(t, hasSchemaVersion)

	root := repoRootFromWorkflowDir(t)
	serverSchemaPath := filepath.Join(
		root,
		"server",
		"internal",
		"workflow",
		"schema",
		fmt.Sprintf("%s.schema.json", definition.WorkflowName),
	)
	serverSchema := loadGeneratedSchema(t, serverSchemaPath)
	require.Equal(t, expectedID, serverSchema.ID)
	require.Equal(t, definition.WorkflowName, serverSchema.Workflow)
	_, hasServerAPIVersion := serverSchema.Properties["apiVersion"]
	_, hasServerSchemaVersion := serverSchema.Properties["schemaVersion"]
	require.False(t, hasServerAPIVersion)
	require.False(t, hasServerSchemaVersion)

	docsPath := filepath.Join(root, "docs", "config-reference", definition.WorkflowName+".md")
	docsBytes, err := os.ReadFile(docsPath)
	require.NoError(t, err)
	docs := string(docsBytes)
	require.Contains(t, docs, fmt.Sprintf("<!-- Source: %s -->", ContractSourcePath))
	require.Contains(t, docs, fmt.Sprintf("- Workflow: `%s`", definition.WorkflowName))
	require.NotContains(t, docs, "API Version")
	require.NotContains(t, docs, "Schema Version")
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
