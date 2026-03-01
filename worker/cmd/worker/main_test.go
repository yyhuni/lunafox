package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/worker/internal/config"
	"github.com/yyhuni/lunafox/worker/internal/pkg"
	"github.com/yyhuni/lunafox/worker/internal/server"
	"github.com/yyhuni/lunafox/worker/internal/workflow"
	"go.uber.org/zap"
)

type fakeRuntimeClient struct {
	closed bool
}

func (c *fakeRuntimeClient) Close() error {
	c.closed = true
	return nil
}

func (c *fakeRuntimeClient) GetProviderConfig(ctx context.Context, scanID int, toolName string) (*server.ProviderConfig, error) {
	return nil, nil
}

func (c *fakeRuntimeClient) EnsureWordlistLocal(ctx context.Context, wordlistName, basePath string) (string, error) {
	return "", nil
}

func (c *fakeRuntimeClient) PostBatch(ctx context.Context, scanID, targetID int, dataType string, items []any) error {
	return nil
}

type fakeWorkflow struct {
	execErr error
	saveErr error
}

func (w *fakeWorkflow) Name() string { return "fake" }

func (w *fakeWorkflow) Execute(params *workflow.Params) (*workflow.Output, error) {
	if w.execErr != nil {
		return nil, w.execErr
	}
	return &workflow.Output{Data: []string{"a"}}, nil
}

func (w *fakeWorkflow) SaveResults(ctx context.Context, client server.ServerClient, params *workflow.Params, output *workflow.Output) error {
	return w.saveErr
}

func TestRunExecuteErrorStillRunsCleanup(t *testing.T) {
	origLoadConfig := loadConfig
	origInitLogger := initLogger
	origSyncLogger := syncLogger
	origNewClient := newClient
	origGetWorkflow := getWorkflow
	origDecodeWorkflowConfig := decodeWorkflowConfig
	origLogger := pkg.Logger
	t.Cleanup(func() {
		loadConfig = origLoadConfig
		initLogger = origInitLogger
		syncLogger = origSyncLogger
		newClient = origNewClient
		getWorkflow = origGetWorkflow
		decodeWorkflowConfig = origDecodeWorkflowConfig
		pkg.Logger = origLogger
	})

	cfg := &config.Config{
		TaskID:       101,
		AgentSocket:  "/tmp/agent.sock",
		TaskToken:    "task-token",
		ScanID:       11,
		TargetID:     22,
		TargetName:   "example.com",
		TargetType:   "domain",
		WorkflowName: "fake-workflow",
		WorkspaceDir: "/tmp/workspace",
		Config:       map[string]any{"a": "b"},
		LogLevel:     "info",
	}
	loadConfig = func() (*config.Config, error) { return cfg, nil }
	initLogger = func(level string) error {
		pkg.Logger = zap.NewNop()
		return nil
	}
	syncCalled := false
	syncLogger = func() { syncCalled = true }

	client := &fakeRuntimeClient{}
	newClient = func(socketPath, taskToken string, taskID int) (runtimeClient, error) {
		return client, nil
	}
	getWorkflow = func(name, workDir string) workflow.Workflow {
		return &fakeWorkflow{execErr: errors.New("boom")}
	}
	decodeWorkflowConfig = func(name string, scanConfig map[string]any) (any, error) {
		return nil, nil
	}

	err := run(context.Background())
	if err == nil {
		t.Fatalf("expected run to return error")
	}
	if !strings.Contains(err.Error(), "execute workflow") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !client.closed {
		t.Fatalf("expected runtime client to be closed on failure")
	}
	if !syncCalled {
		t.Fatalf("expected logger sync to run on failure")
	}
}
