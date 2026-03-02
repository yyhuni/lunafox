package workflow_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yyhuni/lunafox/worker/internal/workflow"
	_ "github.com/yyhuni/lunafox/worker/internal/workflow/all"
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

			workerSchemaPath := filepath.Join(
				contract.Dir,
				"generated",
				fmt.Sprintf("%s-%s-%s.schema.json", contract.WorkflowName, contract.APIVersion, contract.SchemaVersion),
			)
			workerSchema := loadSchemaIdentity(t, workerSchemaPath)
			require.Equal(
				t,
				contract.WorkflowName,
				workerSchema.Engine,
				"worker generated schema x-engine should match workflow name",
			)
			require.Equal(
				t,
				contract.SchemaVersion,
				workerSchema.EngineVersion,
				"worker generated schema x-engine-version should match schema version",
			)
			expectedID := fmt.Sprintf(
				"lunafox://schemas/engines/%s/%s/%s",
				contract.WorkflowName,
				contract.APIVersion,
				contract.SchemaVersion,
			)
			require.Equal(
				t,
				expectedID,
				workerSchema.ID,
				"worker generated schema $id should include workflow name/apiVersion/schemaVersion",
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
				"server schema $id should include workflow name/apiVersion/schemaVersion",
			)
		})
	}
}

func TestWorkflowCatalogConsistencyBetweenWorkerAndServer(t *testing.T) {
	root := repoRoot(t)
	workerSet := map[string]struct{}{}
	for _, contract := range listWorkflowContracts() {
		workerSet[strings.TrimSpace(contract.WorkflowName)] = struct{}{}
	}

	serverSet := listServerSchemaEngines(t, root)
	require.NotEmpty(t, workerSet, "worker contracts must not be empty")
	require.NotEmpty(t, serverSet, "server schema catalog must not be empty")

	for workflowName := range workerSet {
		_, ok := serverSet[workflowName]
		require.Truef(
			t,
			ok,
			"worker workflow %q must exist in server schema catalog",
			workflowName,
		)
	}
	for workflowName := range serverSet {
		_, ok := workerSet[workflowName]
		require.Truef(
			t,
			ok,
			"server schema workflow %q must exist in worker registry contracts",
			workflowName,
		)
	}
}

func listWorkflowContracts() []workflowContract {
	contracts := workflow.ListContracts()
	out := make([]workflowContract, 0, len(contracts))
	for _, definition := range contracts {
		out = append(out, workflowContract{
			Dir:           definition.WorkflowName,
			WorkflowName:  definition.WorkflowName,
			APIVersion:    definition.APIVersion,
			SchemaVersion: definition.SchemaVersion,
		})
	}
	return out
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

func listServerSchemaEngines(t *testing.T, root string) map[string]struct{} {
	t.Helper()
	pattern := filepath.Join(root, "server", "internal", "engineschema", "*.schema.json")
	paths, err := filepath.Glob(pattern)
	require.NoError(t, err)

	set := map[string]struct{}{}
	for _, path := range paths {
		schema := loadSchemaIdentity(t, path)
		name := strings.TrimSpace(schema.Engine)
		require.NotEmptyf(t, name, "server schema missing x-engine: %s", path)
		set[name] = struct{}{}
	}
	return set
}
