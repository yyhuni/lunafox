package workflow

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yyhuni/lunafox/worker/internal/server"
)

type mockWorkflow struct {
	name    string
	workDir string
}

func (m *mockWorkflow) Execute(params *Params) (*Output, error) { return &Output{}, nil }
func (m *mockWorkflow) Name() string                            { return m.name }
func (m *mockWorkflow) SaveResults(ctx context.Context, client server.ServerClient, params *Params, output *Output) error {
	return nil
}

func snapshotRegistry() map[string]registration {
	mu.RLock()
	defer mu.RUnlock()
	clone := make(map[string]registration, len(registry))
	for k, v := range registry {
		clone[k] = v
	}
	return clone
}

func restoreRegistry(snapshot map[string]registration) {
	mu.Lock()
	defer mu.Unlock()
	registry = snapshot
}

func validContract(name string) ContractDefinition {
	return ContractDefinition{
		WorkflowName:  name,
		DisplayName:   name,
		Description:   "test contract",
		APIVersion:    "v1",
		SchemaVersion: "1.0.0",
		TargetTypes:   []string{"domain"},
		Stages: []ContractStageDefinition{
			{
				ID:          "stage-1",
				Name:        "Stage 1",
				Description: "test stage",
				Required:    true,
				Parallel:    true,
				Tools: []ContractToolDefinition{
					{
						ID:          "tool-1",
						Description: "test tool",
						Params: []ContractParamDefinition{
							{
								Key:                 "threads-cli",
								Type:                "integer",
								Description:         "threads",
								RequiredWhenEnabled: true,
							},
						},
					},
				},
			},
		},
	}
}

func TestRegisterGetListExists(t *testing.T) {
	original := snapshotRegistry()
	restoreRegistry(make(map[string]registration))
	t.Cleanup(func() { restoreRegistry(original) })

	Register(Registration{
		Name: "test-workflow",
		Factory: func(workDir string) Workflow {
			return &mockWorkflow{name: "test-workflow", workDir: workDir}
		},
		Contract: validContract("test-workflow"),
	})

	assert.True(t, Exists("test-workflow"))
	assert.False(t, Exists("missing"))

	got := Get("test-workflow", "/tmp")
	require.NotNil(t, got)
	assert.Equal(t, "test-workflow", got.Name())

	missing := Get("missing", "/tmp")
	assert.Nil(t, missing)

	names := List()
	assert.Contains(t, names, "test-workflow")

	contract, err := GetContract("test-workflow")
	require.NoError(t, err)
	assert.Equal(t, "test-workflow", contract.WorkflowName)

	contracts := ListContracts()
	require.Len(t, contracts, 1)
	assert.Equal(t, "test-workflow", contracts[0].WorkflowName)
}

func TestRegisterDuplicatePanics(t *testing.T) {
	original := snapshotRegistry()
	restoreRegistry(make(map[string]registration))
	t.Cleanup(func() { restoreRegistry(original) })

	reg := Registration{
		Name: "dup",
		Factory: func(workDir string) Workflow {
			return &mockWorkflow{name: "dup"}
		},
		Contract: validContract("dup"),
	}

	Register(reg)
	require.Panics(t, func() {
		Register(reg)
	})
}

func TestRegisterDecodeConfig(t *testing.T) {
	original := snapshotRegistry()
	restoreRegistry(make(map[string]registration))
	t.Cleanup(func() { restoreRegistry(original) })

	Register(Registration{
		Name: "typed-workflow",
		Factory: func(workDir string) Workflow {
			return &mockWorkflow{name: "typed-workflow", workDir: workDir}
		},
		ConfigDecoder: func(scanConfig map[string]any) (any, error) {
			return map[string]any{"decoded": scanConfig["raw"]}, nil
		},
		Contract: validContract("typed-workflow"),
	})

	cfg, err := DecodeConfig("typed-workflow", map[string]any{"raw": "v"})
	require.NoError(t, err)
	require.NotNil(t, cfg)
	decoded, ok := cfg.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "v", decoded["decoded"])
}

