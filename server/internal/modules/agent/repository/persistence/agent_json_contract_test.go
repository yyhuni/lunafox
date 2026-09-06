package model

import (
	"reflect"
	"strings"
	"testing"
)

func TestAgentModelJSONTagsUseCamelCase(t *testing.T) {
	tests := map[string]string{
		"InstanceID":          "instanceId",
		"DisplayName":         "displayName",
		"AuthenticationToken": "agentAuthenticationToken",
		"MaxTasks":            "maxTasks",
		"CPUThreshold":        "cpuThreshold",
		"MemThreshold":        "memThreshold",
		"DiskThreshold":       "diskThreshold",
		"RegistrationTokenID": "registrationTokenId",
		"CreatedAt":           "createdAt",
		"UpdatedAt":           "updatedAt",
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

func TestAgentRuntimeStatusUsesObservedConnectionSourceFields(t *testing.T) {
	modelType := reflect.TypeOf(AgentRuntimeStatus{})
	for fieldName, jsonName := range map[string]string{
		"ConnectionIP":         "connectionIp,omitempty",
		"ObservedSourceIP":     "observedSourceIp,omitempty",
		"ObservedIPGeneration": "observedIpGeneration",
	} {
		field, ok := modelType.FieldByName(fieldName)
		if !ok {
			t.Fatalf("agent runtime status model must define %s", fieldName)
		}
		if got := field.Tag.Get("json"); got != jsonName {
			t.Fatalf("%s json tag = %q, want %q", fieldName, got, jsonName)
		}
	}
	if _, ok := modelType.FieldByName("IPAddress"); ok {
		t.Fatal("agent runtime status must not retain ambiguous IPAddress persistence")
	}
}

func TestAgentModelDoesNotDefineRuntimeStatusJSONField(t *testing.T) {
	if _, ok := reflect.TypeOf(Agent{}).FieldByName("RuntimeStatusJSON"); ok {
		t.Fatalf("agent model must not define RuntimeStatusJSON")
	}
}

func TestAgentRuntimeStatusModelHealthStateUsesHealthyDefault(t *testing.T) {
	field, ok := reflect.TypeOf(AgentRuntimeStatus{}).FieldByName("HealthState")
	if !ok {
		t.Fatalf("agent runtime status model must define HealthState")
	}
	tag := field.Tag.Get("gorm")
	if !strings.Contains(tag, "default:'healthy'") {
		t.Fatalf("HealthState gorm tag must default to healthy, got %q", tag)
	}
}

func TestAgentModelTagsExposeListQueryBTreeIndexes(t *testing.T) {
	tests := map[reflect.Type]map[string][]string{
		reflect.TypeOf(Agent{}): {
			"Status":    {"index:idx_agent_status"},
			"CreatedAt": {"index:idx_agent_created_at_id", "priority:1"},
			"ID":        {"index:idx_agent_created_at_id", "priority:2"},
		},
		reflect.TypeOf(AgentRuntimeStatus{}): {
			"HealthState": {"index:idx_agent_runtime_status_health_state_agent_id", "priority:1"},
			"AgentID":     {"index:idx_agent_runtime_status_health_state_agent_id", "priority:2"},
		},
	}

	for modelType, fields := range tests {
		for fieldName, requiredParts := range fields {
			field, ok := modelType.FieldByName(fieldName)
			if !ok {
				t.Fatalf("%s must define %s", modelType.Name(), fieldName)
			}
			tag := field.Tag.Get("gorm")
			for _, required := range requiredParts {
				if !strings.Contains(tag, required) {
					t.Fatalf("%s.%s gorm tag must contain %q, got %q", modelType.Name(), fieldName, required, tag)
				}
			}
		}
	}
}
