package workflow_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yyhuni/lunafox/worker/internal/workflow"
	_ "github.com/yyhuni/lunafox/worker/internal/workflow/all"
)

type workflowContract struct {
	Dir          string
	WorkflowName string
}

type schemaIdentity struct {
	ID       string `json:"$id"`
	Workflow string `json:"x-workflow"`
}

func TestWorkflowSchemaConsistency(t *testing.T) {
	contracts := listWorkflowContracts()
	root := repoRoot(t)

	for _, contract := range contracts {
		contract := contract

		t.Run(contract.Dir, func(t *testing.T) {
			require.NotEmpty(t, contract.WorkflowName, "workflow name is required")

			workerSchemaPath := filepath.Join(
				contract.Dir,
				"generated",
				fmt.Sprintf("%s.schema.json", contract.WorkflowName),
			)
			workerSchema := loadSchemaIdentity(t, workerSchemaPath)
			require.Equal(
				t,
				contract.WorkflowName,
				workerSchema.Workflow,
				"worker generated schema x-workflow should match workflow name",
			)
			expectedID := fmt.Sprintf("lunafox://schemas/workflows/%s", contract.WorkflowName)
			require.Equal(
				t,
				expectedID,
				workerSchema.ID,
				"worker generated schema $id should include workflow name",
			)

			serverSchemaPath := filepath.Join(
				root,
				"server",
				"internal",
				"workflow",
				"schema",
				fmt.Sprintf("%s.schema.json", contract.WorkflowName),
			)
			serverSchema := loadSchemaIdentity(t, serverSchemaPath)
			require.Equal(
				t,
				contract.WorkflowName,
				serverSchema.Workflow,
				"server schema x-workflow should match workflow name",
			)
			require.Equal(
				t,
				expectedID,
				serverSchema.ID,
				"server schema $id should include workflow name",
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

	serverSet := listServerSchemaWorkflows(t, root)
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
			Dir:          definition.WorkflowName,
			WorkflowName: definition.WorkflowName,
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

func listServerSchemaWorkflows(t *testing.T, root string) map[string]struct{} {
	t.Helper()
	pattern := filepath.Join(root, "server", "internal", "workflow", "schema", "*.schema.json")
	paths, err := filepath.Glob(pattern)
	require.NoError(t, err)

	set := map[string]struct{}{}
	for _, path := range paths {
		schema := loadSchemaIdentity(t, path)
		name := strings.TrimSpace(schema.Workflow)
		require.NotEmptyf(t, name, "server schema missing x-workflow: %s", path)
		set[name] = struct{}{}
	}
	return set
}
