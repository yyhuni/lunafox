package service

import (
	"testing"

	runtimev1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/runtime/v1"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

func TestToRuntimeHeartbeatInputMapsWorkerVersion(t *testing.T) {
	input := toRuntimeHeartbeatInput(&runtimev1.Heartbeat{AgentVersion: "2.0.0", WorkerVersion: "1.0.0"})

	if input.Heartbeat == nil {
		t.Fatalf("expected heartbeat payload")
	}
	if input.Heartbeat.AgentVersion != "2.0.0" {
		t.Fatalf("unexpected agent version: %q", input.Heartbeat.AgentVersion)
	}
	if input.Heartbeat.WorkerVersion != "1.0.0" {
		t.Fatalf("unexpected worker version: %q", input.Heartbeat.WorkerVersion)
	}
}

func TestToTaskAssignMapsWorkflowConfigObject(t *testing.T) {
	payload := toTaskAssign(&scanapp.TaskAssignment{
		TaskID:     11,
		ScanID:     22,
		Stage:      1,
		WorkflowID: "subdomain_discovery",
		WorkflowConfig: map[string]any{
			"recon": map[string]any{
				"enabled": false,
			},
		},
	})

	if payload == nil || !payload.Found {
		t.Fatalf("expected found assignment payload")
	}
	if payload.WorkflowConfig == nil {
		t.Fatalf("expected workflow config object payload")
	}
	if _, ok := payload.WorkflowConfig.Fields["recon"]; !ok {
		t.Fatalf("expected workflow config object to include recon field, got %#v", payload.WorkflowConfig)
	}
}
