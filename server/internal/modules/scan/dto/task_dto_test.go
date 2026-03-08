package dto

import (
	"encoding/json"
	"testing"
)

func TestTaskAssignmentJSONUsesWorkflowConfigField(t *testing.T) {
	payload, err := json.Marshal(TaskAssignment{TaskID: 1, WorkflowConfig: map[string]any{"recon": map[string]any{"enabled": true}}})
	if err != nil {
		t.Fatalf("marshal task assignment: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("unmarshal task assignment: %v", err)
	}
	if _, ok := got["workflowConfig"]; !ok {
		t.Fatalf("expected workflowConfig json key, got: %v", got)
	}
	if _, exists := got["workflowConfigYAML"]; exists {
		t.Fatalf("unexpected legacy workflowConfigYAML json key, got: %v", got)
	}
}
