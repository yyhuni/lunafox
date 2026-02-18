package workflow

import (
	"context"
	"testing"

	"github.com/yyhuni/lunafox/worker/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func snapshotRegistry() map[string]Factory {
	mu.RLock()
	defer mu.RUnlock()
	clone := make(map[string]Factory, len(registry))
	for k, v := range registry {
		clone[k] = v
	}
	return clone
}

func restoreRegistry(snapshot map[string]Factory) {
	mu.Lock()
	defer mu.Unlock()
	registry = snapshot
}

func TestRegisterGetListExists(t *testing.T) {
	original := snapshotRegistry()
	restoreRegistry(make(map[string]Factory))
	t.Cleanup(func() { restoreRegistry(original) })

	Register("test-workflow", func(workDir string) Workflow {
		return &mockWorkflow{name: "test-workflow", workDir: workDir}
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
}

func TestRegisterDuplicatePanics(t *testing.T) {
	original := snapshotRegistry()
	restoreRegistry(make(map[string]Factory))
	t.Cleanup(func() { restoreRegistry(original) })

	Register("dup", func(workDir string) Workflow { return &mockWorkflow{name: "dup"} })
	require.Panics(t, func() {
		Register("dup", func(workDir string) Workflow { return &mockWorkflow{name: "dup"} })
	})
}