func TestDecodeConfigNoDecoderReturnsNil(t *testing.T) {
	original := snapshotRegistry()
	restoreRegistry(make(map[string]registration))
	t.Cleanup(func() { restoreRegistry(original) })

	Register(Registration{
		Name: "plain-workflow",
		Factory: func(workDir string) Workflow {
			return &mockWorkflow{name: "plain-workflow", workDir: workDir}
		},
		Contract: validContract("plain-workflow"),
	})

	cfg, err := DecodeConfig("plain-workflow", map[string]any{"raw": "v"})
	require.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestDecodeConfigUnknownWorkflow(t *testing.T) {
	original := snapshotRegistry()
	restoreRegistry(make(map[string]registration))
	t.Cleanup(func() { restoreRegistry(original) })

	_, err := DecodeConfig("missing-workflow", map[string]any{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not registered")
}

func TestGetContractUnknownWorkflow(t *testing.T) {
	original := snapshotRegistry()
	restoreRegistry(make(map[string]registration))
	t.Cleanup(func() { restoreRegistry(original) })

	_, err := GetContract("missing-workflow")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not registered")
}

func TestRegisterMissingContractPanics(t *testing.T) {
	original := snapshotRegistry()
	restoreRegistry(make(map[string]registration))
	t.Cleanup(func() { restoreRegistry(original) })

	require.Panics(t, func() {
		Register(Registration{
			Name: "broken",
			Factory: func(workDir string) Workflow {
				return &mockWorkflow{name: "broken", workDir: workDir}
			},
			Contract: ContractDefinition{},
		})
	})
}

func TestRegisterContractNameMismatchPanics(t *testing.T) {
	original := snapshotRegistry()
	restoreRegistry(make(map[string]registration))
	t.Cleanup(func() { restoreRegistry(original) })

	require.Panics(t, func() {
		Register(Registration{
			Name: "foo",
			Factory: func(workDir string) Workflow {
				return &mockWorkflow{name: "foo", workDir: workDir}
			},
			Contract: validContract("bar"),
		})
	})
}

func TestRegisterInvalidContractAPIVersionPanics(t *testing.T) {
	original := snapshotRegistry()
	restoreRegistry(make(map[string]registration))
	t.Cleanup(func() { restoreRegistry(original) })

	def := validContract("bad-api")
	def.APIVersion = "version1"
	require.Panics(t, func() {
		Register(Registration{
			Name: "bad-api",
			Factory: func(workDir string) Workflow {
				return &mockWorkflow{name: "bad-api", workDir: workDir}
			},
			Contract: def,
		})
	})
}

func TestRegisterInvalidContractSchemaVersionPanics(t *testing.T) {
	original := snapshotRegistry()
	restoreRegistry(make(map[string]registration))
	t.Cleanup(func() { restoreRegistry(original) })

	def := validContract("bad-schema")
	def.SchemaVersion = "1.0"
	require.Panics(t, func() {
		Register(Registration{
			Name: "bad-schema",
			Factory: func(workDir string) Workflow {
				return &mockWorkflow{name: "bad-schema", workDir: workDir}
			},
			Contract: def,
		})
	})
}

func TestRegisterInvalidContractDuplicateStagePanics(t *testing.T) {
	original := snapshotRegistry()
	restoreRegistry(make(map[string]registration))
	t.Cleanup(func() { restoreRegistry(original) })

	def := validContract("bad-stage")
	def.Stages = append(def.Stages, def.Stages[0])
	require.Panics(t, func() {
		Register(Registration{
			Name: "bad-stage",
			Factory: func(workDir string) Workflow {
				return &mockWorkflow{name: "bad-stage", workDir: workDir}
			},
			Contract: def,
		})
	})
}
