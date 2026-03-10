package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yyhuni/lunafox/worker/internal/workflow"
	"gopkg.in/yaml.v3"
)

func TestLoadDefinition(t *testing.T) {
	def, err := loadDefinition("subdomain_discovery")
	require.NoError(t, err)
	require.Equal(t, "subdomain_discovery", def.WorkflowID)
}

func TestResolveOutputPathsFromDirs(t *testing.T) {
	def := workflowDefForTest()
	opts := genOptions{
		workflow:          def.WorkflowID,
		serverSchemaDir:   "/tmp/server-schema",
		serverManifestDir: "/tmp/server-manifest",
		serverProfileDir:  "/tmp/profiles",
		docsDir:           "/tmp/docs",
	}

	require.NoError(t, resolveOutputPaths(def, &opts))
	require.Equal(t, "/tmp/server-schema/subdomain_discovery.schema.json", opts.serverSchemaPath)
	require.Equal(t, "/tmp/server-manifest/subdomain_discovery.manifest.json", opts.serverManifestPath)
	require.Equal(t, "/tmp/profiles/subdomain_discovery.yaml", opts.serverProfilePath)
	require.Equal(t, "/tmp/docs/subdomain_discovery.md", opts.docsPath)
}

func TestResolveOutputPathsRequiresServerManifestPathOrDir(t *testing.T) {
	def := workflowDefForTest()
	opts := genOptions{
		workflow:        def.WorkflowID,
		serverSchemaDir: "/tmp/server-schema",
		docsDir:         "/tmp/docs",
	}
	assertErr := resolveOutputPaths(def, &opts)
	require.Error(t, assertErr)
	require.Contains(t, assertErr.Error(), "server-manifest-output")
}

func TestBuildSchemaOmitsWorkflowMetadataExtensions(t *testing.T) {
	schema := buildSchema(workflowDefForTest())
	payload, err := json.Marshal(schema)
	require.NoError(t, err)
	text := string(payload)
	require.NotContains(t, text, "x-workflow")
	require.NotContains(t, text, "x-metadata")
	require.NotContains(t, text, "x-stage")
}

func TestBuildManifestIncludesExecutorAndStagesOnly(t *testing.T) {
	manifest := buildManifest(workflowDefForTest())
	require.Equal(t, "v1", manifest.ManifestVersion)
	require.Equal(t, "subdomain_discovery", manifest.WorkflowID)
	require.Equal(t, "Subdomain Discovery", manifest.DisplayName)
	require.Equal(t, "builtin", manifest.Executor.Type)
	require.Equal(t, "subdomain_discovery", manifest.Executor.Ref)
	require.NotEmpty(t, manifest.Stages)
	require.Equal(t, "recon", manifest.Stages[0].StageID)
	require.NotEmpty(t, manifest.Stages[0].Tools)
	require.NotEmpty(t, manifest.Stages[0].Tools[0].Params)
}

func TestBuildManifestOmitsConstraintAndDefaultFieldsFromParams(t *testing.T) {
	manifest := buildManifest(workflowDefForTest())
	payload, err := json.Marshal(manifest)
	require.NoError(t, err)
	text := string(payload)
	require.NotContains(t, text, "defaultValue")
	require.NotContains(t, text, "minimum")
	require.NotContains(t, text, "maximum")
	require.NotContains(t, text, "minLength")
	require.NotContains(t, text, "maxLength")
	require.NotContains(t, text, "pattern")
	require.NotContains(t, text, "enum")
	require.Contains(t, text, "requiredWhenEnabled")
	require.Contains(t, text, "valueType")
	require.Contains(t, text, "configKey")
	require.NotContains(t, text, "configSchemaId")
	require.NotContains(t, text, "supportedTargetTypeIds")
	require.NotContains(t, text, "defaultProfileId")
}

