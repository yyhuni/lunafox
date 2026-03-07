package main

import (
	"context"
	"testing"

	"github.com/yyhuni/lunafox/worker/internal/config"
	"github.com/yyhuni/lunafox/worker/internal/pkg"
	"github.com/yyhuni/lunafox/worker/internal/workflow"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestRunLogsSemanticStartupFields(t *testing.T) {
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
		WorkflowID:   "fake-workflow",
		WorkspaceDir: t.TempDir(),
		Config:       map[string]any{"a": "b"},
		LogLevel:     "info",
	}
	loadConfig = func() (*config.Config, error) { return cfg, nil }

	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	initLogger = func(level string) error {
		pkg.Logger = logger
		return nil
	}
	syncLogger = func() {}

	client := &fakeRuntimeClient{}
	newClient = func(socketPath, taskToken string, taskID int) (runtimeClient, error) {
		return client, nil
	}
	getWorkflow = func(name, workDir string) workflow.Workflow {
		return &fakeWorkflow{}
	}
	decodeWorkflowConfig = func(name string, scanConfig map[string]any) (any, error) {
		return nil, nil
	}

	if err := run(context.Background()); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	entries := logs.FilterMessage("Worker starting").All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 startup log, got %d", len(entries))
	}
	ctx := entries[0].ContextMap()
	for _, key := range []string{"task.id", "scan.id", "target.id", "target.name", "target.type", "workflow.id"} {
		if _, ok := ctx[key]; !ok {
			t.Fatalf("expected %s field, got %v", key, ctx)
		}
	}
	for _, key := range []string{"taskId", "scanId", "targetId", "targetName", "targetType", "workflow"} {
		if _, ok := ctx[key]; ok {
			t.Fatalf("expected legacy field %s removed, got %v", key, ctx)
		}
	}

	completeEntries := logs.FilterMessage("Worker completed successfully").All()
	if len(completeEntries) != 1 {
		t.Fatalf("expected 1 completion log, got %d", len(completeEntries))
	}
	completeCtx := completeEntries[0].ContextMap()
	if _, ok := completeCtx["scan.id"]; !ok {
		t.Fatalf("expected scan.id field, got %v", completeCtx)
	}
	if _, ok := completeCtx["scanId"]; ok {
		t.Fatalf("expected legacy scanId removed, got %v", completeCtx)
	}
}
