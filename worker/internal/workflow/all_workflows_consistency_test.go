package workflow_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAllWorkflowsImportFileMatchesWorkflowDirectories(t *testing.T) {
	root := workerRoot(t)
	allPath := filepath.Join(root, "internal", "workflow", "all", "all_workflows_gen.go")
	content, err := os.ReadFile(allPath)
	require.NoError(t, err)
	text := string(content)

	entries, err := os.ReadDir(filepath.Join(root, "internal", "workflow"))
	require.NoError(t, err)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "all" {
			continue
		}
		workflowFile := filepath.Join(root, "internal", "workflow", name, "workflow.go")
		if _, err := os.Stat(workflowFile); err != nil {
			continue
		}

		importPath := "github.com/yyhuni/lunafox/worker/internal/workflow/" + name
		require.Truef(
			t,
			strings.Contains(text, importPath),
			"all_workflows_gen.go missing import for workflow %s",
			name,
		)
	}
}
