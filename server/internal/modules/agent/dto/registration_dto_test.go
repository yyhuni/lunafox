package dto

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAgentRegistrationRequestRejectsLegacyWorkerVersion(t *testing.T) {
	for _, value := range []string{`"1.2.3"`, "null"} {
		t.Run(value, func(t *testing.T) {
			var request AgentRegistrationRequest
			err := json.Unmarshal([]byte(`{
				"token":"abcd1234",
				"observedHostname":"node-a",
				"agentVersion":"1.2.3",
				"workerVersion":`+value+`
			}`), &request)
			if err == nil || !strings.Contains(err.Error(), "workerVersion is not supported") {
				t.Fatalf("expected workerVersion rejection, got %v", err)
			}
		})
	}
}

func TestAgentRegistrationRequestAcceptsAgentOnlyPayload(t *testing.T) {
	var request AgentRegistrationRequest
	err := json.Unmarshal([]byte(`{
		"token":"abcd1234",
		"observedHostname":"node-a",
		"agentVersion":"1.2.3"
	}`), &request)
	if err != nil {
		t.Fatalf("expected Agent-only registration payload, got %v", err)
	}
	if request.AgentVersion != "1.2.3" {
		t.Fatalf("unexpected agent version: %q", request.AgentVersion)
	}
}
