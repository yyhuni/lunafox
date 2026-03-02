package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yyhuni/lunafox/worker/internal/workflow"
)

func TestLoadDefinitionFromRegistry(t *testing.T) {
	def, err := loadDefinition("subdomain_discovery")
	require.NoError(t, err)
	require.Equal(t, "subdomain_discovery", def.WorkflowName)
	require.Equal(t, "v1", def.APIVersion)
	require.Equal(t, "1.0.0", def.SchemaVersion)
}

func TestLoadDefinitionUnknownWorkflow(t *testing.T) {
	_, err := loadDefinition("missing_workflow")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported workflow")
}

func TestBuildTypedGo(t *testing.T) {
	def, err := loadDefinition("subdomain_discovery")
	require.NoError(t, err)

	output := buildTypedGo(def, "subdomain_discovery")
	require.True(t, strings.Contains(output, "type WorkflowConfig struct"))
	require.True(t, strings.Contains(output, "`json:\"apiVersion\"`"))
	require.True(t, strings.Contains(output, "type BruteforceSubdomainBruteforceToolConfig struct"))
}

func TestResolveOutputPathsFromDirs(t *testing.T) {
	def := workflowDefForTest()
	opts := genOptions{
		workerSchemaDir: "worker/internal/workflow/subdomain_discovery/generated",
		serverSchemaDir: "server/internal/engineschema",
		docsDir:         "docs/config-reference",
	}

	err := resolveOutputPaths(def, &opts)
	require.NoError(t, err)
	require.Equal(
		t,
		filepath.Join("worker/internal/workflow/subdomain_discovery/generated", "subdomain_discovery-v1-1.0.0.schema.json"),
		opts.workerSchemaPath,
	)
	require.Equal(
		t,
		filepath.Join("server/internal/engineschema", "subdomain_discovery-v1-1.0.0.schema.json"),
		opts.serverSchemaPath,
	)
	require.Equal(
		t,
		filepath.Join("docs/config-reference", "subdomain_discovery.md"),
		opts.docsPath,
	)
}

func TestResolveOutputPathsRequiresServerPathOrDir(t *testing.T) {
	def := workflowDefForTest()
	opts := genOptions{
		workerSchemaDir: "worker/internal/workflow/subdomain_discovery/generated",
		docsDir:         "docs/config-reference",
	}

	err := resolveOutputPaths(def, &opts)
	require.Error(t, err)
	require.Contains(t, err.Error(), "server-schema-output")
}

func TestResolveOutputPathsRequiresDocsPathOrDir(t *testing.T) {
	def := workflowDefForTest()
	opts := genOptions{
		workerSchemaDir: "worker/internal/workflow/subdomain_discovery/generated",
		serverSchemaDir: "server/internal/engineschema",
	}

	err := resolveOutputPaths(def, &opts)
	require.Error(t, err)
	require.Contains(t, err.Error(), "docs-output")
}

func TestBuildTypedGoToolTypeNamesIncludeStage(t *testing.T) {
	def := workflow.ContractDefinition{
		WorkflowName:  "demo",
		APIVersion:    "v1",
		SchemaVersion: "1.0.0",
		Stages: []workflow.ContractStageDefinition{
			{
				ID: "stage-a",
				Tools: []workflow.ContractToolDefinition{
					{ID: "same-tool"},
				},
			},
			{
				ID: "stage-b",
				Tools: []workflow.ContractToolDefinition{
					{ID: "same-tool"},
				},
			},
		},
	}

	output := buildTypedGo(def, "demo")
	require.Contains(t, output, "type StageASameToolToolConfig struct")
	require.Contains(t, output, "type StageBSameToolToolConfig struct")
}

func workflowDefForTest() workflow.ContractDefinition {
	def, err := loadDefinition("subdomain_discovery")
	if err != nil {
		panic(err)
	}
	return def
}
