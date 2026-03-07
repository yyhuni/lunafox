package subdomain_discovery

import (
	"testing"

	"github.com/yyhuni/lunafox/worker/internal/pkg"
	"github.com/yyhuni/lunafox/worker/internal/workflow"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func withObservedWorkflowLogger(t *testing.T) *observer.ObservedLogs {
	t.Helper()
	core, logs := observer.New(zap.DebugLevel)
	prev := pkg.Logger
	pkg.Logger = zap.New(core)
	t.Cleanup(func() {
		pkg.Logger = prev
	})
	return logs
}

func TestInitializeLogsSemanticFields(t *testing.T) {
	logs := withObservedWorkflowLogger(t)
	w := New(t.TempDir())

	_, err := w.initialize(&workflow.Params{
		ScanID:         7,
		TargetType:     "domain",
		TargetName:     "example.com",
		WorkDir:        t.TempDir(),
		WorkflowConfig: validTypedWorkflowConfig(t),
		ServerClient:   providerClient{},
	})
	if err != nil {
		t.Fatalf("initialize returned error: %v", err)
	}

	entries := logs.FilterMessage("Workflow initialized").All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 workflow initialized log, got %d", len(entries))
	}
	ctx := entries[0].ContextMap()
	for _, key := range []string{"scan.id", "target.name", "target.type"} {
		if _, ok := ctx[key]; !ok {
			t.Fatalf("expected %s field, got %v", key, ctx)
		}
	}
	for _, key := range []string{"scanId", "targetName", "targetType"} {
		if _, ok := ctx[key]; ok {
			t.Fatalf("expected legacy field %s removed, got %v", key, ctx)
		}
	}
}