func TestBuildTypedGoDisambiguatesDuplicateToolNamesAcrossStages(t *testing.T) {
	def := workflow.ContractDefinition{
		WorkflowID:  "demo",
		DisplayName: "Demo",
		Stages: []workflow.ContractStageDefinition{
			{
				ID:          "stage-a",
				Name:        "Stage A",
				Description: "A",
				Required:    true,
				Tools:       []workflow.ContractToolDefinition{{ID: "same-tool"}},
			},
			{
				ID:          "stage-b",
				Name:        "Stage B",
				Description: "B",
				Required:    false,
				Tools:       []workflow.ContractToolDefinition{{ID: "same-tool"}},
			},
		},
	}
	output := buildTypedGo(def, "demo")
	require.Contains(t, output, "type StageASameToolToolConfig struct")
	require.Contains(t, output, "type StageBSameToolToolConfig struct")
}

func TestBuildProfileYAMLFromContractDefaults(t *testing.T) {
	minimum := 1
	def := workflow.ContractDefinition{
		WorkflowID:  "demo",
		DisplayName: "Demo Workflow",
		Description: "Demo workflow description",
		Stages: []workflow.ContractStageDefinition{
			{
				ID:          "recon",
				Name:        "Recon",
				Description: "Recon stage",
				Required:    true,
				Parallel:    true,
				Tools: []workflow.ContractToolDefinition{{
					ID:          "tool-a",
					Description: "Tool A",
					Params: []workflow.ContractParamDefinition{
						{Key: "threads-cli", Type: "integer", Description: "threads", RequiredWhenEnabled: true, Minimum: &minimum, Default: 20},
						{Key: "wordlist-runtime", Type: "string", Description: "wordlist", RequiredWhenEnabled: true, Default: "top1m.txt"},
					},
				}},
			},
		},
	}

	schema := buildSchema(def)
	schemaJSON, err := json.Marshal(schema)
	require.NoError(t, err)

	payload, err := buildProfileYAML(def, schemaJSON)
	require.NoError(t, err)
	text := string(payload)
	require.True(t, strings.HasPrefix(text, "# Code generated by workflow-contract-gen. DO NOT EDIT.\n"))
	require.Contains(t, text, "id: demo")
	require.Contains(t, text, "name: Demo Workflow")
	require.Contains(t, text, "description: Demo workflow description")
	require.Contains(t, text, "demo:")
	require.Contains(t, text, "threads-cli: 20")
	require.Contains(t, text, "wordlist-runtime: top1m.txt")

	var profileDoc struct {
		Configuration map[string]any `yaml:"configuration"`
	}
	require.NoError(t, yaml.Unmarshal(payload, &profileDoc))
	require.Contains(t, profileDoc.Configuration, "demo")
}

func TestBuildProfileYAMLFailsWhenRequiredDefaultMissing(t *testing.T) {
	minimum := 1
	def := workflow.ContractDefinition{
		WorkflowID:  "demo",
		DisplayName: "Demo Workflow",
		Description: "Demo workflow description",
		Stages: []workflow.ContractStageDefinition{{
			ID:          "recon",
			Name:        "Recon",
			Description: "Recon stage",
			Required:    true,
			Parallel:    true,
			Tools: []workflow.ContractToolDefinition{{
				ID:          "tool-a",
				Description: "Tool A",
				Params: []workflow.ContractParamDefinition{{
					Key:                 "threads-cli",
					Type:                "integer",
					Description:         "threads",
					RequiredWhenEnabled: true,
					Minimum:             &minimum,
				}},
			}},
		}},
	}

	schema := buildSchema(def)
	schemaJSON, err := json.Marshal(schema)
	require.NoError(t, err)
	_, err = buildProfileYAML(def, schemaJSON)
	require.Error(t, err)
	require.Contains(t, err.Error(), "default value is required")
}

func workflowDefForTest() workflow.ContractDefinition {
	def, err := loadDefinition("subdomain_discovery")
	if err != nil {
		panic(err)
	}
	return def
}
