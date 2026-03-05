package main

import (
	"encoding/json"
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
	require.True(t, strings.Contains(output, "type BruteforceSubdomainBruteforceToolConfig struct"))
}

func TestResolveOutputPathsFromDirs(t *testing.T) {
	def := workflowDefForTest()
	opts := genOptions{
		workerSchemaDir:  "worker/internal/workflow/subdomain_discovery/generated",
		serverSchemaDir:  "server/internal/workflow/schema",
		serverProfileDir: "server/internal/workflow/profile/profiles",
		docsDir:          "docs/config-reference",
	}

	err := resolveOutputPaths(def, &opts)
	require.NoError(t, err)
	require.Equal(
		t,
		filepath.Join("worker/internal/workflow/subdomain_discovery/generated", "subdomain_discovery.schema.json"),
		opts.workerSchemaPath,
	)
	require.Equal(
		t,
		filepath.Join("server/internal/workflow/schema", "subdomain_discovery.schema.json"),
		opts.serverSchemaPath,
	)
	require.Equal(
		t,
		filepath.Join("server/internal/workflow/profile/profiles", "subdomain_discovery.yaml"),
		opts.serverProfilePath,
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
		serverSchemaDir: "server/internal/workflow/schema",
	}

	err := resolveOutputPaths(def, &opts)
	require.Error(t, err)
	require.Contains(t, err.Error(), "docs-output")
}

func TestBuildTypedGoToolTypeNamesIncludeStage(t *testing.T) {
	def := workflow.ContractDefinition{
		WorkflowName: "demo",
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

func TestBuildSchemaIncludesParamConstraints(t *testing.T) {
	minimum := 1
	maximum := 64
	minLength := 1
	maxLength := 128

	def := workflow.ContractDefinition{
		WorkflowName: "demo",
		DisplayName:  "Demo",
		Description:  "Demo workflow",
		TargetTypes:  []string{"domain"},
		Stages: []workflow.ContractStageDefinition{
			{
				ID:          "recon",
				Name:        "Recon",
				Description: "Recon stage",
				Required:    true,
				Parallel:    true,
				Tools: []workflow.ContractToolDefinition{
					{
						ID:          "tool-a",
						Description: "Tool A",
						Params: []workflow.ContractParamDefinition{
							{
								Key:                 "threads-cli",
								Type:                "integer",
								Description:         "threads",
								RequiredWhenEnabled: true,
								Minimum:             &minimum,
								Maximum:             &maximum,
							},
							{
								Key:                 "wordlist-name-runtime",
								Type:                "string",
								Description:         "wordlist name",
								RequiredWhenEnabled: true,
								MinLength:           &minLength,
								MaxLength:           &maxLength,
								Pattern:             "^[a-z0-9_\\-]+$",
								Enum:                []string{"common", "large"},
							},
						},
					},
				},
			},
		},
	}

	schema := buildSchema(def)
	toolSchema := schema.Properties["recon"].Properties["tools"].Properties["tool-a"]

	intProp := toolSchema.Properties["threads-cli"]
	require.NotNil(t, intProp)
	require.NotNil(t, intProp.Minimum)
	require.NotNil(t, intProp.Maximum)
	require.Equal(t, minimum, *intProp.Minimum)
	require.Equal(t, maximum, *intProp.Maximum)

	stringProp := toolSchema.Properties["wordlist-name-runtime"]
	require.NotNil(t, stringProp)
	require.NotNil(t, stringProp.MinLength)
	require.NotNil(t, stringProp.MaxLength)
	require.Equal(t, minLength, *stringProp.MinLength)
	require.Equal(t, maxLength, *stringProp.MaxLength)
	require.Equal(t, "^[a-z0-9_\\-]+$", stringProp.Pattern)
	require.Equal(t, []string{"common", "large"}, stringProp.Enum)
}

func TestBuildSchemaUsesDisplayNameForMetadataName(t *testing.T) {
	def := workflow.ContractDefinition{
		WorkflowName: "demo_key",
		DisplayName:  "Demo Workflow",
		Description:  "Demo workflow",
		TargetTypes:  []string{"domain"},
		Stages: []workflow.ContractStageDefinition{
			{
				ID:          "recon",
				Name:        "Recon",
				Description: "Recon stage",
				Required:    true,
				Parallel:    true,
				Tools: []workflow.ContractToolDefinition{
					{
						ID:          "tool-a",
						Description: "Tool A",
					},
				},
			},
		},
	}

	schema := buildSchema(def)
	metadataName, ok := schema.Metadata["name"].(string)
	require.True(t, ok)
	require.Equal(t, "Demo Workflow", metadataName)
}

func TestBuildProfileYAMLFromContractDefaults(t *testing.T) {
	minimum := 1
	def := workflow.ContractDefinition{
		WorkflowName: "demo",
		DisplayName:  "Demo Workflow",
		Description:  "Demo workflow description",
		DefaultProfile: workflow.ContractProfileDefinition{
			ID:          "demo_default",
			Name:        "Demo 默认配置",
			Description: "Demo 默认 profile",
		},
		Stages: []workflow.ContractStageDefinition{
			{
				ID:          "recon",
				Name:        "Recon",
				Description: "Recon stage",
				Required:    true,
				Parallel:    true,
				Tools: []workflow.ContractToolDefinition{
					{
						ID:          "tool-a",
						Description: "Tool A",
						Params: []workflow.ContractParamDefinition{
							{
								Key:                 "threads-cli",
								Type:                "integer",
								Description:         "threads",
								RequiredWhenEnabled: true,
								Minimum:             &minimum,
								Default:             20,
							},
							{
								Key:                 "wordlist-runtime",
								Type:                "string",
								Description:         "wordlist",
								RequiredWhenEnabled: true,
								Default:             "top1m.txt",
							},
						},
					},
				},
			},
		},
	}

	schema := buildSchema(def)
	schemaJSON, err := json.Marshal(schema)
	require.NoError(t, err)

	payload, err := buildProfileYAML(def, schemaJSON)
	require.NoError(t, err)

	text := string(payload)
	require.Contains(t, text, "id: demo_default")
	require.Contains(t, text, "name: Demo 默认配置")
	require.Contains(t, text, "description: Demo 默认 profile")
	require.Contains(t, text, "configuration:")
	require.Contains(t, text, "demo:")
	require.Contains(t, text, "threads-cli: 20")
	require.Contains(t, text, "wordlist-runtime: top1m.txt")
}

func TestBuildProfileYAMLFailsWhenRequiredDefaultMissing(t *testing.T) {
	minimum := 1
	def := workflow.ContractDefinition{
		WorkflowName: "demo",
		DisplayName:  "Demo Workflow",
		Description:  "Demo workflow description",
		Stages: []workflow.ContractStageDefinition{
			{
				ID:          "recon",
				Name:        "Recon",
				Description: "Recon stage",
				Required:    true,
				Parallel:    true,
				Tools: []workflow.ContractToolDefinition{
					{
						ID:          "tool-a",
						Description: "Tool A",
						Params: []workflow.ContractParamDefinition{
							{
								Key:                 "threads-cli",
								Type:                "integer",
								Description:         "threads",
								RequiredWhenEnabled: true,
								Minimum:             &minimum,
							},
						},
					},
				},
			},
		},
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
