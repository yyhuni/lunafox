package service

import (
	"testing"
	"time"

	"github.com/yyhuni/lunafox/server/internal/agentproto"
	runtimev1 "github.com/yyhuni/lunafox/server/internal/grpc/runtime/v1/gen"
)

func TestAgentRuntimeEventPublisherPublishesTypedEvents(t *testing.T) {
	registry := NewAgentStreamRegistry()
	outbound, unregister := registry.Register(9)
	defer unregister()

	publisher := NewAgentRuntimeEventPublisher(registry)
	maxTasks := 4
	cpuThreshold := 75
	publisher.SendConfigUpdate(9, agentproto.ConfigUpdatePayload{
		MaxTasks:     &maxTasks,
		CPUThreshold: &cpuThreshold,
	})

	configEvent := mustReceiveRuntimeEvent(t, outbound)
	if configEvent.GetConfigUpdate() == nil {
		t.Fatalf("expected config_update event, got %+v", configEvent)
	}
	if configEvent.GetConfigUpdate().GetMaxTasks() != 4 {
		t.Fatalf("unexpected max_tasks: %d", configEvent.GetConfigUpdate().GetMaxTasks())
	}
	if configEvent.GetConfigUpdate().GetCpuThreshold() != 75 {
		t.Fatalf("unexpected cpu_threshold: %d", configEvent.GetConfigUpdate().GetCpuThreshold())
	}

	delivered := publisher.SendUpdateRequired(9, agentproto.UpdateRequiredPayload{
		Version:  "v2.0.0",
		ImageRef: "registry.example.com/lunafox-agent:v2.0.0",
	})
	if !delivered {
		t.Fatalf("expected update_required delivery")
	}

	updateEvent := mustReceiveRuntimeEvent(t, outbound)
	if updateEvent.GetUpdateRequired() == nil || updateEvent.GetUpdateRequired().GetTargetVersion() != "v2.0.0" {
		t.Fatalf("unexpected update_required event: %+v", updateEvent)
	}

	publisher.SendTaskCancel(9, 123)
	cancelEvent := mustReceiveRuntimeEvent(t, outbound)
	if cancelEvent.GetTaskCancel() == nil || cancelEvent.GetTaskCancel().GetTaskId() != 123 {
		t.Fatalf("unexpected task_cancel event: %+v", cancelEvent)
	}
}

func TestAgentRuntimeEventPublisherNoTargetReturnsFalse(t *testing.T) {
	publisher := NewAgentRuntimeEventPublisher(NewAgentStreamRegistry())
	delivered := publisher.SendUpdateRequired(100, agentproto.UpdateRequiredPayload{
		Version: "v9.9.9",
	})
	if delivered {
		t.Fatalf("expected no delivery when agent has no active stream")
	}
}

func mustReceiveRuntimeEvent(t *testing.T, outbound <-chan *runtimev1.AgentRuntimeEvent) *runtimev1.AgentRuntimeEvent {
	t.Helper()
	select {
	case event := <-outbound:
		return event
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for runtime event")
		return nil
	}
}
