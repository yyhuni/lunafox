package domain

import "testing"

func TestNewRegisteredAgentAppliesDefaultsAndNormalization(t *testing.T) {
	agent := NewRegisteredAgent("abcd1234", "  host-a  ", "1.0.0", " 1.2.3.4 ", "key12345", AgentRegistrationOptions{})

	if agent.Hostname != "host-a" {
		t.Fatalf("expected normalized hostname, got %q", agent.Hostname)
	}
	if agent.Name != "agent-host-a" {
		t.Fatalf("expected generated name, got %q", agent.Name)
	}
	if agent.Status != "offline" {
		t.Fatalf("expected offline status, got %q", agent.Status)
	}
	if agent.MaxTasks != 10 || agent.CPUThreshold != 80 || agent.MemThreshold != 80 || agent.DiskThreshold != 85 {
		t.Fatalf("expected default config 10/80/80/85, got %d/%d/%d/%d", agent.MaxTasks, agent.CPUThreshold, agent.MemThreshold, agent.DiskThreshold)
	}
}

func TestAgentApplyConfigUpdatePartial(t *testing.T) {
	agent := &Agent{MaxTasks: 5, CPUThreshold: 70, MemThreshold: 75, DiskThreshold: 80}
	maxTasks := 12
	mem := 90
	agent.ApplyConfigUpdate(AgentConfigUpdate{MaxTasks: &maxTasks, MemThreshold: &mem})

	if agent.MaxTasks != 12 {
		t.Fatalf("expected maxTasks updated to 12, got %d", agent.MaxTasks)
	}
	if agent.CPUThreshold != 70 {
		t.Fatalf("expected cpu unchanged, got %d", agent.CPUThreshold)
	}
	if agent.MemThreshold != 90 {
		t.Fatalf("expected mem updated to 90, got %d", agent.MemThreshold)
	}
	if agent.DiskThreshold != 80 {
		t.Fatalf("expected disk unchanged, got %d", agent.DiskThreshold)
	}
}
