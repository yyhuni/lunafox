package runtime

import (
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/agent/internal/domain"
	runtimev1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/runtime/v1"
)

func TestBuildRuntimeTarget(t *testing.T) {
	tests := []struct {
		name       string
		runtimeURL string
		target     string
		secure     bool
		shouldFail bool
	}{
		{name: "https default port", runtimeURL: "https://example.com", target: "example.com:443", secure: true},
		{name: "http default port", runtimeURL: "http://example.com", target: "example.com:80", secure: false},
		{name: "explicit port", runtimeURL: "https://example.com:9443/api", target: "example.com:9443", secure: true},
		{name: "unsupported scheme", runtimeURL: "ftp://example.com", shouldFail: true},
		{name: "missing scheme", runtimeURL: "example.com", shouldFail: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			target, secure, err := buildRuntimeTarget(tc.runtimeURL)
			if tc.shouldFail {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if target != tc.target || secure != tc.secure {
				t.Fatalf("unexpected target result target=%s secure=%v", target, secure)
			}
		})
	}
}

func TestBuildRuntimeTargetErrorMessageUsesRuntimeURLTerm(t *testing.T) {
	_, _, err := buildRuntimeTarget("")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "runtime url") {
		t.Fatalf("error should use runtime URL term, got: %v", err)
	}
}

func TestDispatchEventCallbacks(t *testing.T) {
	client := NewClient("https://example.com", "agent-key")
	var (
		cancelledTaskID int
		configUpdate    domain.ConfigUpdate
		updateRequired  domain.UpdateRequiredPayload
	)
	client.OnTaskCancel(func(taskID int) {
		cancelledTaskID = taskID
	})
	client.OnConfigUpdate(func(update domain.ConfigUpdate) {
		configUpdate = update
	})
	client.OnUpdateRequired(func(update domain.UpdateRequiredPayload) {
		updateRequired = update
	})

	client.dispatchEvent(&runtimev1.AgentRuntimeEvent{
		Payload: &runtimev1.AgentRuntimeEvent_TaskCancel{
			TaskCancel: &runtimev1.TaskCancel{TaskId: 11},
		},
	})
	if cancelledTaskID != 11 {
		t.Fatalf("unexpected cancelled task id: %d", cancelledTaskID)
	}

	maxTasks := int32(8)
	client.dispatchEvent(&runtimev1.AgentRuntimeEvent{
		Payload: &runtimev1.AgentRuntimeEvent_ConfigUpdate{
			ConfigUpdate: &runtimev1.ConfigUpdate{
				MaxTasks: &maxTasks,
			},
		},
	})
	if configUpdate.MaxTasks == nil || *configUpdate.MaxTasks != 8 {
		t.Fatalf("unexpected config update: %+v", configUpdate)
	}

	client.dispatchEvent(&runtimev1.AgentRuntimeEvent{
		Payload: &runtimev1.AgentRuntimeEvent_UpdateRequired{
			UpdateRequired: &runtimev1.UpdateRequired{
				TargetVersion: "v2.0.0",
				ImageRef:      "registry.example.com/lunafox-agent:v2.0.0",
			},
		},
	})
	if updateRequired.Version != "v2.0.0" || updateRequired.ImageRef == "" {
		t.Fatalf("unexpected update required payload: %+v", updateRequired)
	}
}

func TestDispatchTaskAssignResolvesPendingPull(t *testing.T) {
	client := NewClient("https://example.com", "agent-key")
	waiter := make(chan pullResult, 1)
	client.pendingPull = waiter

	client.dispatchEvent(&runtimev1.AgentRuntimeEvent{
		Payload: &runtimev1.AgentRuntimeEvent_TaskAssign{
			TaskAssign: &runtimev1.TaskAssign{
				Found:  true,
				TaskId: 9,
			},
		},
	})

	select {
	case result := <-waiter:
		if result.assign == nil || result.assign.TaskId != 9 {
			t.Fatalf("unexpected assign result: %+v", result)
		}
	default:
		t.Fatalf("expected pending pull to receive assignment")
	}
}
