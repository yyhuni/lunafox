package service

import (
	"testing"

	runtimev1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/runtime/v1"
)

func TestToRuntimeHeartbeatInputMapsWorkerVersion(t *testing.T) {
	input := toRuntimeHeartbeatInput(&runtimev1.Heartbeat{
		AgentVersion:  "2.0.0",
		WorkerVersion: "1.0.0",
	})

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
