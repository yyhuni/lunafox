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
		WorkflowID:  name,
		DisplayName: name,
		Description: "test contract",
		TargetTypes: []string{"domain"},
		Executor: ContractExecutorBinding{
			Type: "builtin",
			Ref:  name,
		},
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

func TestRegisterMissingExecutorBindingPanics(t *testing.T) {
	original := snapshotRegistry()
	restoreRegistry(make(map[string]registration))
	t.Cleanup(func() { restoreRegistry(original) })

	require.Panics(t, func() {
		Register(Registration{
			Name: "broken",
			Factory: func(workDir string) Workflow {
				return &mockWorkflow{name: "broken", workDir: workDir}
			},
			Contract: ContractDefinition{
				WorkflowID:  "broken",
				DisplayName: "broken",
				Stages: []ContractStageDefinition{{
					ID:       "stage-1",
					Name:     "Stage 1",
					Required: true,
					Tools:    []ContractToolDefinition{{ID: "tool-1"}},
				}},
			},
		})
	})
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
	assert.Equal(t, "test-workflow", contract.WorkflowID)

	contracts := ListContracts()
	require.Len(t, contracts, 1)
	assert.Equal(t, "test-workflow", contracts[0].WorkflowID)
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

func TestResolveExecutorBindingReturnsBuiltinBinding(t *testing.T) {
	original := snapshotRegistry()
	restoreRegistry(make(map[string]registration))
	t.Cleanup(func() { restoreRegistry(original) })

	Register(Registration{
		Name: "bound-workflow",
		Factory: func(workDir string) Workflow {
			return &mockWorkflow{name: "bound-workflow", workDir: workDir}
		},
		Contract: validContract("bound-workflow"),
	})

	binding, err := ResolveExecutorBinding("bound-workflow")
	require.NoError(t, err)
	assert.Equal(t, "builtin", binding.Type)
	assert.Equal(t, "bound-workflow", binding.Ref)
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

func TestRegisterInvalidContractMinimumGreaterThanMaximumPanics(t *testing.T) {
	original := snapshotRegistry()
	restoreRegistry(make(map[string]registration))
	t.Cleanup(func() { restoreRegistry(original) })

	def := validContract("bad-min-max")
	minimum := 10
	maximum := 1
	def.Stages[0].Tools[0].Params[0].Minimum = &minimum
	def.Stages[0].Tools[0].Params[0].Maximum = &maximum

	require.Panics(t, func() {
		Register(Registration{
			Name: "bad-min-max",
			Factory: func(workDir string) Workflow {
				return &mockWorkflow{name: "bad-min-max", workDir: workDir}
			},
			Contract: def,
		})
	})
}

func TestRegisterInvalidContractStringConstraintOnIntegerPanics(t *testing.T) {
	original := snapshotRegistry()
	restoreRegistry(make(map[string]registration))
	t.Cleanup(func() { restoreRegistry(original) })

	def := validContract("bad-integer-pattern")
	def.Stages[0].Tools[0].Params[0].Pattern = "^[0-9]+$"

	require.Panics(t, func() {
		Register(Registration{
			Name: "bad-integer-pattern",
			Factory: func(workDir string) Workflow {
				return &mockWorkflow{name: "bad-integer-pattern", workDir: workDir}
			},
			Contract: def,
		})
	})
}
