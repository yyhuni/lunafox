package service

import (
	"testing"

	runtimev1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/runtime/v1"
)

func TestToRuntimeHeartbeatInputMapsWorkerCapability(t *testing.T) {
	input := toRuntimeHeartbeatInput(&runtimev1.Heartbeat{
		Version:       "2.0.0",
		WorkerVersion: "1.0.0",
		SupportedWorkflows: []*runtimev1.WorkerWorkflowSupport{
			{
				Workflow:      "subdomain_discovery",
				ApiVersion:    "v1",
				SchemaVersion: "1.0.0",
			},
		},
	})

	if input.Heartbeat == nil {
		t.Fatalf("expected heartbeat payload")
	}
	if input.Heartbeat.WorkerVersion != "1.0.0" {
		t.Fatalf("unexpected worker version: %q", input.Heartbeat.WorkerVersion)
	}
	if len(input.Heartbeat.SupportedWorkflows) != 1 {
		t.Fatalf("expected one supported workflow")
	}
	item := input.Heartbeat.SupportedWorkflows[0]
	if item.Workflow != "subdomain_discovery" || item.APIVersion != "v1" || item.SchemaVersion != "1.0.0" {
		t.Fatalf("unexpected supported workflow mapping: %+v", item)
	}
}
