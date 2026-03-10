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
	Dir        string
	WorkflowID string
}

type schemaIdentity struct {
	ID string `json:"$id"`
}

type manifestIdentity struct {
	WorkflowID string `json:"workflowId"`
	Executor   struct {
		Type string `json:"type"`
		Ref  string `json:"ref"`
	} `json:"executor"`
}

func TestWorkflowSchemaAndManifestConsistency(t *testing.T) {
	contracts := listWorkflowContracts()
	root := repoRoot(t)

	for _, contract := range contracts {
		contract := contract
		t.Run(contract.Dir, func(t *testing.T) {
			require.NotEmpty(t, contract.WorkflowID)
			expectedSchemaID := fmt.Sprintf("lunafox://schemas/workflows/%s", contract.WorkflowID)

			serverSchema := loadSchemaIdentity(t, filepath.Join(root, "server", "internal", "workflow", "schema", fmt.Sprintf("%s.schema.json", contract.WorkflowID)))
			require.Equal(t, expectedSchemaID, serverSchema.ID)

			serverManifest := loadManifestIdentity(t, filepath.Join(root, "server", "internal", "workflow", "manifest", fmt.Sprintf("%s.manifest.json", contract.WorkflowID)))
			require.Equal(t, contract.WorkflowID, serverManifest.WorkflowID)
			require.Equal(t, "builtin", serverManifest.Executor.Type)
			require.Equal(t, contract.WorkflowID, serverManifest.Executor.Ref)
		})
	}
}

func TestWorkflowCatalogConsistencyBetweenWorkerAndServer(t *testing.T) {
	root := repoRoot(t)
	workerSet := map[string]struct{}{}
	for _, contract := range listWorkflowContracts() {
		workerSet[strings.TrimSpace(contract.WorkflowID)] = struct{}{}
	}

	serverSet := listServerManifestWorkflows(t, root)
	require.NotEmpty(t, workerSet)
	require.NotEmpty(t, serverSet)

	for workflowID := range workerSet {
		_, ok := serverSet[workflowID]
		require.Truef(t, ok, "worker workflow %q must exist in server manifest catalog", workflowID)
	}
	for workflowID := range serverSet {
		_, ok := workerSet[workflowID]
		require.Truef(t, ok, "server workflow %q must exist in worker registry contracts", workflowID)
	}
}

func listWorkflowContracts() []workflowContract {
	contracts := workflow.ListContracts()
	out := make([]workflowContract, 0, len(contracts))
	for _, definition := range contracts {
		out = append(out, workflowContract{Dir: definition.WorkflowID, WorkflowID: definition.WorkflowID})
	}
	return out
}

func loadSchemaIdentity(t *testing.T, path string) schemaIdentity {
	t.Helper()
	payload, err := os.ReadFile(path)
	require.NoErrorf(t, err, "schema file not found: %s", path)
	var identity schemaIdentity
	require.NoError(t, json.Unmarshal(payload, &identity))
	return identity
}

func loadManifestIdentity(t *testing.T, path string) manifestIdentity {
	t.Helper()
	payload, err := os.ReadFile(path)
	require.NoErrorf(t, err, "manifest file not found: %s", path)
	var identity manifestIdentity
	require.NoError(t, json.Unmarshal(payload, &identity))
	return identity
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Clean(filepath.Join(wd, "..", "..", ".."))
}

func listServerManifestWorkflows(t *testing.T, root string) map[string]struct{} {
	t.Helper()
	pattern := filepath.Join(root, "server", "internal", "workflow", "manifest", "*.manifest.json")
	paths, err := filepath.Glob(pattern)
	require.NoError(t, err)
	set := map[string]struct{}{}
	for _, path := range paths {
		manifest := loadManifestIdentity(t, path)
		name := strings.TrimSpace(manifest.WorkflowID)
		require.NotEmptyf(t, name, "server manifest missing workflowId: %s", path)
		set[name] = struct{}{}
	}
	return set
}
