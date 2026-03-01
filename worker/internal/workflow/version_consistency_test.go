package workflow_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
	subdomain "github.com/yyhuni/lunafox/worker/internal/workflow/subdomain_discovery"
)

var workflowVersionRegex = regexp.MustCompile(`^\d+\.\d+\.\d+([\-+][0-9A-Za-z.-]+)?$`)
var workflowAPIVersionRegex = regexp.MustCompile(`^v[0-9]+$`)

type workflowContract struct {
	Dir           string
	WorkflowName  string
	APIVersion    string
	SchemaVersion string
}

type schemaIdentity struct {
	ID            string `json:"$id"`
	Engine        string `json:"x-engine"`
	EngineVersion string `json:"x-engine-version"`
}

func TestWorkflowVersionConsistency(t *testing.T) {
	contracts := listWorkflowContracts()
	root := repoRoot(t)

	for _, contract := range contracts {
		contract := contract

		t.Run(contract.Dir, func(t *testing.T) {
			require.NotEmpty(t, contract.WorkflowName, "workflow name is required")
			require.NotEmpty(t, contract.APIVersion, "workflow api version is required")
			require.NotEmpty(t, contract.SchemaVersion, "workflow schema version is required")
			require.Truef(
				t,
				workflowAPIVersionRegex.MatchString(contract.APIVersion),
				"workflow api version must match v<major>, got %q",
				contract.APIVersion,
			)
			require.Truef(
				t,
				workflowVersionRegex.MatchString(contract.SchemaVersion),
				"workflow schema version must be semver (e.g. 1.0.0), got %q",
				contract.SchemaVersion,
			)

			workerSchemaPath := filepath.Join(contract.Dir, "schema_generated.json")
			workerSchema := loadSchemaIdentity(t, workerSchemaPath)
			require.Equal(
				t,
				contract.WorkflowName,
				workerSchema.Engine,
				"schema_generated.json x-engine should match workflow name",
			)
			require.Equal(
				t,
				contract.SchemaVersion,
				workerSchema.EngineVersion,
				"schema_generated.json x-engine-version should match schema version",
			)
			expectedID := fmt.Sprintf(
				"lunafox://schemas/engines/%s/%s",
				contract.WorkflowName,
				contract.SchemaVersion,
			)
			require.Equal(
				t,
				expectedID,
				workerSchema.ID,
				"schema_generated.json $id should include workflow name and schema version",
			)

			serverSchemaPath := filepath.Join(
				root,
				"server",
				"internal",
				"engineschema",
				fmt.Sprintf("%s-%s-%s.schema.json", contract.WorkflowName, contract.APIVersion, contract.SchemaVersion),
			)
			serverSchema := loadSchemaIdentity(t, serverSchemaPath)
			require.Equal(
				t,
				contract.WorkflowName,
				serverSchema.Engine,
				"server schema x-engine should match workflow name",
			)
			require.Equal(
				t,
				contract.SchemaVersion,
				serverSchema.EngineVersion,
				"server schema x-engine-version should match schema version",
			)
			require.Equal(
				t,
				expectedID,
				serverSchema.ID,
				"server schema $id should include workflow name and schema version",
			)
		})
	}
}

func listWorkflowContracts() []workflowContract {
	definition := subdomain.GetContractDefinition()
	return []workflowContract{
		{
			Dir:           "subdomain_discovery",
			WorkflowName:  definition.WorkflowName,
			APIVersion:    definition.APIVersion,
			SchemaVersion: definition.SchemaVersion,
		},
	}
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
