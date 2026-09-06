package repository

import (
	"reflect"
	"testing"

	model "github.com/yyhuni/lunafox/server/internal/modules/agent/repository/persistence"
)

func TestModelAgentToDomainOmitsRemovedRuntimeStatusField(t *testing.T) {
	agent := modelAgentToDomain(&model.Agent{
		ID:                  1,
		InstanceID:          "agt-123",
		DisplayName:         "agent-1",
		AuthenticationToken: "deadbeef",
		Status:              "online",
		RuntimeStatus: &model.AgentRuntimeStatus{
			AgentVersion: "1.2.3",
		},
	})

	if agent == nil {
		t.Fatalf("expected mapped domain agent")
	}
	if agent.AgentVersion != "1.2.3" {
		t.Fatalf("expected Agent version mapped, got %#v", agent)
	}
	if agent.DisplayName != "agent-1" {
		t.Fatalf("expected display name mapped, got %#v", agent)
	}
	if _, ok := reflect.TypeOf(*agent).FieldByName("RuntimeStatus"); ok {
		t.Fatalf("domain agent must not expose RuntimeStatus")
	}
	if _, ok := reflect.TypeOf(*agent).FieldByName("WorkerVersion"); ok {
		t.Fatalf("domain agent must not expose WorkerVersion")
	}
	if _, ok := reflect.TypeOf(model.AgentRuntimeStatus{}).FieldByName("WorkerVersion"); ok {
		t.Fatalf("persistence runtime status must not retain WorkerVersion")
	}
}
