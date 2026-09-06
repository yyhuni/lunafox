package domain

import "testing"

func TestAgentOutboundEventTypeConstantsStable(t *testing.T) {
	if AgentOutboundConfigUpdate != "config_update" {
		t.Fatalf("unexpected config update constant: %q", AgentOutboundConfigUpdate)
	}
	if AgentOutboundUpdateRequired != "update_required" {
		t.Fatalf("unexpected update required constant: %q", AgentOutboundUpdateRequired)
	}
	if AgentOutboundTaskCancel != "task_cancel" {
		t.Fatalf("unexpected task cancel constant: %q", AgentOutboundTaskCancel)
	}
}
