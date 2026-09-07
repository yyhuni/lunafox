package agentcontrol

import (
	"testing"
	"time"

	agentcontrolv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/control/v1"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

func TestAgentControlEventPublisherPublishesTypedEvents(t *testing.T) {
	registry := NewAgentStreamRegistry()
	streamID, outbound, unregister := registry.RegisterStream(9)
	defer unregister()
	registry.SetActiveStream(9, streamID)

	publisher := NewAgentControlEventPublisher(registry)
	maxTasks := 4
	cpuThreshold := 75
	memThreshold := 80
	diskThreshold := 85
	publisher.SendConfigUpdate(9, agentdomain.ConfigUpdatePayload{
		MaxTasks:      &maxTasks,
		CPUThreshold:  &cpuThreshold,
		MemThreshold:  &memThreshold,
		DiskThreshold: &diskThreshold,
	})

	configEvent := mustReceiveControlEvent(t, outbound)
	if configEvent.GetConfigUpdate() == nil {
		t.Fatalf("expected config_update event, got %+v", configEvent)
	}
	if configEvent.GetConfigUpdate().GetMaxTasks() != 4 {
		t.Fatalf("unexpected max_tasks: %d", configEvent.GetConfigUpdate().GetMaxTasks())
	}
	if configEvent.GetConfigUpdate().GetCpuThreshold() != 75 {
		t.Fatalf("unexpected cpu_threshold: %d", configEvent.GetConfigUpdate().GetCpuThreshold())
	}
	if configEvent.GetConfigUpdate().GetMemThreshold() != 80 {
		t.Fatalf("unexpected mem_threshold: %d", configEvent.GetConfigUpdate().GetMemThreshold())
	}
	if configEvent.GetConfigUpdate().GetDiskThreshold() != 85 {
		t.Fatalf("unexpected disk_threshold: %d", configEvent.GetConfigUpdate().GetDiskThreshold())
	}

	delivered := publisher.SendUpdateRequired(9, agentdomain.UpdateRequiredPayload{
		AgentVersion:  "v2.0.0",
		AgentImageRef: "registry.example.com/lunafox-agent:v2.0.0",
	})
	if !delivered {
		t.Fatalf("expected update_required delivery")
	}

	updateEvent := mustReceiveControlEvent(t, outbound)
	if updateEvent.GetUpdateRequired() == nil || updateEvent.GetUpdateRequired().GetAgentVersion() != "v2.0.0" {
		t.Fatalf("unexpected update_required event: %+v", updateEvent)
	}
	if updateEvent.GetUpdateRequired().GetAgentImage() != "registry.example.com/lunafox-agent:v2.0.0" {
		t.Fatalf("unexpected agent image ref: %q", updateEvent.GetUpdateRequired().GetAgentImage())
	}

	if delivered := publisher.TrySendTaskCancel(9, 12, 123); !delivered {
		t.Fatal("expected task_cancel delivery")
	}
	cancelEvent := mustReceiveControlEvent(t, outbound)
	if cancelEvent.GetTaskCancel() == nil || cancelEvent.GetTaskCancel().GetTask() != resourcenames.Task(12, 123) {
		t.Fatalf("unexpected task_cancel event: %+v", cancelEvent)
	}
}

func TestAgentControlEventPublisherNoTargetReturnsFalse(t *testing.T) {
	publisher := NewAgentControlEventPublisher(NewAgentStreamRegistry())
	delivered := publisher.SendUpdateRequired(100, agentdomain.UpdateRequiredPayload{
		AgentVersion: "v9.9.9",
	})
	if delivered {
		t.Fatalf("expected no delivery when agent has no active stream")
	}
	if delivered := publisher.TrySendTaskCancel(100, 12, 123); delivered {
		t.Fatal("expected task_cancel delivery to report false with no active stream")
	}
}

func TestAgentStreamRegistryPublishDropsWhenOutboundBufferIsFull(t *testing.T) {
	registry := NewAgentStreamRegistry()
	streamID, outbound, unregister := registry.RegisterStream(7)
	defer unregister()
	registry.SetActiveStream(7, streamID)

	for i := 0; i < defaultAgentStreamBufferSize; i++ {
		if delivered := registry.Publish(7, &agentcontrolv1.ConnectResponse{}); !delivered {
			t.Fatalf("expected buffer slot %d to accept event", i)
		}
	}

	publishDone := make(chan bool, 1)
	go func() {
		publishDone <- registry.Publish(7, &agentcontrolv1.ConnectResponse{})
	}()

	select {
	case delivered := <-publishDone:
		if delivered {
			t.Fatal("expected publish to drop event when outbound buffer is full")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected publish to remain non-blocking when outbound buffer is full")
	}

	for i := 0; i < defaultAgentStreamBufferSize; i++ {
		mustReceiveControlEvent(t, outbound)
	}
}

func TestAgentStreamRegistryUnregisterClearsActiveStreamAndAgentEntries(t *testing.T) {
	registry := NewAgentStreamRegistry()
	streamID, _, unregister := registry.RegisterStream(11)
	registry.SetActiveStream(11, streamID)

	unregister()

	if _, ok := registry.activeStreamByAgent[11]; ok {
		t.Fatalf("expected unregister to clear active stream entry, got %d", registry.activeStreamByAgent[11])
	}
	if _, ok := registry.streams[11]; ok {
		t.Fatalf("expected unregister to clear agent stream entry, got %+v", registry.streams[11])
	}

	unregister()

	if _, ok := registry.activeStreamByAgent[11]; ok {
		t.Fatalf("expected unregister to remain idempotent for active stream entry, got %d", registry.activeStreamByAgent[11])
	}
	if _, ok := registry.streams[11]; ok {
		t.Fatalf("expected unregister to remain idempotent for agent stream entry, got %+v", registry.streams[11])
	}
}

func TestAgentStreamRegistrySetAndClearActiveStreamIgnoreNonMatchingStreams(t *testing.T) {
	registry := NewAgentStreamRegistry()
	activeStreamID, _, unregisterActive := registry.RegisterStream(12)
	defer unregisterActive()
	otherStreamID, _, unregisterOther := registry.RegisterStream(12)
	defer unregisterOther()

	registry.SetActiveStream(12, activeStreamID)
	registry.SetActiveStream(12, 999)
	if got := registry.activeStreamByAgent[12]; got != activeStreamID {
		t.Fatalf("expected unknown stream activation to be ignored, got %d want %d", got, activeStreamID)
	}

	registry.ClearActiveStream(12, otherStreamID)
	if got := registry.activeStreamByAgent[12]; got != activeStreamID {
		t.Fatalf("expected clearing non-active stream to be ignored, got %d want %d", got, activeStreamID)
	}
}

func TestAgentStreamRegistryGuardClauses(t *testing.T) {
	t.Run("nil registry", func(t *testing.T) {
		var registry *AgentStreamRegistry

		streamID, outbound, unregister := registry.RegisterStream(9)
		if streamID != 0 {
			t.Fatalf("expected nil registry RegisterStream to return zero stream id, got %d", streamID)
		}
		if outbound == nil {
			t.Fatal("expected nil registry RegisterStream to return a non-nil channel")
		}
		unregister()

		registry.SetActiveStream(9, 1)
		registry.ClearActiveStream(9, 1)
		if delivered := registry.Publish(9, &agentcontrolv1.ConnectResponse{}); delivered {
			t.Fatal("expected nil registry publish to return false")
		}
	})

	t.Run("invalid ids and nil event", func(t *testing.T) {
		registry := NewAgentStreamRegistry()
		streamID, _, unregister := registry.RegisterStream(13)
		defer unregister()

		if len(registry.streams) != 1 {
			t.Fatalf("expected valid RegisterStream to add one stream entry, got %d", len(registry.streams))
		}

		invalidStreamID, _, invalidUnregister := registry.RegisterStream(0)
		if invalidStreamID != 0 {
			t.Fatalf("expected invalid RegisterStream to return zero stream id, got %d", invalidStreamID)
		}
		invalidUnregister()
		if len(registry.streams) != 1 {
			t.Fatalf("expected invalid RegisterStream to keep stream entries unchanged, got %d", len(registry.streams))
		}

		registry.SetActiveStream(13, streamID)
		registry.SetActiveStream(0, streamID)
		registry.SetActiveStream(13, 0)
		if got := registry.activeStreamByAgent[13]; got != streamID {
			t.Fatalf("expected invalid SetActiveStream inputs to be ignored, got %d want %d", got, streamID)
		}

		registry.ClearActiveStream(0, streamID)
		registry.ClearActiveStream(13, 0)
		if got := registry.activeStreamByAgent[13]; got != streamID {
			t.Fatalf("expected invalid ClearActiveStream inputs to be ignored, got %d want %d", got, streamID)
		}

		if delivered := registry.Publish(0, &agentcontrolv1.ConnectResponse{}); delivered {
			t.Fatal("expected publish with invalid agent id to return false")
		}
		if delivered := registry.Publish(13, nil); delivered {
			t.Fatal("expected publish with nil event to return false")
		}
	})
}

func TestAgentControlEventPublisherGuardClauses(t *testing.T) {
	payload := agentdomain.UpdateRequiredPayload{AgentVersion: "v1.0.0"}

	t.Run("nil publisher receiver", func(t *testing.T) {
		var publisher *AgentControlEventPublisher

		publisher.SendConfigUpdate(1, agentdomain.ConfigUpdatePayload{})
		publisher.SendTaskCancel(1, 1, 1)
		if delivered := publisher.SendUpdateRequired(1, payload); delivered {
			t.Fatal("expected nil publisher SendUpdateRequired to return false")
		}
	})

	t.Run("nil registry in publisher", func(t *testing.T) {
		publisher := &AgentControlEventPublisher{}

		publisher.SendConfigUpdate(1, agentdomain.ConfigUpdatePayload{})
		publisher.SendTaskCancel(1, 1, 1)
		if delivered := publisher.SendUpdateRequired(1, payload); delivered {
			t.Fatal("expected publisher with nil registry SendUpdateRequired to return false")
		}
	})
}

func TestIntPtrToInt32PtrHandlesNil(t *testing.T) {
	if got := intPtrToInt32Ptr(nil); got != nil {
		t.Fatalf("expected nil input to stay nil, got %v", *got)
	}
}

func mustReceiveControlEvent(t *testing.T, outbound <-chan *agentcontrolv1.ConnectResponse) *agentcontrolv1.ConnectResponse {
	t.Helper()
	select {
	case event := <-outbound:
		return event
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for control-plane event")
		return nil
	}
}
