package agentcontrol

import (
	"testing"

	agentcontrolv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/control/v1"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
)

func TestToAgentHeartbeatEventMapsAgentFields(t *testing.T) {
	event, err := toAgentHeartbeatEvent(&agentcontrolv1.Heartbeat{
		Agent:            resourcenames.Agent("agt-1"),
		Session:          resourcenames.AgentSession("agt-1", "session-1"),
		ObservedHostname: "node-a",
		AgentVersion:     "2.0.0",
		Health: &agentcontrolv1.HealthStatus{
			State: agentcontrolv1.HealthState_HEALTH_STATE_PAUSED,
		},
	})
	if err != nil {
		t.Fatalf("toAgentHeartbeatEvent returned error: %v", err)
	}

	if event.AgentVersion != "2.0.0" {
		t.Fatalf("unexpected agent version: %q", event.AgentVersion)
	}
	if event.InstanceID != "agt-1" {
		t.Fatalf("unexpected instance id: %q", event.InstanceID)
	}
	if event.SessionID != "session-1" {
		t.Fatalf("unexpected session id: %q", event.SessionID)
	}
	if event.ObservedHostname != "node-a" {
		t.Fatalf("unexpected observed hostname: %q", event.ObservedHostname)
	}
	if event.Health == nil || event.Health.State != "paused" {
		t.Fatalf("expected paused health mapping, got %#v", event.Health)
	}
}

func TestToAgentHeartbeatEventRejectsUnspecifiedHealthState(t *testing.T) {
	_, err := toAgentHeartbeatEvent(&agentcontrolv1.Heartbeat{
		Health: &agentcontrolv1.HealthStatus{
			State: agentcontrolv1.HealthState_HEALTH_STATE_UNSPECIFIED,
		},
	})
	if err == nil {
		t.Fatalf("expected unspecified health state to be rejected")
	}
}

func TestFromProtoTerminalTaskResultStateCoversAllBranches(t *testing.T) {
	cases := []struct {
		name   string
		input  agentcontrolv1.TerminalTaskResultState
		want   string
		hasErr bool
	}{
		{name: "succeeded", input: agentcontrolv1.TerminalTaskResultState_TERMINAL_TASK_RESULT_STATE_SUCCEEDED, want: "succeeded"},
		{name: "failed", input: agentcontrolv1.TerminalTaskResultState_TERMINAL_TASK_RESULT_STATE_FAILED, want: "failed"},
		{name: "cancelled", input: agentcontrolv1.TerminalTaskResultState_TERMINAL_TASK_RESULT_STATE_CANCELLED, want: "cancelled"},
		{name: "unspecified", input: agentcontrolv1.TerminalTaskResultState_TERMINAL_TASK_RESULT_STATE_UNSPECIFIED, hasErr: true},
		{name: "unknown", input: agentcontrolv1.TerminalTaskResultState(99), hasErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := fromProtoTerminalTaskResultState(tc.input)
			if tc.hasErr {
				if err == nil {
					t.Fatalf("expected error for %v", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("unexpected mapped terminal task result: %q", got)
			}
		})
	}
}

func TestFromProtoHealthStateRejectsUnknownValue(t *testing.T) {
	if _, err := fromProtoHealthState(agentcontrolv1.HealthState(99)); err == nil {
		t.Fatalf("expected unknown health state to be rejected")
	}
}
