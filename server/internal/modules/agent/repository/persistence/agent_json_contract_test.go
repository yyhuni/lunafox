package model

import (
	"reflect"
	"testing"
)

func TestAgentModelJSONTagsUseCamelCase(t *testing.T) {
	tests := map[string]string{
		"APIKey":            "apiKey",
		"IPAddress":         "ipAddress",
		"AgentVersion":      "agentVersion",
		"WorkerVersion":     "workerVersion",
		"MaxTasks":          "maxTasks",
		"CPUThreshold":      "cpuThreshold",
		"MemThreshold":      "memThreshold",
		"DiskThreshold":     "diskThreshold",
		"HealthState":       "healthState",
		"HealthReason":      "healthReason,omitempty",
		"HealthMessage":     "healthMessage,omitempty",
		"HealthSince":       "healthSince,omitempty",
		"RegistrationToken": "registrationToken,omitempty",
		"ConnectedAt":       "connectedAt,omitempty",
		"LastHeartbeat":     "lastHeartbeat,omitempty",
		"CreatedAt":         "createdAt",
		"UpdatedAt":         "updatedAt",
	}

	for fieldName, expectedTag := range tests {
		field, ok := reflect.TypeOf(Agent{}).FieldByName(fieldName)
		if !ok {
			t.Fatalf("agent model must define %s", fieldName)
		}
		if got := field.Tag.Get("json"); got != expectedTag {
			t.Fatalf("%s must use %s json tag, got %q", fieldName, expectedTag, got)
		}
	}
}
