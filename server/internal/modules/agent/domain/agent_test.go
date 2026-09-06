package domain

import (
	"reflect"
	"testing"
)

func TestAgentDoesNotDefineRuntimeStatus(t *testing.T) {
	if _, ok := reflect.TypeOf(Agent{}).FieldByName("RuntimeStatus"); ok {
		t.Fatalf("Agent must not define RuntimeStatus")
	}
}

func TestNewRegisteredAgentDefaultsHealthStateHealthy(t *testing.T) {
	agent := NewRegisteredAgent(
		7,
		"agt_123456",
		"agent-test",
		"1.0.0",
		"deadbeef",
		AgentRegistrationOptions{},
	)
	if agent == nil {
		t.Fatalf("expected agent")
	}
	if agent.HealthState != "healthy" {
		t.Fatalf("expected healthy default health state, got %q", agent.HealthState)
	}
	if agent.InstanceID != "agt_123456" {
		t.Fatalf("expected instance id persisted, got %q", agent.InstanceID)
	}
	if agent.DisplayName == "" {
		t.Fatalf("expected display name initialized")
	}
	if agent.RegistrationTokenID != 7 {
		t.Fatalf("expected non-secret registration token identity 7, got %d", agent.RegistrationTokenID)
	}
	if agent.ObservedSourceIP != "" || agent.ObservedIPGeneration != 0 {
		t.Fatalf("registration must not initialize connection-source observation: %#v", agent)
	}
}

func TestAgentDoesNotExposeWorkerVersion(t *testing.T) {
	if _, ok := reflect.TypeOf(Agent{}).FieldByName("WorkerVersion"); ok {
		t.Fatal("Agent must not expose the removed Worker version")
	}
}
