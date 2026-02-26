package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfigSuccess(t *testing.T) {
	t.Setenv("TASK_ID", "99")
	t.Setenv("AGENT_SOCKET", "/run/lunafox/worker-runtime.sock")
	t.Setenv("TASK_TOKEN", "task-token")
	t.Setenv("SCAN_ID", "10")
	t.Setenv("TARGET_ID", "20")
	t.Setenv("TARGET_NAME", "example.com")
	t.Setenv("TARGET_TYPE", "domain")
	t.Setenv("WORKFLOW_NAME", "subdomain_discovery")
	t.Setenv("WORKSPACE_DIR", "/tmp/workspace")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("CONFIG", "threads: 10\nflag: true\n")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, 99, cfg.TaskID)
	assert.Equal(t, "/run/lunafox/worker-runtime.sock", cfg.AgentSocket)
	assert.Equal(t, "task-token", cfg.TaskToken)
	assert.Equal(t, 10, cfg.ScanID)
	assert.Equal(t, 20, cfg.TargetID)
	assert.Equal(t, "example.com", cfg.TargetName)
	assert.Equal(t, "domain", cfg.TargetType)
	assert.Equal(t, "subdomain_discovery", cfg.WorkflowName)
	assert.Equal(t, "/tmp/workspace", cfg.WorkspaceDir)
	assert.Equal(t, "debug", cfg.LogLevel)
	require.NotNil(t, cfg.Config)
	assert.EqualValues(t, 10, cfg.Config["threads"])
	assert.EqualValues(t, true, cfg.Config["flag"])
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	t.Setenv("TASK_ID", "99")
	t.Setenv("AGENT_SOCKET", "/run/lunafox/worker-runtime.sock")
	t.Setenv("TASK_TOKEN", "task-token")
	t.Setenv("SCAN_ID", "10")
	t.Setenv("TARGET_ID", "20")
	t.Setenv("TARGET_NAME", "example.com")
	t.Setenv("TARGET_TYPE", "domain")
	t.Setenv("WORKFLOW_NAME", "subdomain_discovery")
	t.Setenv("CONFIG", "{bad")

	_, err := Load()
	require.Error(t, err)
}

func TestValidateMissingFields(t *testing.T) {
	valid := Config{
		TaskID:       99,
		AgentSocket:  "/run/lunafox/worker-runtime.sock",
		TaskToken:    "task-token",
		ScanID:       1,
		TargetID:     2,
		TargetName:   "example.com",
		TargetType:   "domain",
		WorkflowName: "subdomain_discovery",
		Config:       map[string]any{},
	}

	tests := []struct {
		name string
		cfg  Config
		err  error
	}{
		{name: "missing task id", cfg: func() Config { c := valid; c.TaskID = 0; return c }(), err: ErrMissingTaskID},
		{name: "missing agent socket", cfg: func() Config { c := valid; c.AgentSocket = ""; return c }(), err: ErrMissingAgentSocket},
		{name: "missing task token", cfg: func() Config { c := valid; c.TaskToken = ""; return c }(), err: ErrMissingTaskToken},
		{name: "missing scan id", cfg: func() Config { c := valid; c.ScanID = 0; return c }(), err: ErrMissingScanID},
		{name: "missing target id", cfg: func() Config { c := valid; c.TargetID = 0; return c }(), err: ErrMissingTargetID},
		{name: "missing target name", cfg: func() Config { c := valid; c.TargetName = ""; return c }(), err: ErrMissingTargetName},
		{name: "missing target type", cfg: func() Config { c := valid; c.TargetType = ""; return c }(), err: ErrMissingTargetType},
		{name: "missing workflow name", cfg: func() Config { c := valid; c.WorkflowName = ""; return c }(), err: ErrMissingWorkflowName},
		{name: "missing config", cfg: func() Config { c := valid; c.Config = nil; return c }(), err: ErrMissingConfig},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ErrorIs(t, tt.cfg.Validate(), tt.err)
		})
	}
}
